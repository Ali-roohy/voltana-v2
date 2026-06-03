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
	"strings"
	"time"

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

	// Email-verification token + rate limits (API contract, TASK-0009).
	verificationTokenTTL = 24 * time.Hour
	verifyRateWindow     = 15 * time.Minute
	verifyRateMax        = int64(20)
	resendIPWindow       = time.Hour
	resendIPMax          = int64(5)
	resendEmailWindow    = time.Hour
	resendEmailMax       = int64(3)
	resendCooldown       = 60 * time.Second
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrRateLimitExceeded  = errors.New("too many login attempts")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenRevoked       = errors.New("token has been revoked")

	ErrEmailNotVerified         = errors.New("email not verified")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
)

// Mailer sends account emails. Implemented by internal/mailer; mocked in tests
// so SMTP is never reached during unit tests.
type Mailer interface {
	SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error
}

// TokenStore abstracts Redis operations needed by AuthService.
// The concrete implementation lives in repository to keep service testable.
type TokenStore interface {
	StoreRefreshToken(ctx context.Context, jti, userID string, ttl time.Duration) error
	// ConsumeRefreshToken atomically reads and deletes the token (single-use).
	ConsumeRefreshToken(ctx context.Context, jti string) (userID string, err error)
	DeleteRefreshToken(ctx context.Context, jti string) error
	CheckRateLimit(ctx context.Context, key string, max int64, window time.Duration) (allowed bool, err error)
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
}

func NewAuthService(
	users repository.UserRepository,
	verifications repository.VerificationTokenRepository,
	tokens TokenStore,
	mailer Mailer,
	appURL, jwtSecret string,
) *AuthService {
	// Generate once at startup so Login timing is uniform whether the email
	// exists or not.
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

// Register creates a new user with a bcrypt-hashed password (cost 12) and issues
// an email-verification token (emailed as a link). Email delivery is best-effort:
// registration still succeeds if the token cannot be stored or the email fails to
// send (the user can request a resend) — failures are logged, never surfaced.
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

// issueVerification mints a token, stores only its SHA-256 hash, and emails the
// link. Best-effort: any failure is logged (by user ID — never the token/email).
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

// Login verifies credentials, enforces per-IP rate limiting, and issues tokens.
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
		// Always run bcrypt to prevent email-enumeration via timing side-channel
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	// Gate ONLY after a successful credential check — a wrong password must still
	// return ErrInvalidCredentials so verification state is never leaked.
	if !user.IsEmailVerified {
		return "", "", ErrEmailNotVerified
	}

	return s.issueTokenPair(ctx, user.ID)
}

// VerifyEmail consumes a verification token (per-IP rate limited) and marks the
// account verified. Returns whether the account was already verified.
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
			// Do not distinguish unknown / expired / used — anti-enumeration.
			return false, ErrInvalidVerificationToken
		}
		return false, fmt.Errorf("consume token: %w", err)
	}
	_ = userID
	return already, nil
}

// ResendVerification re-issues a verification email for an unverified account.
// Always succeeds (HTTP 202) for a well-formed request unless a rate limit is hit;
// the work is done uniformly regardless of whether the email exists or is verified
// (anti-enumeration). Returns ErrRateLimitExceeded when the per-IP/per-email caps trip.
func (s *AuthService) ResendVerification(ctx context.Context, email, ip string) error {
	emailHash := sha256Hex(strings.ToLower(strings.TrimSpace(email)))

	// Abuse caps (429 when exceeded): per-IP protects the relay, per-email protects
	// a victim from email-bombing. Keys hash the email so raw addresses never hit Redis.
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

	// 60s cooldown between successful sends (modeled as max=1 / 60s window). Within
	// the cooldown we silently skip the send but still return 202 (anti-enumeration).
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

// Refresh validates a refresh token, atomically revokes it, and issues a new pair.
func (s *AuthService) Refresh(ctx context.Context, oldRefreshToken string) (accessToken, refreshToken string, err error) {
	claims, err := s.parseToken(oldRefreshToken, "refresh")
	if err != nil {
		return "", "", ErrInvalidToken
	}

	// Atomically consume (read + delete) the old token. Under concurrent reuse
	// of the same token, exactly one caller succeeds here; all others see it
	// already gone and are rejected — this blocks both sequential and concurrent
	// replay, and the consume happens BEFORE any new pair is issued.
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

// Logout revokes the refresh token in Redis.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.parseToken(refreshToken, "refresh")
	if err != nil {
		return ErrInvalidToken
	}
	return s.tokens.DeleteRefreshToken(ctx, claims.ID)
}

// IsAdmin reports whether the user is an administrator. It reads the flag fresh
// from the database (not from the access-token claims) so revoking admin takes
// effect immediately, within the access token's lifetime. Used by the AdminOnly
// middleware to gate the station write endpoints.
func (s *AuthService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return user.IsAdmin, nil
}

// GetUser returns the user record for the given id (the authenticated user).
// Backs GET /v1/me so the frontend can read identity + the is_admin flag, which
// is intentionally NOT carried in the access token (see IsAdmin).
func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

// ValidateAccessToken parses and validates an access JWT, returning its claims.
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

// ── private ───────────────────────────────────────────────────────────────────

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

// generateVerificationToken returns a high-entropy raw token (base64url, 256-bit)
// and its SHA-256 hex hash. Only the hash is persisted; the raw token travels
// solely in the emailed link and the verify request body.
func generateVerificationToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("read random: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, sha256Hex(raw), nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
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
