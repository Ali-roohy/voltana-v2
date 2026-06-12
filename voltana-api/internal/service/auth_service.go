package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"
	"unicode"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost      = 12
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	loginRateWindow = 15 * time.Minute
	loginRateMax    = int64(10)

	// Email-verification token + rate limits (TASK-0009).
	verificationTokenTTL = 24 * time.Hour
	verifyRateWindow     = 15 * time.Minute
	verifyRateMax        = int64(20)
	resendIPWindow       = time.Hour
	resendIPMax          = int64(5)
	resendEmailWindow    = time.Hour
	resendEmailMax       = int64(3)
	resendCooldown       = 60 * time.Second

	// OTP rate limits and TTLs (TASK-0017 / TASK-0026).
	otpTTL             = 60 * time.Second // B4: 60s to match the UI countdown
	otpAttemptsTTL     = 5 * time.Minute
	otpMaxAttempts     = int64(5)
	otpPhoneRateWindow = 15 * time.Minute
	otpPhoneRateMax    = int64(3)
	otpIPRateWindow    = 15 * time.Minute
	otpIPRateMax       = int64(10)
	otpCooldownTTL     = 60 * time.Second

	// Bot-link token TTL (TASK-0017).
	botLinkTTL        = 10 * time.Minute
	botPendingLinkTTL = 10 * time.Minute

	// Registration contact TTL (TASK-0026 B5): how long the bot stores
	// a phone→chatID mapping after a cold-start /start before registration.
	regContactTTL = 30 * time.Minute
)

// Platform identifies a messaging platform for OTP delivery.
type Platform string

