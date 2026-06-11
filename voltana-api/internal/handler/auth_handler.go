package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"voltana-api/internal/middleware"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/auth"
)

// AuthHandler wires HTTP requests to AuthService.
type AuthHandler struct {
	auth   *service.AuthService
	isProd bool
	sysSet *service.SystemSettingsService // for GET /auth/otp/config (may be nil)
}

func NewAuthHandler(auth *service.AuthService, isProd bool, sysSet *service.SystemSettingsService) *AuthHandler {
	return &AuthHandler{auth: auth, isProd: isProd, sysSet: sysSet}
}

// ── request / response types ─────────────────────────────────────────────────

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email        string `json:"email"          binding:"required,email"`
	Password     string `json:"password"       binding:"required"`
	StayLoggedIn bool   `json:"stay_logged_in"`
}

type loginPhoneRequest struct {
	Phone        string `json:"phone"          binding:"required"`
	Password     string `json:"password"       binding:"required"`
	StayLoggedIn bool   `json:"stay_logged_in"`
}

type verifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type resendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type otpRequestBody struct {
	Phone    string `json:"phone"    binding:"required"`
	Platform string `json:"platform"` // "bale" | "telegram"; defaults to "bale"
}

type otpVerifyBody struct {
	Phone        string `json:"phone"          binding:"required"`
	Code         string `json:"code"           binding:"required,len=6"`
	Platform     string `json:"platform"`       // "bale" | "telegram"; must match the request
	StayLoggedIn bool   `json:"stay_logged_in"`
}

type otpRegisterBody struct {
	Phone    string  `json:"phone"    binding:"required"`
	Code     string  `json:"code"     binding:"required,len=6"`
	Platform string  `json:"platform" binding:"required"`
	Email    *string `json:"email"`
}

// ── handlers ──────────────────────────────────────────────────────────────────

// Register godoc
// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		log.Printf("register: user=%s err=%v", maskEmail(req.Email), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	log.Printf("register: new user id=%s", user.ID)
	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID,
		"email":   user.Email,
	})
}

// Login godoc
// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	access, refresh, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case errors.Is(err, service.ErrEmailNotVerified):
			c.JSON(http.StatusForbidden, gin.H{"error": "email not verified", "code": "EMAIL_NOT_VERIFIED"})
		case errors.Is(err, service.ErrRateLimitExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
		default:
			log.Printf("login: unexpected error: %v", err) // no credentials in log
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	h.setRefreshCookie(c, refresh, req.StayLoggedIn)
	c.JSON(http.StatusOK, tokenResponse{AccessToken: access})
}

