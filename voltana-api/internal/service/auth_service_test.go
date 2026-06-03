package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/google/uuid"
)

// ── in-memory mocks ───────────────────────────────────────────────────────────

type mockUserRepo struct {
	byEmail map[string]*domain.User
	byID    map[uuid.UUID]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[uuid.UUID]*domain.User),
	}
}

func (m *mockUserRepo) Create(_ context.Context, email, passwordHash string) (*domain.User, error) {
	if _, exists := m.byEmail[email]; exists {
		return nil, repository.ErrEmailTaken
	}
	u := &domain.User{ID: uuid.New(), Email: email, PasswordHash: passwordHash}
	m.byEmail[email] = u
	m.byID[u.ID] = u
	return u, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

type mockTokenStore struct {
	tokens map[string]string
	counts map[string]int64
}

func newMockTokenStore() *mockTokenStore {
	return &mockTokenStore{
		tokens: make(map[string]string),
		counts: make(map[string]int64),
	}
}

func (m *mockTokenStore) StoreRefreshToken(_ context.Context, jti, userID string, _ time.Duration) error {
	m.tokens[jti] = userID
	return nil
}

func (m *mockTokenStore) ConsumeRefreshToken(_ context.Context, jti string) (string, error) {
	uid, ok := m.tokens[jti]
	if !ok {
		return "", errors.New("not found")
	}
	delete(m.tokens, jti)
	return uid, nil
}

func (m *mockTokenStore) DeleteRefreshToken(_ context.Context, jti string) error {
	delete(m.tokens, jti)
	return nil
}

func (m *mockTokenStore) CheckRateLimit(_ context.Context, key string, max int64, _ time.Duration) (bool, error) {
	m.counts[key]++
	return m.counts[key] <= max, nil
}

func (m *mockUserRepo) markVerified(email string) {
	if u, ok := m.byEmail[email]; ok {
		u.IsEmailVerified = true
	}
}

// mockVerifRepo mirrors the real repo's atomic consume+verify by holding a
// reference to the user repo (so ConsumeVerificationToken can flip the user).
type mockVerifRepo struct {
	users  *mockUserRepo
	byHash map[string]verifEntry
}

type verifEntry struct {
	userID    uuid.UUID
	expiresAt time.Time
}

func newMockVerifRepo(users *mockUserRepo) *mockVerifRepo {
	return &mockVerifRepo{users: users, byHash: make(map[string]verifEntry)}
}

func (m *mockVerifRepo) ReplaceVerificationToken(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	for h, e := range m.byHash {
		if e.userID == userID {
			delete(m.byHash, h)
		}
	}
	m.byHash[tokenHash] = verifEntry{userID: userID, expiresAt: expiresAt}
	return nil
}

func (m *mockVerifRepo) ConsumeVerificationToken(_ context.Context, tokenHash string) (uuid.UUID, bool, error) {
	e, ok := m.byHash[tokenHash]
	if !ok || time.Now().After(e.expiresAt) {
		return uuid.Nil, false, repository.ErrNotFound
	}
	for h, ee := range m.byHash {
		if ee.userID == e.userID {
			delete(m.byHash, h)
		}
	}
	already := false
	if u, ok := m.users.byID[e.userID]; ok {
		already = u.IsEmailVerified
		u.IsEmailVerified = true
	}
	return e.userID, already, nil
}

type mockMailer struct {
	sent []sentEmail
}

type sentEmail struct {
	to        string
	verifyURL string
}

func (m *mockMailer) SendVerificationEmail(_ context.Context, toEmail, verifyURL string) error {
	m.sent = append(m.sent, sentEmail{to: toEmail, verifyURL: verifyURL})
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

const testSecret = "test-jwt-secret-key-at-least-32-chars-long"

func newTestService() (*service.AuthService, *mockUserRepo, *mockTokenStore) {
	svc, users, store, _, _ := newTestServiceFull()
	return svc, users, store
}

func newTestServiceFull() (*service.AuthService, *mockUserRepo, *mockTokenStore, *mockMailer, *mockVerifRepo) {
	users := newMockUserRepo()
	store := newMockTokenStore()
	mail := &mockMailer{}
	verif := newMockVerifRepo(users)
	svc := service.NewAuthService(users, verif, store, mail, "https://app.test", testSecret)
	return svc, users, store, mail, verif
}

func mustRegister(t *testing.T, svc *service.AuthService, email, password string) *domain.User {
	t.Helper()
	u, err := svc.Register(context.Background(), email, password)
	if err != nil {
		t.Fatalf("Register(%q): %v", email, err)
	}
	return u
}

// mustRegisterVerified registers then marks the account verified, for tests that
// exercise the post-verification login path.
func mustRegisterVerified(t *testing.T, svc *service.AuthService, users *mockUserRepo, email, password string) *domain.User {
	t.Helper()
	u := mustRegister(t, svc, email, password)
	users.markVerified(email)
	return u
}

// tokenFromURL extracts the raw token from a "...token=<raw>" verification URL.
func tokenFromURL(t *testing.T, url string) string {
	t.Helper()
	const marker = "token="
	i := strings.Index(url, marker)
	if i < 0 {
		t.Fatalf("no token in URL %q", url)
	}
	return url[i+len(marker):]
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc, _, _ := newTestService()
	u := mustRegister(t, svc, "alice@example.com", "password123")

	if u.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", u.Email)
	}
	if u.PasswordHash == "password123" {
		t.Error("password must be hashed, not stored in plain text")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _, _ := newTestService()
	mustRegister(t, svc, "alice@example.com", "password123")

	_, err := svc.Register(context.Background(), "alice@example.com", "other")
	if !errors.Is(err, repository.ErrEmailTaken) {
		t.Errorf("want ErrEmailTaken, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, users, _ := newTestService()
	mustRegisterVerified(t, svc, users, "bob@example.com", "s3cret!")

	access, refresh, err := svc.Login(context.Background(), "bob@example.com", "s3cret!", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if access == "" || refresh == "" {
		t.Error("expected non-empty access and refresh tokens")
	}
	if access == refresh {
		t.Error("access and refresh tokens must differ")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _, _ := newTestService()
	mustRegister(t, svc, "bob@example.com", "s3cret!")

	_, _, err := svc.Login(context.Background(), "bob@example.com", "wrong", "127.0.0.1")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc, _, _ := newTestService()

	_, _, err := svc.Login(context.Background(), "nobody@example.com", "pass", "127.0.0.1")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestRefresh_RotatesToken(t *testing.T) {
	svc, users, _ := newTestService()
	mustRegisterVerified(t, svc, users, "carol@example.com", "password")

	_, refresh, err := svc.Login(context.Background(), "carol@example.com", "password", "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}

	newAccess, newRefresh, err := svc.Refresh(context.Background(), refresh)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("expected non-empty tokens after refresh")
	}
}

func TestRefresh_PreventsReplay(t *testing.T) {
	svc, users, _ := newTestService()
	mustRegisterVerified(t, svc, users, "carol@example.com", "password")

	_, refresh, _ := svc.Login(context.Background(), "carol@example.com", "password", "1.2.3.4")

	// First refresh consumes the token
	if _, _, err := svc.Refresh(context.Background(), refresh); err != nil {
		t.Fatalf("first Refresh failed: %v", err)
	}

	// Replaying the old token must fail
	_, _, err := svc.Refresh(context.Background(), refresh)
	if !errors.Is(err, service.ErrTokenRevoked) {
		t.Errorf("want ErrTokenRevoked on replay, got %v", err)
	}
}

func TestLogout_RevokesToken(t *testing.T) {
	svc, users, _ := newTestService()
	mustRegisterVerified(t, svc, users, "dave@example.com", "password")

	_, refresh, _ := svc.Login(context.Background(), "dave@example.com", "password", "1.2.3.4")

	if err := svc.Logout(context.Background(), refresh); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	_, _, err := svc.Refresh(context.Background(), refresh)
	if !errors.Is(err, service.ErrTokenRevoked) {
		t.Errorf("want ErrTokenRevoked after logout, got %v", err)
	}
}

func TestRateLimit_BlocksAfterMax(t *testing.T) {
	svc, _, _ := newTestService()
	mustRegister(t, svc, "victim@example.com", "correctpass")

	// Exhaust 10 attempts with wrong password
	for i := 0; i < 10; i++ {
		_, _, _ = svc.Login(context.Background(), "victim@example.com", "wrong", "192.168.1.1")
	}

	// 11th attempt — even with correct password — must be rate-limited
	_, _, err := svc.Login(context.Background(), "victim@example.com", "correctpass", "192.168.1.1")
	if !errors.Is(err, service.ErrRateLimitExceeded) {
		t.Errorf("want ErrRateLimitExceeded, got %v", err)
	}
}

func TestValidateAccessToken(t *testing.T) {
	svc, users, _ := newTestService()
	mustRegisterVerified(t, svc, users, "eve@example.com", "password")

	access, _, _ := svc.Login(context.Background(), "eve@example.com", "password", "1.2.3.4")

	claims, err := svc.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.UserID == uuid.Nil {
		t.Error("UserID must not be nil")
	}
	if claims.JTI == "" {
		t.Error("JTI must not be empty")
	}
}

func TestValidateAccessToken_RefreshTokenRejected(t *testing.T) {
	svc, users, _ := newTestService()
	mustRegisterVerified(t, svc, users, "eve@example.com", "password")

	_, refresh, _ := svc.Login(context.Background(), "eve@example.com", "password", "1.2.3.4")

	// A refresh token must not be accepted as an access token
	_, err := svc.ValidateAccessToken(refresh)
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("want ErrInvalidToken for refresh token, got %v", err)
	}
}

// ── email verification (TASK-0009) ──────────────────────────────────────────────

func TestRegister_IssuesVerificationToken(t *testing.T) {
	svc, _, _, mail, verif := newTestServiceFull()
	u := mustRegister(t, svc, "newuser@example.com", "password123")

	if u.IsEmailVerified {
		t.Error("new user must start unverified")
	}
	if len(mail.sent) != 1 || mail.sent[0].to != "newuser@example.com" {
		t.Fatalf("want 1 verification email to newuser@, got %+v", mail.sent)
	}
	if len(verif.byHash) != 1 {
		t.Errorf("want 1 stored token hash, got %d", len(verif.byHash))
	}
	// The raw token (from the link) must NOT be a key — only its SHA-256 hash is stored.
	raw := tokenFromURL(t, mail.sent[0].verifyURL)
	if _, ok := verif.byHash[raw]; ok {
		t.Error("raw token must never be stored; only its hash")
	}
}

func TestLogin_UnverifiedRejected(t *testing.T) {
	svc, _, _ := newTestService()
	mustRegister(t, svc, "unverified@example.com", "password123")

	_, _, err := svc.Login(context.Background(), "unverified@example.com", "password123", "127.0.0.1")
	if !errors.Is(err, service.ErrEmailNotVerified) {
		t.Errorf("want ErrEmailNotVerified, got %v", err)
	}
}

func TestLogin_UnverifiedWrongPasswordStillInvalidCredentials(t *testing.T) {
	svc, _, _ := newTestService()
	mustRegister(t, svc, "unverified@example.com", "password123")

	// Wrong password must not leak verification state (no enumeration).
	_, _, err := svc.Login(context.Background(), "unverified@example.com", "wrong", "127.0.0.1")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("want ErrInvalidCredentials (not ErrEmailNotVerified), got %v", err)
	}
}

func TestVerifyEmail_SuccessThenLogin(t *testing.T) {
	svc, _, _, mail, _ := newTestServiceFull()
	mustRegister(t, svc, "verify@example.com", "password123")
	raw := tokenFromURL(t, mail.sent[0].verifyURL)

	already, err := svc.VerifyEmail(context.Background(), raw, "127.0.0.1")
	if err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	if already {
		t.Error("first verify must not report already-verified")
	}
	if _, _, err := svc.Login(context.Background(), "verify@example.com", "password123", "127.0.0.1"); err != nil {
		t.Errorf("login after verify should succeed, got %v", err)
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.VerifyEmail(context.Background(), "not-a-real-token", "127.0.0.1")
	if !errors.Is(err, service.ErrInvalidVerificationToken) {
		t.Errorf("want ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerifyEmail_SingleUse(t *testing.T) {
	svc, _, _, mail, _ := newTestServiceFull()
	mustRegister(t, svc, "once@example.com", "password123")
	raw := tokenFromURL(t, mail.sent[0].verifyURL)

	if _, err := svc.VerifyEmail(context.Background(), raw, "127.0.0.1"); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	_, err := svc.VerifyEmail(context.Background(), raw, "127.0.0.1")
	if !errors.Is(err, service.ErrInvalidVerificationToken) {
		t.Errorf("want ErrInvalidVerificationToken on replay, got %v", err)
	}
}

func TestResendVerification_UnverifiedSendsEmail(t *testing.T) {
	svc, _, _, mail, _ := newTestServiceFull()
	mustRegister(t, svc, "resend@example.com", "password123") // 1 email on register

	if err := svc.ResendVerification(context.Background(), "resend@example.com", "9.9.9.9"); err != nil {
		t.Fatalf("ResendVerification: %v", err)
	}
	if len(mail.sent) != 2 {
		t.Errorf("want 2 emails (register + resend), got %d", len(mail.sent))
	}
}

func TestResendVerification_UnknownEmailSucceedsNoSend(t *testing.T) {
	svc, _, _, mail, _ := newTestServiceFull()
	// Anti-enumeration: unknown email still returns nil (202) and sends nothing.
	if err := svc.ResendVerification(context.Background(), "ghost@example.com", "9.9.9.9"); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if len(mail.sent) != 0 {
		t.Errorf("no email for unknown address, got %d", len(mail.sent))
	}
}

func TestResendVerification_AlreadyVerifiedNoSend(t *testing.T) {
	svc, users, _, mail, _ := newTestServiceFull()
	mustRegister(t, svc, "done@example.com", "password123")
	users.markVerified("done@example.com")
	mail.sent = nil // ignore the register email

	if err := svc.ResendVerification(context.Background(), "done@example.com", "9.9.9.9"); err != nil {
		t.Fatalf("ResendVerification: %v", err)
	}
	if len(mail.sent) != 0 {
		t.Errorf("verified account should get no resend email, got %d", len(mail.sent))
	}
}

func TestGetUser_ReturnsIdentityWithAdminFlag(t *testing.T) {
	svc, users, _ := newTestService()
	u := mustRegister(t, svc, "me@example.com", "password123")
	users.byID[u.ID].IsAdmin = true

	got, err := svc.GetUser(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Email != "me@example.com" || !got.IsAdmin {
		t.Errorf("want me@example.com/admin=true, got %s/admin=%v", got.Email, got.IsAdmin)
	}
}

func TestGetUser_UnknownID(t *testing.T) {
	svc, _, _ := newTestService()
	if _, err := svc.GetUser(context.Background(), uuid.New()); err == nil {
		t.Fatal("want error for unknown user id, got nil")
	}
}