const (
	PlatformBale     Platform = "bale"
	PlatformTelegram Platform = "telegram"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrRateLimitExceeded  = errors.New("too many login attempts")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")

	ErrEmailNotVerified         = errors.New("email not verified")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")

	ErrOTPInvalid    = errors.New("invalid or expired OTP")
	ErrOTPLocked     = errors.New("OTP attempts exhausted — account locked")
	ErrNoBotLinked   = errors.New("no bot chat linked to this account")
	ErrInvalidPhone  = errors.New("invalid phone number")
	ErrNoBotConfig   = errors.New("no bot configured")
	ErrPhoneTaken    = errors.New("phone already registered")
	ErrEmailTaken    = errors.New("email already registered")
	ErrNoPasswordSet    = errors.New("no password set for this account — use OTP to log in")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

// OTPInvalidError wraps ErrOTPInvalid and carries the remaining-attempt count
// so the HTTP handler can surface it in the 401 body (B3, TASK-0026).
type OTPInvalidError struct {
	RemainingAttempts int64
}

func (e *OTPInvalidError) Error() string { return ErrOTPInvalid.Error() }
func (e *OTPInvalidError) Unwrap() error { return ErrOTPInvalid }

// Mailer sends account emails. Implemented by internal/mailer; mocked in tests
// so SMTP is never reached during unit tests.
type Mailer interface {
	SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error
	SendOTPEmail(ctx context.Context, toEmail, code string) error
}

// OTPSender sends a 6-digit OTP to a bot chat. Concrete implementations live
// in internal/bot (BaleSender, TelegramSender, LogOTPSender).
type OTPSender interface {
	Platform() Platform
	Send(ctx context.Context, chatID, code string) error
}

// TokenStore abstracts Redis operations needed by AuthService.
// The concrete implementation lives in repository to keep service testable.
type TokenStore interface {
	StoreRefreshToken(ctx context.Context, jti, userID string, ttl time.Duration) error
	// ConsumeRefreshToken atomically reads and deletes the token (single-use).
	ConsumeRefreshToken(ctx context.Context, jti string) (userID string, err error)
	DeleteRefreshToken(ctx context.Context, jti string) error
	CheckRateLimit(ctx context.Context, key string, max int64, window time.Duration) (allowed bool, err error)

	// Generic cache (OTP, bot-link, analytics).
	CacheGet(ctx context.Context, key string) (val string, ok bool, err error)
	CacheSet(ctx context.Context, key, val string, ttl time.Duration) error
	CacheDel(ctx context.Context, key string) error
	// CacheGetDel atomically reads and deletes a key (single-use, OTP / botlink).
	CacheGetDel(ctx context.Context, key string) (val string, ok bool, err error)
	// IncrWithTTL atomically increments a counter and sets TTL on first increment.
	IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type voltanaClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}

// OTPDeepLinkInfo is returned by RequestOTP when the delivery method is
// "deeplink" (BaleURL/TgURL set) or when a registration OTP is pending
// contact-share confirmation (AwaitingContact = true).
// NotRegistered is set when the caller is in login mode and the phone does not
// exist in the database — lets the handler surface a helpful error instead of
// a silent anti-enum 202 that leaves the user waiting for an OTP that will
// never arrive.
// A nil value means the normal anti-enum 202 path should be used.
type OTPDeepLinkInfo struct {
	BaleURL         string
	TgURL           string
	AwaitingContact bool // contact_share mode: contact not yet in Redis
	NotRegistered   bool // login mode: phone not in DB
}

const (
	// otpPendingTTL is how long a "pending" registration OTP survives while we
	// wait for the user to share their contact in the bot.
	otpPendingTTL = 10 * time.Minute
	// otpSentTTL is the TTL for the "already sent" sentinel (slightly longer than
	// the active OTP so re-polls return "otp_sent" until the code expires).
	otpSentTTL = otpTTL + 30*time.Second
	// deeplinkPendingTTL is how long a deep-link session stays in "awaiting_bot"
	// before the status poll reports it expired. Generous on purpose: the user
	// may need to install/open Bale before tapping the deep link (TASK-0036 BUG-5).
	deeplinkPendingTTL = 15 * time.Minute
)

// AuthService handles all authentication business logic.
type AuthService struct {
	users         repository.UserRepository
	verifications repository.VerificationTokenRepository
	tokens        TokenStore
	mailer        Mailer
	appURL        string
	secret        []byte
	dummyHash     []byte // pre-computed to enforce constant-time on unknown email

	// OTP / bot-link (set via SetBotSenders; optional)
	baleSender      OTPSender
	tgSender        OTPSender
	baleBotUsername string
	tgBotUsername   string

	// System settings (optional; nil → treat as contact_share / legacy).
	sysSettings repository.SystemSettingsRepository
}

func NewAuthService(
	users repository.UserRepository,
	verifications repository.VerificationTokenRepository,
	tokens TokenStore,
	mailer Mailer,
	appURL, jwtSecret string,
) *AuthService {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-timing-placeholder"), bcryptCost)
	if err != nil {
		panic("auth: failed to generate dummy hash: " + err.Error())
	}
	return &AuthService{
		users:         users,
		verifications: verifications,
		tokens:        tokens,
		mailer:        mailer,
		appURL:        appURL,
		secret:        []byte(jwtSecret),
		dummyHash:     dummy,
	}
}

// SetBotSenders wires OTP senders after construction. bale and/or tg may be
// nil (not configured). Called from main.go when bot tokens are present.
func (s *AuthService) SetBotSenders(bale, tg OTPSender, baleBotUsername, tgBotUsername string) {
	s.baleSender = bale
	s.tgSender = tg
	s.baleBotUsername = baleBotUsername
	s.tgBotUsername = tgBotUsername
}

// SetSystemSettingsRepo wires the system-settings repository. When nil (the
// default), RequestOTP uses the legacy contact_share behaviour.
func (s *AuthService) SetSystemSettingsRepo(repo repository.SystemSettingsRepository) {
	s.sysSettings = repo
}

// GetBotUsernames returns the configured bot usernames (may be empty strings when not set).
func (s *AuthService) GetBotUsernames() (bale, tg string) {
	return s.baleBotUsername, s.tgBotUsername
}

// ── email/password auth (existing, unchanged) ─────────────────────────────────

func (s *AuthService) Register(ctx context.Context, email, password, fullName, phone string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	var namePtr *string
	if trimmed := strings.TrimSpace(fullName); trimmed != "" {
		namePtr = &trimmed
	}
	var phonePtr *string
	if strings.TrimSpace(phone) != "" {
		normalized, normErr := normalizePhone(phone)
		if normErr != nil {
			return nil, ErrInvalidPhone
		}
		phonePtr = &normalized
	}
	user, err := s.users.Create(ctx, email, string(hash), namePtr, phonePtr)
	if err != nil {
		return nil, err
	}
	s.issueVerification(ctx, user)
	return user, nil
}

func (s *AuthService) issueVerification(ctx context.Context, user *domain.User) {
	raw, tokenHash, err := generateVerificationToken()
	if err != nil {
		log.Printf("verification: token generation failed for user=%s: %v", user.ID, err)
		return
	}
	if err := s.verifications.ReplaceVerificationToken(ctx, user.ID, tokenHash, time.Now().Add(verificationTokenTTL)); err != nil {
		log.Printf("verification: storing token failed for user=%s: %v", user.ID, err)
		return
	}
	verifyURL := s.appURL + "/verify-email?token=" + raw
	if err := s.mailer.SendVerificationEmail(ctx, user.Email, verifyURL); err != nil {
		log.Printf("verification: sending email failed for user=%s: %v", user.ID, err)
	}
}

func (s *AuthService) Login(ctx context.Context, email, password, ip string) (accessToken, refreshToken string, err error) {
	allowed, err := s.tokens.CheckRateLimit(ctx, "ratelimit:login:"+ip, loginRateMax, loginRateWindow)
	if err != nil {
		return "", "", fmt.Errorf("rate limit: %w", err)
	}
	if !allowed {
		return "", "", ErrRateLimitExceeded
	}

	user, lookupErr := s.users.FindByEmail(ctx, email)
	if lookupErr != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	if !user.IsEmailVerified {
		return "", "", ErrEmailNotVerified
	}

	return s.issueTokenPair(ctx, user.ID)
}

func (s *AuthService) VerifyEmail(ctx context.Context, rawToken, ip string) (alreadyVerified bool, err error) {
	allowed, err := s.tokens.CheckRateLimit(ctx, "ratelimit:verify:"+ip, verifyRateMax, verifyRateWindow)
	if err != nil {
		return false, fmt.Errorf("rate limit: %w", err)
	}
	if !allowed {
		return false, ErrRateLimitExceeded
	}

	userID, already, err := s.verifications.ConsumeVerificationToken(ctx, sha256Hex(rawToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, ErrInvalidVerificationToken
		}
		return false, fmt.Errorf("consume token: %w", err)
	}
	_ = userID
	return already, nil
}

func (s *AuthService) ResendVerification(ctx context.Context, email, ip string) error {
	emailHash := sha256Hex(strings.ToLower(strings.TrimSpace(email)))

	if ok, err := s.tokens.CheckRateLimit(ctx, "ratelimit:resend:ip:"+ip, resendIPMax, resendIPWindow); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	} else if !ok {
		return ErrRateLimitExceeded
	}
	if ok, err := s.tokens.CheckRateLimit(ctx, "ratelimit:resend:email:"+emailHash, resendEmailMax, resendEmailWindow); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	} else if !ok {
		return ErrRateLimitExceeded
	}

	fresh, err := s.tokens.CheckRateLimit(ctx, "cooldown:resend:"+emailHash, 1, resendCooldown)
	if err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}

	user, lookupErr := s.users.FindByEmail(ctx, email)
	if fresh && lookupErr == nil && !user.IsEmailVerified {
		s.issueVerification(ctx, user)
	}
	return nil
}