// LoginPhone godoc
// POST /auth/login/phone
func (h *AuthHandler) LoginPhone(c *gin.Context) {
	var req loginPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	access, refresh, err := h.auth.LoginWithPhone(c.Request.Context(), req.Phone, req.Password, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoPasswordSet):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "NO_PASSWORD_SET"})
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials", "code": "INVALID_CREDENTIALS"})
		case errors.Is(err, service.ErrRateLimitExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
		default:
			log.Printf("login/phone: unexpected error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	h.setRefreshCookie(c, refresh, req.StayLoggedIn)
	c.JSON(http.StatusOK, tokenResponse{AccessToken: access})
}

// VerifyEmail godoc
// POST /auth/verify-email
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "code": "INVALID_REQUEST"})
		return
	}

	already, err := h.auth.VerifyEmail(c.Request.Context(), req.Token, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRateLimitExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later", "code": "RATE_LIMITED"})
		case errors.Is(err, service.ErrInvalidVerificationToken):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token", "code": "INVALID_VERIFICATION_TOKEN"})
		default:
			log.Printf("verify-email: unexpected error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		}
		return
	}

	if already {
		c.JSON(http.StatusOK, gin.H{"message": "email already verified"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

// ResendVerification godoc
// POST /auth/resend-verification
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "code": "INVALID_REQUEST"})
		return
	}

	if err := h.auth.ResendVerification(c.Request.Context(), req.Email, c.ClientIP()); err != nil {
		switch {
		case errors.Is(err, service.ErrRateLimitExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, try again later", "code": "RATE_LIMITED"})
		default:
			log.Printf("resend-verification: unexpected error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
		}
		return
	}

	// Always 202 for a well-formed request (anti-enumeration).
	c.JSON(http.StatusAccepted, gin.H{"message": "if the account exists and is unverified, a verification email has been sent"})
}

// Refresh godoc
// POST /auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil || strings.TrimSpace(refreshToken) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	access, newRefresh, err := h.auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	h.setRefreshCookie(c, newRefresh, false)
	c.JSON(http.StatusOK, tokenResponse{AccessToken: access})
}

// Logout godoc
// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err == nil && refreshToken != "" {
		if rErr := h.auth.Logout(c.Request.Context(), refreshToken); rErr != nil {
			log.Printf("logout: revocation failed: %v", rErr)
		}
	}

	// Clear the cookie regardless of revocation outcome
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", h.isProd, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Me godoc
// GET /v1/me — identity for the authenticated user, including the is_admin flag
// (which is deliberately not in the access token). Backs the frontend admin guard;
// the API itself remains the real boundary via AdminOnly on write routes.
// Also returns phone/bot linked status for the Settings bot-link card.
func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.auth.GetUser(c.Request.Context(), c.MustGet(middleware.UserIDKey).(uuid.UUID))
	if err != nil {
		log.Printf("me: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":              user.ID,
		"email":           user.Email,
		"is_admin":        user.IsAdmin,
		"phone":           user.Phone,
		"bale_linked":     user.BaleChatID != nil,
		"telegram_linked": user.TelegramChatID != nil,
		"password_set":    len(user.PasswordHash) > 0,
	})
}

// OTPRequest godoc
// POST /auth/otp/request — sends a 6-digit OTP to the user's linked bot chat.
// In contact_share mode (default) always returns 202 (anti-enum).
// In deeplink mode, returns 200 + bale_url/telegram_url when the user has no
// existing chat_id so the frontend can open the bot via a deep link.
func (h *AuthHandler) OTPRequest(c *gin.Context) {
	var req otpRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		// Still 202: don't reveal whether the phone format was invalid.
		c.JSON(http.StatusAccepted, gin.H{"message": "if the account is linked, an OTP has been sent"})
		return
	}

	platform := normalizePlatform(req.Platform)
	deepLink, err := h.auth.RequestOTP(c.Request.Context(), req.Phone, c.ClientIP(), platform)
	if err != nil {
		if errors.Is(err, service.ErrRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
			return
		}
		log.Printf("otp/request: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
		return
	}

	if deepLink != nil {
		if deepLink.AwaitingContact {
			c.JSON(http.StatusOK, gin.H{"status": "awaiting_contact_share"})
			return
		}
		resp := gin.H{"status": "deep_link"}
		if deepLink.BaleURL != "" {
			resp["bale_url"] = deepLink.BaleURL
		} else {
			resp["bale_url"] = nil
		}
		if deepLink.TgURL != "" {
			resp["telegram_url"] = deepLink.TgURL
		} else {
			resp["telegram_url"] = nil
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "if the account is linked, an OTP has been sent"})
}

