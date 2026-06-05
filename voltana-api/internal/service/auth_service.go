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
	baleSender     OTPSender
	tgSender       OTPSender
	baleBotUsername string
	tgBotUsername   string
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

// ── email/password auth (existing, unchanged) ─────────────────────────────────

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.users.Create(ctx, email, string(hash))
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
// Always returns nil (anti-enumeration) unless a rate limit is exceeded.
func (s *AuthService) RequestOTP(ctx context.Context, phone, ip string, platform Platform) error {
	// Per-IP guard (10/15m)
	if ok, err := s.tokens.CheckRateLimit(ctx, "otp:rl:ip:"+ip, otpIPRateMax, otpIPRateWindow); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	} else if !ok {
		return ErrRateLimitExceeded
	}

	normalized, err := normalizePhone(phone)
	if err != nil {
		// Anti-enumeration: bad format → still nil, just don't send.
		return nil
	}

	// Per-phone guard (3/15m)
	if ok, err := s.tokens.CheckRateLimit(ctx, "otp:rl:phone:"+normalized, otpPhoneRateMax, otpPhoneRateWindow); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	} else if !ok {
		return ErrRateLimitExceeded
	}

	// 60s cooldown — skip the send but still return nil.
	_, _ = s.tokens.CheckRateLimit(ctx, "otp:cooldown:"+normalized, 1, otpCooldownTTL)

	// Look up user + linked chat. Unknown phone or no chat → silent return.
	user, err := s.users.FindByPhone(ctx, normalized)
	if err != nil || (user.BaleChatID == nil && user.TelegramChatID == nil) {
		return nil
	}

	code, err := generateOTPCode()
	if err != nil {
		log.Printf("otp: code generation failed: %v", err)
		return nil
	}

	// B2: namespace Redis key by platform so Bale/Telegram OTPs don't collide.
	otpKey := "otp:login:" + string(platform) + ":" + normalized
	if err := s.tokens.CacheSet(ctx, otpKey, code, otpTTL); err != nil {
		log.Printf("otp: cache set failed for phone: %v", err)
		return nil
	}

	if err := s.resolveAndSendOTP(ctx, user, code, platform); err != nil {
		log.Printf("otp: send failed: %v", err)
	}
	return nil
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
// failover (legacy behaviour preserved as the default).
func (s *AuthService) resolveAndSendOTP(ctx context.Context, user *domain.User, code string, platform Platform) error {
	switch platform {
	case PlatformTelegram:
		if s.tgSender != nil && user.TelegramChatID != nil {
			return s.tgSender.Send(ctx, *user.TelegramChatID, code)
		}
		log.Printf("otp: Telegram not linked for user=%s", user.ID)
		return nil
	default: // PlatformBale (and any unrecognised value)
		if s.baleSender != nil && user.BaleChatID != nil {
			err := s.baleSender.Send(ctx, *user.BaleChatID, code)
			if err == nil {
				return nil
			}
			log.Printf("otp: Bale send failed, trying Telegram: %v", err)
		}
		if s.tgSender != nil && user.TelegramChatID != nil {
			return s.tgSender.Send(ctx, *user.TelegramChatID, code)
		}
		log.Printf("otp: no sender available for user=%s", user.ID)
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
		// already E.164
	case strings.HasPrefix(s, "98"):
		s = "+" + s
	case strings.HasPrefix(s, "0"):
		s = "+98" + s[1:]
	default:
		return "", ErrInvalidPhone
	}

	// E.164: "+" followed by 7–15 digits.
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