func (s *AuthService) Refresh(ctx context.Context, oldRefreshToken string) (accessToken, refreshToken string, err error) {
	claims, err := s.parseToken(oldRefreshToken, "refresh")
	if err != nil {
		return "", "", ErrInvalidToken
	}

	storedUID, err := s.tokens.ConsumeRefreshToken(ctx, claims.ID)
	if err != nil || storedUID != claims.Subject {
		return "", "", ErrTokenRevoked
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	return s.issueTokenPair(ctx, userID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.parseToken(refreshToken, "refresh")
	if err != nil {
		return ErrInvalidToken
	}
	return s.tokens.DeleteRefreshToken(ctx, claims.ID)
}

func (s *AuthService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return user.IsAdmin, nil
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (*domain.TokenClaims, error) {
	claims, err := s.parseToken(tokenStr, "access")
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return &domain.TokenClaims{UserID: userID, JTI: claims.ID}, nil
}

// ── OTP login (Slice B, TASK-0017) ────────────────────────────────────────────

// RequestOTP generates and delivers a 6-digit OTP to the user's linked bot
// chat. platform selects which messenger to route to (B1/B2, TASK-0026).
// Returns (nil, nil) for the normal anti-enum 202 path.
// Returns (*OTPDeepLinkInfo, nil) when the system is in "deeplink" mode and
// the user has no existing chat_id — the handler should return 200 + URL.
// Returns (nil, ErrRateLimitExceeded) when the caller is rate-limited.
func (s *AuthService) RequestOTP(ctx context.Context, phone, ip string, platform Platform, isRegister bool) (*OTPDeepLinkInfo, error) {
	// Per-IP guard (10/15m)
	if ok, err := s.tokens.CheckRateLimit(ctx, "otp:rl:ip:"+ip, otpIPRateMax, otpIPRateWindow); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	} else if !ok {
		return nil, ErrRateLimitExceeded
	}

	normalized, err := normalizePhone(phone)
	if err != nil {
		// Anti-enumeration: bad format → still nil, just don't send.
		return nil, nil
	}

	// Per-phone guard (3/15m)
	if ok, err := s.tokens.CheckRateLimit(ctx, "otp:rl:phone:"+normalized, otpPhoneRateMax, otpPhoneRateWindow); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	} else if !ok {
		return nil, ErrRateLimitExceeded
	}

	// Check delivery method — default to "contact_share" when not configured.
	deliveryMethod := "contact_share"
	if s.sysSettings != nil {
		if m, err := s.sysSettings.GetOTPDeliveryMethod(ctx); err == nil {
			deliveryMethod = m
		}
	}

	if deliveryMethod == "deeplink" {
		return s.requestOTPDeeplink(ctx, normalized, platform, isRegister)
	}

	// contact_share mode: per-phone cooldown.
	_, _ = s.tokens.CheckRateLimit(ctx, "otp:cooldown:"+normalized, 1, otpCooldownTTL)

	// Look up user + linked chat.
	user, lookupErr := s.users.FindByPhone(ctx, normalized)
	if lookupErr != nil {
		if errors.Is(lookupErr, repository.ErrNotFound) {
			// Phone not yet registered.
			// Only engage the registration pending-OTP flow when the caller explicitly
			// indicated registration intent. For login attempts with an unknown phone
			// return NotRegistered so the frontend can show a helpful error instead of
			// silently starting a countdown for an OTP that will never arrive.
			if !isRegister {
				return &OTPDeepLinkInfo{NotRegistered: true}, nil
			}
			// If the bot contact is already in Redis (user pre-shared), send OTP
			// immediately (old fast path). Otherwise store a pending OTP and let
			// the frontend poll /auth/otp/contact-status.
			contactKey := "reg:contact:" + string(platform) + ":" + normalized
			if _, contactFound, _ := s.tokens.CacheGet(ctx, contactKey); contactFound {
				s.sendRegContactOTP(ctx, normalized, platform)
				return nil, nil
			}
			// No contact yet — generate pending OTP for polling flow.
			if err := s.storeRegPendingOTP(ctx, normalized, platform); err != nil {
				log.Printf("otp: pending store failed: %v", err)
			}
			return &OTPDeepLinkInfo{AwaitingContact: true}, nil
		}
		return nil, nil
	}
	if user.BaleChatID == nil && user.TelegramChatID == nil {
		return nil, nil
	}

	code, err := generateOTPCode()
	if err != nil {
		log.Printf("otp: code generation failed: %v", err)
		return nil, nil
	}

	// B2: namespace Redis key by platform so Bale/Telegram OTPs don't collide.
	otpKey := "otp:login:" + string(platform) + ":" + normalized
	if err := s.tokens.CacheSet(ctx, otpKey, code, otpTTL); err != nil {
		log.Printf("otp: cache set failed for phone: %v", err)
		return nil, nil
	}

	if err := s.resolveAndSendOTP(ctx, user, code, platform); err != nil {
		log.Printf("otp: send failed: %v", err)
	}
	return nil, nil
}

// requestOTPDeeplink handles the "deeplink" delivery mode. When the user has no
// chat_id for the requested platform, it returns a deep-link URL instead of
// sending an OTP (the OTP is sent later by HandleDeepLinkOTP when the bot
// receives /start phone_<E164>). When the user already has a chat_id, it sends
// the OTP directly and returns (nil, nil) → handler uses 202.
func (s *AuthService) requestOTPDeeplink(ctx context.Context, normalized string, platform Platform, isRegister bool) (*OTPDeepLinkInfo, error) {
	user, err := s.users.FindByPhone(ctx, normalized)
	hasChatID := false
	if err == nil {
		switch platform {
		case PlatformTelegram:
			hasChatID = user.TelegramChatID != nil
		default:
			hasChatID = user.BaleChatID != nil
		}
	} else if errors.Is(err, repository.ErrNotFound) && !isRegister {
		// Login attempt with an unregistered phone — signal the frontend so it
		// can show a helpful error instead of starting a countdown that never ends.
		return &OTPDeepLinkInfo{NotRegistered: true}, nil
	}

	if hasChatID {
		// Already linked — send OTP directly (existing path).
		code, codeErr := generateOTPCode()
		if codeErr != nil {
			log.Printf("otp: deeplink: code gen failed: %v", codeErr)
			return nil, nil
		}
		otpKey := "otp:login:" + string(platform) + ":" + normalized
		if setErr := s.tokens.CacheSet(ctx, otpKey, code, otpTTL); setErr != nil {
			log.Printf("otp: deeplink: cache set failed: %v", setErr)
			return nil, nil
		}
		if sendErr := s.resolveAndSendOTP(ctx, user, code, platform); sendErr != nil {
			log.Printf("otp: deeplink: send failed: %v", sendErr)
		}
		return nil, nil
	}

	// No chat_id — return deep-link URL so the frontend can open the bot.
	// Mark the session as awaiting the bot so the status poll can distinguish
	// "user hasn't tapped the deep link yet" from a genuinely expired session
	// (TASK-0036 BUG-5 — without this, the first poll returned "expired").
	dlKey := "otp:dlpending:" + string(platform) + ":" + normalized
	if setErr := s.tokens.CacheSet(ctx, dlKey, "1", deeplinkPendingTTL); setErr != nil {
		log.Printf("otp: deeplink: pending marker set failed: %v", setErr)
	}
	info := &OTPDeepLinkInfo{}
	if s.baleBotUsername != "" {
		info.BaleURL = "https://ble.ir/" + s.baleBotUsername + "?start=phone_" + normalized
	}
	if s.tgBotUsername != "" {
		info.TgURL = "https://t.me/" + s.tgBotUsername + "?start=phone_" + normalized
	}
	return info, nil
}

// GetOTPDeliveryMethod returns the current system-level OTP delivery method.
// Falls back to "contact_share" when the system settings repo is not configured.
func (s *AuthService) GetOTPDeliveryMethod(ctx context.Context) string {
	if s.sysSettings == nil {
		return "contact_share"
	}
	m, err := s.sysSettings.GetOTPDeliveryMethod(ctx)
	if err != nil {
		return "contact_share"
	}
	return m
}

// sendRegContactOTP checks Redis for a registration contact stored by the bot
// cold-start flow (B5) and, if found, sends a registration OTP to that chat.
func (s *AuthService) sendRegContactOTP(ctx context.Context, normalized string, platform Platform) {
	contactKey := "reg:contact:" + string(platform) + ":" + normalized
	chatID, found, err := s.tokens.CacheGet(ctx, contactKey)
	if err != nil || !found {
		return
	}

	code, err := generateOTPCode()
	if err != nil {
		log.Printf("otp: reg code generation failed: %v", err)
		return
	}

	otpKey := "otp:reg:" + string(platform) + ":" + normalized
	if err := s.tokens.CacheSet(ctx, otpKey, code, otpTTL); err != nil {
		log.Printf("otp: reg cache set failed: %v", err)
		return
	}

	sender := s.senderForPlatform(platform)
	if sender == nil {
		log.Printf("otp: no sender configured for platform %s (reg)", platform)
		return
	}
	if err := sender.Send(ctx, chatID, code); err != nil {
		log.Printf("otp: reg send failed: %v", err)
	}
}

// storeRegPendingOTP generates an OTP code and stores it in the "pending" slot
// used by the contact-share polling flow. The OTP is not sent yet.
func (s *AuthService) storeRegPendingOTP(ctx context.Context, normalized string, platform Platform) error {
	code, err := generateOTPCode()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	key := "otp:pending:reg:" + string(platform) + ":" + normalized
	return s.tokens.CacheSet(ctx, key, code, otpPendingTTL)
}

// CheckContactShareStatus is polled by the frontend after RequestOTP returns
// {status:"awaiting_contact_share"}. Returns:
//   - "awaiting_contact_share" — user hasn't shared contact in the bot yet
//   - "otp_sent"               — OTP just dispatched (or already dispatched); timer can start
//   - "expired"                — pending OTP TTL elapsed; user must restart
func (s *AuthService) CheckContactShareStatus(ctx context.Context, phone string, platform Platform) (string, error) {
	normalized, err := normalizePhone(phone)
	if err != nil {
		return "expired", nil
	}

	sentKey := "otp:sent:reg:" + string(platform) + ":" + normalized
	pendingKey := "otp:pending:reg:" + string(platform) + ":" + normalized
	contactKey := "reg:contact:" + string(platform) + ":" + normalized

	// Idempotency: already dispatched on a previous poll (contact_share path).
	if _, sent, _ := s.tokens.CacheGet(ctx, sentKey); sent {
		return "otp_sent", nil
	}

	// Deep-link path: HandleDeepLinkOTP stores the active OTP directly under
	// otp:reg: or otp:login: without going through the pending flow. Detect
	// these so the frontend can start the 60 s timer as soon as the bot sends
	// the code (instead of immediately when the deep-link URL is displayed).
	for _, prefix := range []string{"otp:reg:", "otp:login:"} {
		if _, ok, _ := s.tokens.CacheGet(ctx, prefix+string(platform)+":"+normalized); ok {
			return "otp_sent", nil
		}
	}

	// Deep-link session where the user hasn't opened the bot yet (TASK-0036
	// BUG-5): keep the frontend in its neutral "open the bot" state instead of
	// reporting expiry. The marker's 15-minute TTL is the real timeout.
	if _, awaitingBot, _ := s.tokens.CacheGet(ctx, "otp:dlpending:"+string(platform)+":"+normalized); awaitingBot {
		return "awaiting_bot", nil
	}

	// No pending OTP → session expired (contact_share flow only; deep_link
	// sessions are covered by the dlpending marker above, so reaching here
	// with neither key means the session is genuinely over).
	if _, hasPending, _ := s.tokens.CacheGet(ctx, pendingKey); !hasPending {
		return "expired", nil
	}

	// Contact not yet shared in the bot.
	chatID, hasContact, _ := s.tokens.CacheGet(ctx, contactKey)
	if !hasContact {
		return "awaiting_contact_share", nil
	}

	// Atomically claim the pending code (CacheGetDel prevents double-dispatch).
	code, claimed, _ := s.tokens.CacheGetDel(ctx, pendingKey)
	if !claimed {
		// Another concurrent poll already handled it.
		return "otp_sent", nil
	}

	// Promote to active OTP (60 s window — same as deeplink / login flow).
	activeKey := "otp:reg:" + string(platform) + ":" + normalized
	if setErr := s.tokens.CacheSet(ctx, activeKey, code, otpTTL); setErr != nil {
		log.Printf("otp: contact-status: cache set failed: %v", setErr)
		return "expired", nil
	}

	// Sent sentinel — keeps idempotent re-polls returning "otp_sent".
	_ = s.tokens.CacheSet(ctx, sentKey, "1", otpSentTTL)

	// Dispatch.
	sender := s.senderForPlatform(platform)
	if sender == nil {
		log.Printf("otp: contact-status: no sender for platform %s", platform)
	} else if sendErr := sender.Send(ctx, chatID, code); sendErr != nil {
		log.Printf("otp: contact-status: send failed: %v", sendErr)
	}

	return "otp_sent", nil
}

// senderForPlatform returns the OTPSender for the given platform (may be nil).
func (s *AuthService) senderForPlatform(platform Platform) OTPSender {
	switch platform {
	case PlatformTelegram:
		return s.tgSender
	default:
		return s.baleSender
	}
}

// CompleteOTPLogin validates the OTP and issues a token pair. Returns
// *OTPInvalidError for wrong/expired codes or ErrOTPLocked after exhaustion
// (B2/B3, TASK-0026). Platform is used to namespace the Redis key.
func (s *AuthService) CompleteOTPLogin(ctx context.Context, phone, code, ip string, platform Platform) (accessToken, refreshToken string, err error) {
	normalized, normErr := normalizePhone(phone)
	if normErr != nil {
		return "", "", &OTPInvalidError{RemainingAttempts: otpMaxAttempts}
	}

	// Check attempt lockout BEFORE consuming the OTP.
	attempts, _, _ := s.tokens.CacheGet(ctx, "otp:attempts:"+normalized)
	currentAttempts := countStr(attempts)
	if currentAttempts >= otpMaxAttempts {
		return "", "", ErrOTPLocked
	}
	remaining := otpMaxAttempts - currentAttempts

	// Single-use consume: GetDel returns ("", false) when the key is absent (expired).
	otpKey := "otp:login:" + string(platform) + ":" + normalized
	stored, ok, err := s.tokens.CacheGetDel(ctx, otpKey)
	if err != nil {
		return "", "", fmt.Errorf("otp: cache: %w", err)
	}
	if !ok || stored == "" {
		return "", "", &OTPInvalidError{RemainingAttempts: remaining}
	}

	// Constant-time comparison to prevent timing-based enumeration.
	if !constantTimeEqual(stored, code) {
		cnt, _ := s.tokens.IncrWithTTL(ctx, "otp:attempts:"+normalized, otpAttemptsTTL)
		if cnt >= otpMaxAttempts {
			return "", "", ErrOTPLocked
		}
		return "", "", &OTPInvalidError{RemainingAttempts: otpMaxAttempts - cnt}
	}

	// Success: clear attempt counter.
	_ = s.tokens.CacheDel(ctx, "otp:attempts:"+normalized)

	user, err := s.users.FindByPhone(ctx, normalized)
	if err != nil {
		return "", "", &OTPInvalidError{RemainingAttempts: remaining}
	}

	return s.issueTokenPair(ctx, user.ID)
}

// ── Bot-link flow (Slice A, TASK-0017) ────────────────────────────────────────

// InitiateBotLink mints a link token, stores it in Redis (10 min), and returns
// the deep links for whichever platforms are configured. At least one URL is
// returned; both are returned if both senders are configured.
func (s *AuthService) InitiateBotLink(ctx context.Context, userID uuid.UUID) (baleURL, telegramURL string, err error) {
	// A URL can only be generated when the bot username is set — without it we
	// can't build the deep link even if a sender is wired up.
	if s.baleBotUsername == "" && s.tgBotUsername == "" {
		return "", "", ErrNoBotConfig
	}

	token, err := generateSecureToken()
	if err != nil {
		return "", "", fmt.Errorf("generate link token: %w", err)
	}

	if err := s.tokens.CacheSet(ctx, "botlink:"+token, userID.String(), botLinkTTL); err != nil {
		return "", "", fmt.Errorf("store link token: %w", err)
	}

	if s.baleSender != nil && s.baleBotUsername != "" {
		baleURL = "https://ble.ir/" + s.baleBotUsername + "?start=" + token
	}
	if s.tgSender != nil && s.tgBotUsername != "" {
		telegramURL = "https://t.me/" + s.tgBotUsername + "?start=" + token
	}
	return baleURL, telegramURL, nil
}

// ConsumeBotLinkToken is called by the Poller on a /start <token> message.
// Single-use (GetDel): returns the owning userID and whether the token existed.
func (s *AuthService) ConsumeBotLinkToken(ctx context.Context, token string) (userID string, found bool, err error) {
	return s.tokens.CacheGetDel(ctx, "botlink:"+token)
}

// StorePendingLink saves intermediate state while the user has opened the bot
// but hasn't yet shared their contact.
func (s *AuthService) StorePendingLink(ctx context.Context, platform, chatID, userID string) error {
	return s.tokens.CacheSet(ctx, "botlink:pending:"+platform+":"+chatID, userID, botPendingLinkTTL)
}

// ConsumePendingLink retrieves and removes the pending link state.
func (s *AuthService) ConsumePendingLink(ctx context.Context, platform, chatID string) (userID string, found bool, err error) {
	return s.tokens.CacheGetDel(ctx, "botlink:pending:"+platform+":"+chatID)
}

// CompleteBotLink is called by the Poller after a verified contact share. It
// normalizes the phone and writes phone + chat_id to the user row.
func (s *AuthService) CompleteBotLink(ctx context.Context, userIDStr, platform, chatID, phone string) error {
	normalized, err := normalizePhone(phone)
	if err != nil {
		return fmt.Errorf("normalize phone %q: %w", phone, err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("parse user id: %w", err)
	}

	var baleID, tgID *string
	switch Platform(platform) {
	case PlatformBale:
		baleID = &chatID
	case PlatformTelegram:
		tgID = &chatID
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}

	return s.users.UpdateBotLink(ctx, userID, normalized, baleID, tgID)
}

// StoreRegistrationContact is called by the bot Poller when a user shares their
// phone after a bare /start (no link token). Stores phone→chatID in Redis so
// that a subsequent POST /auth/otp/register can create the account (B5).
func (s *AuthService) StoreRegistrationContact(ctx context.Context, platform, chatID, phone string) error {
	normalized, err := normalizePhone(phone)
	if err != nil {
		return fmt.Errorf("normalize phone: %w", err)
	}
	return s.tokens.CacheSet(ctx, "reg:contact:"+platform+":"+normalized, chatID, regContactTTL)
}

// CompleteOTPRegister validates a registration OTP and creates a new user
// identified by phone + bot chat_id. email is optional (B6, TASK-0026).
func (s *AuthService) CompleteOTPRegister(ctx context.Context, phone, code, ip string, platform Platform, email *string) (accessToken, refreshToken string, err error) {
	normalized, normErr := normalizePhone(phone)
	if normErr != nil {
		return "", "", &OTPInvalidError{RemainingAttempts: otpMaxAttempts}
	}

	// Check attempt lockout BEFORE consuming the OTP.
	attempts, _, _ := s.tokens.CacheGet(ctx, "otp:attempts:"+normalized)
	currentAttempts := countStr(attempts)
	if currentAttempts >= otpMaxAttempts {
		return "", "", ErrOTPLocked
	}
	remaining := otpMaxAttempts - currentAttempts

	// Single-use consume from the registration OTP key.
	otpKey := "otp:reg:" + string(platform) + ":" + normalized
	stored, ok, cacheErr := s.tokens.CacheGetDel(ctx, otpKey)
	if cacheErr != nil {
		return "", "", fmt.Errorf("otp: cache: %w", cacheErr)
	}
	if !ok || stored == "" {
		return "", "", &OTPInvalidError{RemainingAttempts: remaining}
	}

	// Constant-time comparison to prevent timing-based enumeration.
	if !constantTimeEqual(stored, code) {
		cnt, _ := s.tokens.IncrWithTTL(ctx, "otp:attempts:"+normalized, otpAttemptsTTL)
		if cnt >= otpMaxAttempts {
			return "", "", ErrOTPLocked
		}
		return "", "", &OTPInvalidError{RemainingAttempts: otpMaxAttempts - cnt}
	}

	// OTP valid — clear attempt counter.
	_ = s.tokens.CacheDel(ctx, "otp:attempts:"+normalized)

	// Phone already registered → conflict.
	if _, lookupErr := s.users.FindByPhone(ctx, normalized); lookupErr == nil {
		return "", "", ErrPhoneTaken
	}

	// Email uniqueness pre-check — avoids a DB constraint 500 if the email is
	// already registered via a different account.
	if email != nil && *email != "" {
		if _, lookupErr := s.users.FindByEmail(ctx, *email); lookupErr == nil {
			return "", "", ErrEmailTaken
		}
	}

	// Look up the chatID stored by the bot cold-start (B5).
	// Anti-enum: if no contact stored → consume OTP, return INVALID_OTP (no hint).
	contactKey := "reg:contact:" + string(platform) + ":" + normalized
	chatID, found, redisErr := s.tokens.CacheGetDel(ctx, contactKey)
	if redisErr != nil || !found {
		return "", "", &OTPInvalidError{RemainingAttempts: remaining}
	}

	var baleChatID, tgChatID *string
	switch platform {
	case PlatformBale:
		baleChatID = &chatID
	default:
		tgChatID = &chatID
	}

	user, createErr := s.users.CreateWithPhone(ctx, normalized, email, baleChatID, tgChatID)
	if createErr != nil {
		if errors.Is(createErr, repository.ErrPhoneTaken) {
			return "", "", ErrPhoneTaken
		}
		if errors.Is(createErr, repository.ErrEmailTaken) {
			return "", "", ErrEmailTaken
		}
		return "", "", fmt.Errorf("otp: create user: %w", createErr)
	}

	return s.issueTokenPair(ctx, user.ID)
}

// ── phone + password login (TASK-0029 A1) ────────────────────────────────────

// LoginWithPhone authenticates a phone user by password. Returns the same error
// for "not found" and "wrong password" to prevent enumeration.
// Returns ErrNoPasswordSet (400) when the account has never set a password.
func (s *AuthService) LoginWithPhone(ctx context.Context, phone, password, ip string) (accessToken, refreshToken string, err error) {
	allowed, err := s.tokens.CheckRateLimit(ctx, "ratelimit:login:"+ip, loginRateMax, loginRateWindow)
	if err != nil {
		return "", "", fmt.Errorf("rate limit: %w", err)
	}
	if !allowed {
		return "", "", ErrRateLimitExceeded
	}

	normalized, normErr := normalizePhone(phone)
	if normErr != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", "", ErrInvalidCredentials
	}

	user, lookupErr := s.users.FindByPhone(ctx, normalized)
	if lookupErr != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", "", ErrInvalidCredentials
	}

	if user.PasswordHash == "" {
		return "", "", ErrNoPasswordSet
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, user.ID)
}