// OTPContactStatus polls whether the user has shared their bot contact.
// GET /auth/otp/contact-status?phone=09121234567&platform=bale
func (h *AuthHandler) OTPContactStatus(c *gin.Context) {
	phone := c.Query("phone")
	platform := normalizePlatform(c.Query("platform"))
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}
	status, err := h.auth.CheckContactShareStatus(c.Request.Context(), phone, platform)
	if err != nil {
		log.Printf("otp/contact-status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "status check failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}

// OTPConfig godoc
// GET /auth/otp/config — public endpoint to expose the current OTP delivery method.
func (h *AuthHandler) OTPConfig(c *gin.Context) {
	method := "contact_share"
	if h.sysSet != nil {
		if settings, err := h.sysSet.GetSettings(c.Request.Context()); err == nil {
			method = settings.OTPDeliveryMethod
		}
	}
	resp := gin.H{"delivery_method": method}
	baleUser, tgUser := h.auth.GetBotUsernames()
	if baleUser != "" {
		resp["bale_username"] = baleUser
	}
	if tgUser != "" {
		resp["tg_username"] = tgUser
	}
	c.JSON(http.StatusOK, resp)
}

// OTPVerify godoc
// POST /auth/otp/verify — validates the OTP and issues tokens.
func (h *AuthHandler) OTPVerify(c *gin.Context) {
	var req otpVerifyBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	platform := normalizePlatform(req.Platform)
	access, refresh, err := h.auth.CompleteOTPLogin(c.Request.Context(), req.Phone, req.Code, c.ClientIP(), platform)
	if err != nil {
		// B3: differentiate wrong-code (with remaining attempts) from locked.
		if errors.Is(err, service.ErrOTPLocked) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "account locked", "code": "OTP_LOCKED"})
			return
		}
		var otpErr *service.OTPInvalidError
		if errors.As(err, &otpErr) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":              "invalid or expired code",
				"code":               "INVALID_OTP",
				"remaining_attempts": otpErr.RemainingAttempts,
			})
			return
		}
		if errors.Is(err, service.ErrRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts"})
			return
		}
		log.Printf("otp/verify: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}

	h.setRefreshCookie(c, refresh, req.StayLoggedIn)
	c.JSON(http.StatusOK, tokenResponse{AccessToken: access})
}

// OTPRegister godoc
// POST /auth/otp/register — validates a registration OTP and creates a new account.
func (h *AuthHandler) OTPRegister(c *gin.Context) {
	var req otpRegisterBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	platform := normalizePlatform(req.Platform)
	access, refresh, err := h.auth.CompleteOTPRegister(c.Request.Context(), req.Phone, req.Code, c.ClientIP(), platform, req.Email)
	if err != nil {
		if errors.Is(err, service.ErrPhoneTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "phone already registered", "code": "PHONE_TAKEN"})
			return
		}
		if errors.Is(err, service.ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered", "code": "EMAIL_TAKEN"})
			return
		}
		if errors.Is(err, service.ErrOTPLocked) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "account locked", "code": "OTP_LOCKED"})
			return
		}
		var otpErr *service.OTPInvalidError
		if errors.As(err, &otpErr) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":              "invalid or expired code",
				"code":               "INVALID_OTP",
				"remaining_attempts": otpErr.RemainingAttempts,
			})
			return
		}
		if errors.Is(err, service.ErrRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts"})
			return
		}
		log.Printf("otp/register: unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}

	h.setRefreshCookie(c, refresh, false)
	c.JSON(http.StatusOK, tokenResponse{AccessToken: access})
}

// ── helpers ───────────────────────────────────────────────────────────────────

// normalizePlatform coerces arbitrary input to a valid Platform, defaulting to
// Bale so the old single-tab behaviour is preserved for callers that omit it.
func normalizePlatform(p string) service.Platform {
	if p == string(service.PlatformTelegram) {
		return service.PlatformTelegram
	}
	return service.PlatformBale
}

// setRefreshCookie writes the httpOnly refresh-token cookie.
// stayLoggedIn=true → 30-day persistent cookie; false → session cookie (Max-Age=0).
// The token's Redis TTL is always 30 days regardless of this flag.
func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string, stayLoggedIn bool) {
	maxAge := 0
	if stayLoggedIn {
		maxAge = int((30 * 24 * time.Hour).Seconds())
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, token, maxAge, refreshCookiePath, "", h.isProd, true)
}

// maskEmail returns "a***@domain.com" to avoid leaking full emails in logs.
func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 1 {
		return "***"
	}
	return string(email[0]) + "***" + email[at:]
}