// ── optional password setup (TASK-0029 A3) ───────────────────────────────────

// SetPassword hashes and stores a new password for the given user. Idempotent
// (a second call replaces the existing hash). No current-password check because
// first-time set is always from a passwordless phone account.
func (s *AuthService) SetPassword(ctx context.Context, userID uuid.UUID, password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.users.SetPasswordHash(ctx, userID, string(hash))
}

// ── deep-link OTP handler (TASK-0029 B5) ─────────────────────────────────────

// HandleDeepLinkOTP is called by the bot Poller when it receives
// "/start phone_<E164>" in deeplink mode. It generates and sends an OTP
// directly to the chatID that sent the message.
//
// If a valid OTP already exists in Redis for this phone+platform, no new code
// is generated (dedup).
//
// For an unregistered phone, a registration OTP is stored and
// StoreRegistrationContact is called so CompleteOTPRegister can link the chat.
// For a registered phone, a login OTP is stored (and the chat_id is linked if
// the user didn't have one yet).
func (s *AuthService) HandleDeepLinkOTP(ctx context.Context, platform, chatID, rawPhone string) error {
	normalized, err := normalizePhone(rawPhone)
	if err != nil {
		log.Printf("otp: deeplink: invalid phone %q: %v", rawPhone, err)
		return nil
	}

	sender := s.senderForPlatform(Platform(platform))
	if sender == nil {
		log.Printf("otp: deeplink: no sender configured for platform %s", platform)
		return nil
	}

	// The /start receipt IS the bot interaction — end the "awaiting_bot" state
	// (the active OTP key set below takes over for the status poll).
	_, _, _ = s.tokens.CacheGetDel(ctx, "otp:dlpending:"+platform+":"+normalized)

	user, lookupErr := s.users.FindByPhone(ctx, normalized)
	if lookupErr != nil {
		// Unregistered phone — registration OTP flow.
		regKey := "otp:reg:" + platform + ":" + normalized
		if existing, ok, _ := s.tokens.CacheGet(ctx, regKey); ok && existing != "" {
			return nil // already pending
		}
		// Store chatID as a registration contact for CompleteOTPRegister.
		if storeErr := s.StoreRegistrationContact(ctx, platform, chatID, normalized); storeErr != nil {
			log.Printf("otp: deeplink: store reg contact: %v", storeErr)
		}
		code, genErr := generateOTPCode()
		if genErr != nil {
			return fmt.Errorf("generate otp: %w", genErr)
		}
		if setErr := s.tokens.CacheSet(ctx, regKey, code, otpTTL); setErr != nil {
			return fmt.Errorf("cache set: %w", setErr)
		}
		if sendErr := sender.Send(ctx, chatID, code); sendErr != nil {
			log.Printf("otp: deeplink: send failed: %v", sendErr)
		}
		return nil
	}

	// Registered user — login OTP flow.
	loginKey := "otp:login:" + platform + ":" + normalized
	if existing, ok, _ := s.tokens.CacheGet(ctx, loginKey); ok && existing != "" {
		return nil // already pending
	}

	// Link bot chat_id if the user doesn't have one yet.
	switch Platform(platform) {
	case PlatformTelegram:
		if user.TelegramChatID == nil {
			if linkErr := s.users.UpdateBotLink(ctx, user.ID, normalized, nil, &chatID); linkErr != nil {
				log.Printf("otp: deeplink: update bot link: %v", linkErr)
			}
		}
	default:
		if user.BaleChatID == nil {
			if linkErr := s.users.UpdateBotLink(ctx, user.ID, normalized, &chatID, nil); linkErr != nil {
				log.Printf("otp: deeplink: update bot link: %v", linkErr)
			}
		}
	}

	code, genErr := generateOTPCode()
	if genErr != nil {
		return fmt.Errorf("generate otp: %w", genErr)
	}
	if setErr := s.tokens.CacheSet(ctx, loginKey, code, otpTTL); setErr != nil {
		return fmt.Errorf("cache set: %w", setErr)
	}
	if sendErr := sender.Send(ctx, chatID, code); sendErr != nil {
		log.Printf("otp: deeplink: send failed: %v", sendErr)
	}
	return nil
}

// ── private helpers ───────────────────────────────────────────────────────────

func (s *AuthService) issueTokenPair(ctx context.Context, userID uuid.UUID) (string, string, error) {
	now := time.Now()
	sub := userID.String()

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, voltanaClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
		Type: "access",
	}).SignedString(s.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refreshJTI := uuid.NewString()
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, voltanaClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			ID:        refreshJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
		},
		Type: "refresh",
	}).SignedString(s.secret)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	if err := s.tokens.StoreRefreshToken(ctx, refreshJTI, sub, refreshTokenTTL); err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// resolveAndSendOTP routes the OTP to the requested platform (B1, TASK-0026).
// Telegram goes directly to Telegram only; Bale uses Bale-first with Telegram
// Each platform is routed strictly — no cross-platform fallback so the user
// always receives the OTP on the messenger they selected.
func (s *AuthService) resolveAndSendOTP(ctx context.Context, user *domain.User, code string, platform Platform) error {
	switch platform {
	case PlatformTelegram:
		if user.TelegramChatID == nil {
			log.Printf("otp: telegram not linked for user=%s", user.ID)
			return nil
		}
		if s.tgSender == nil {
			log.Printf("otp: telegram sender not configured")
			return nil
		}
		if err := s.tgSender.Send(ctx, *user.TelegramChatID, code); err != nil {
			log.Printf("otp: telegram send failed for user=%s: %v", user.ID, err)
		}
		return nil
	default: // PlatformBale
		if user.BaleChatID == nil {
			log.Printf("otp: bale not linked for user=%s", user.ID)
			return nil
		}
		if s.baleSender == nil {
			log.Printf("otp: bale sender not configured")
			return nil
		}
		if err := s.baleSender.Send(ctx, *user.BaleChatID, code); err != nil {
			log.Printf("otp: bale send failed for user=%s: %v", user.ID, err)
		}
		return nil
	}
}

// generateVerificationToken returns a high-entropy raw token (base64url, 256-bit)
// and its SHA-256 hex hash.
func generateVerificationToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("read random: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, sha256Hex(raw), nil
}

// generateSecureToken returns a URL-safe 256-bit random string.
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateOTPCode returns a cryptographically-random 6-digit string (000000–999999).
func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// normalizePhone accepts Iranian phone numbers in several formats and returns
// the canonical E.164 form (e.g. "+989121234567").
func normalizePhone(raw string) (string, error) {
	var b strings.Builder
	for i, c := range raw {
		if c == '+' && i == 0 {
			b.WriteRune(c)
		} else if unicode.IsDigit(c) {
			b.WriteRune(c)
		}
	}
	s := b.String()

	switch {
	case strings.HasPrefix(s, "+"):
		// Already E.164 — keep as-is.
	case strings.HasPrefix(s, "00"):
		// International exit-code prefix (00<cc><number>) → +<cc><number>.
		s = "+" + s[2:]
	case strings.HasPrefix(s, "98"):
		// Iran without leading +.
		s = "+" + s
	case strings.HasPrefix(s, "0") && len(s) == 11:
		// Standard Iranian mobile: 0 + 10 digits → +98 + 10 digits.
		s = "+98" + s[1:]
	case strings.HasPrefix(s, "0"):
		// Leading 0 that is NOT the Iranian 11-digit format.
		// Treat 0 as a local international-access digit and strip it.
		// e.g. 0971522890098 (UAE) → +971522890098.
		s = "+" + s[1:]
	default:
		// Bare digits with country code but no prefix symbol.
		// e.g. 971522890098 (UAE) → +971522890098.
		s = "+" + s
	}

	// E.164: "+" followed by 7–15 digits (total 8–16 chars).
	if len(s) < 8 || len(s) > 16 {
		return "", ErrInvalidPhone
	}
	return s, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// constantTimeEqual compares two strings in constant time to avoid
// timing side-channels on OTP comparisons.
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var x byte
	for i := 0; i < len(a); i++ {
		x |= a[i] ^ b[i]
	}
	return x == 0
}

// countStr parses a Redis integer string; returns 0 on error.
func countStr(s string) int64 {
	if s == "" {
		return 0
	}
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func (s *AuthService) parseToken(tokenStr, expectedType string) (*voltanaClaims, error) {
	claims := &voltanaClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Type != expectedType {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// TestOTPDelivery sends a fixed test code ("000000") to the admin's own linked
// channel for the given platform. No Redis key is written — this is fire-and-forget.
// Returns a human-readable message on success, or a descriptive error.
func (s *AuthService) TestOTPDelivery(ctx context.Context, userID uuid.UUID, platform string) (string, error) {
	const testCode = "000000"

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("load user: %w", err)
	}

	switch platform {
	case "bale":
		if user.BaleChatID == nil {
			return "", fmt.Errorf("bale not linked")
		}
		sender := s.senderForPlatform(PlatformBale)
		if sender == nil {
			return "", fmt.Errorf("bale sender not configured")
		}
		if err := sender.Send(ctx, *user.BaleChatID, testCode); err != nil {
			return "", err
		}
		return "OTP sent via bale", nil

	case "telegram":
		if user.TelegramChatID == nil {
			return "", fmt.Errorf("telegram not linked")
		}
		sender := s.senderForPlatform(PlatformTelegram)
		if sender == nil {
			return "", fmt.Errorf("telegram sender not configured")
		}
		if err := sender.Send(ctx, *user.TelegramChatID, testCode); err != nil {
			return "", err
		}
		return "OTP sent via telegram", nil

	case "email":
		if user.Email == "" {
			return "", fmt.Errorf("email not set")
		}
		if err := s.mailer.SendOTPEmail(ctx, user.Email, testCode); err != nil {
			return "", err
		}
		return "OTP sent via email", nil

	default:
		return "", fmt.Errorf("invalid platform: must be bale, telegram, or email")
	}
}
