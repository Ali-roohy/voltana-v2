package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"voltana-api/internal/domain"
	"voltana-api/internal/middleware"
	"voltana-api/internal/repository"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BotConnectionTester is satisfied by bot.ConnectionTester — kept as a local
// interface so the handler layer stays decoupled from the bot package (same
// pattern as bot.LinkCallback).
type BotConnectionTester interface {
	Test(ctx context.Context, platform string) (username string, latency time.Duration, err error)
}

// AdminHandler backs the /v1/admin/* routes (JWT + AdminOnly required).
type AdminHandler struct {
	auth      *service.AuthService
	admin     *service.AdminService
	sysSet    *service.SystemSettingsService
	botTester BotConnectionTester
}

func NewAdminHandler(auth *service.AuthService, admin *service.AdminService, sysSet *service.SystemSettingsService, botTester BotConnectionTester) *AdminHandler {
	return &AdminHandler{auth: auth, admin: admin, sysSet: sysSet, botTester: botTester}
}

type testOTPReq struct {
	Platform string `json:"platform" binding:"required"`
}

type testOTPResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestOTPDelivery handles POST /v1/admin/test-otp.
// Sends a fixed test code to the admin's own linked channel; no Redis key is written.
func (h *AdminHandler) TestOTPDelivery(c *gin.Context) {
	var req testOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, testOTPResp{Success: false, Message: "platform is required"})
		return
	}

	p := strings.ToLower(strings.TrimSpace(req.Platform))
	if p != "bale" && p != "telegram" && p != "email" {
		c.JSON(http.StatusBadRequest, testOTPResp{Success: false, Message: "platform must be bale, telegram, or email"})
		return
	}

	userID, ok := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	msg, err := h.auth.TestOTPDelivery(c.Request.Context(), userID, p)
	if err != nil {
		errMsg := err.Error()
		if strings.HasSuffix(errMsg, "not linked") || strings.HasSuffix(errMsg, "not set") {
			c.JSON(http.StatusBadRequest, testOTPResp{Success: false, Message: errMsg})
			return
		}
		c.JSON(http.StatusInternalServerError, testOTPResp{Success: false, Message: errMsg})
		return
	}

	c.JSON(http.StatusOK, testOTPResp{Success: true, Message: msg})
}

type testBotConnReq struct {
	Platform string `json:"platform" binding:"required"`
}

type testBotConnResp struct {
	OK          bool   `json:"ok"`
	BotUsername string `json:"bot_username,omitempty"`
	LatencyMS   int64  `json:"latency_ms,omitempty"`
	Error       string `json:"error,omitempty"`
}

// TestBotConnection handles POST /v1/admin/test-bot-connection (TASK-0036
// BUG-8). Calls the platform's getMe endpoint server-side with the env token;
// the token never reaches the client and errors are pre-sanitized by the
// tester (no token in URLs/messages).
func (h *AdminHandler) TestBotConnection(c *gin.Context) {
	var req testBotConnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, testBotConnResp{OK: false, Error: "platform is required"})
		return
	}
	p := strings.ToLower(strings.TrimSpace(req.Platform))
	if p != "bale" && p != "telegram" {
		c.JSON(http.StatusBadRequest, testBotConnResp{OK: false, Error: "platform must be bale or telegram"})
		return
	}

	username, latency, err := h.botTester.Test(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusOK, testBotConnResp{OK: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, testBotConnResp{
		OK:          true,
		BotUsername: username,
		LatencyMS:   latency.Milliseconds(),
	})
}

// ── user management ──────────────────────────────────────────────────────────

type userSummary struct {
	ID              string    `json:"id"`
	FullName        *string   `json:"full_name"`
	Email           *string   `json:"email"`
	Phone           *string   `json:"phone"`
	IsAdmin         bool      `json:"is_admin"`
	IsEmailVerified bool      `json:"is_email_verified"`
	BaleLinked      bool      `json:"bale_linked"`
	TelegramLinked  bool      `json:"telegram_linked"`
	CreatedAt       time.Time `json:"created_at"`
}

func toUserSummary(u *domain.User) userSummary {
	s := userSummary{
		ID:              u.ID.String(),
		FullName:        u.FullName,
		IsAdmin:         u.IsAdmin,
		IsEmailVerified: u.IsEmailVerified,
		BaleLinked:      u.BaleChatID != nil,
		TelegramLinked:  u.TelegramChatID != nil,
		CreatedAt:       u.CreatedAt,
	}
	if u.Email != "" {
		s.Email = &u.Email
	}
	s.Phone = u.Phone
	return s
}

type usersPage struct {
	Items  []userSummary `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// ListUsers handles GET /v1/admin/users?limit=20&offset=0.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	users, total, err := h.admin.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	items := make([]userSummary, len(users))
	for i, u := range users {
		items[i] = toUserSummary(u)
	}
	c.JSON(http.StatusOK, usersPage{Items: items, Total: total, Limit: limit, Offset: offset})
}

// GetUser handles GET /v1/admin/users/:id.
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	u, err := h.admin.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, toUserSummary(u))
}

type updateUserReq struct {
	IsAdmin         *bool `json:"is_admin"`
	IsEmailVerified *bool `json:"is_email_verified"`
}

// UpdateUser handles PUT /v1/admin/users/:id.
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	callerID, ok := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.IsAdmin == nil && req.IsEmailVerified == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one field required"})
		return
	}

	u, err := h.admin.UpdateUser(c.Request.Context(), callerID, targetID, req.IsAdmin, req.IsEmailVerified)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRemoveSelfAdmin):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrLastAdmin):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, repository.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, toUserSummary(u))
}

// DeleteUser handles DELETE /v1/admin/users/:id.
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	callerID, ok := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err = h.admin.DeleteUser(c.Request.Context(), callerID, targetID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeleteSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrLastAdmin):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, repository.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ── system settings ──────────────────────────────────────────────────────────

// GetSystemSettings handles GET /v1/admin/system-settings.
func (h *AdminHandler) GetSystemSettings(c *gin.Context) {
	settings, err := h.sysSet.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"otp_delivery_method":  settings.OTPDeliveryMethod,
		"default_peak_rate":    settings.DefaultPeakRate,
		"default_mid_rate":     settings.DefaultMidRate,
		"default_offpeak_rate": settings.DefaultOffpeakRate,
	})
}

type updateSystemSettingsReq struct {
	OTPDeliveryMethod string `json:"otp_delivery_method" binding:"required"`
	// Default rates are optional so the existing OTP-only PUT keeps working;
	// when any is present all three are required (full-replace, FEAT-6).
	DefaultPeakRate    *float64 `json:"default_peak_rate"`
	DefaultMidRate     *float64 `json:"default_mid_rate"`
	DefaultOffpeakRate *float64 `json:"default_offpeak_rate"`
}

// UpdateSystemSettings handles PUT /v1/admin/system-settings.
func (h *AdminHandler) UpdateSystemSettings(c *gin.Context) {
	var req updateSystemSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "otp_delivery_method required"})
		return
	}

	if err := h.sysSet.SetOTPDeliveryMethod(c.Request.Context(), req.OTPDeliveryMethod); err != nil {
		if errors.Is(err, service.ErrInvalidOTPDeliveryMethod) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	if req.DefaultPeakRate != nil || req.DefaultMidRate != nil || req.DefaultOffpeakRate != nil {
		if req.DefaultPeakRate == nil || req.DefaultMidRate == nil || req.DefaultOffpeakRate == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "all three default rates are required together"})
			return
		}
		if err := h.sysSet.SetDefaultRates(c.Request.Context(), *req.DefaultPeakRate, *req.DefaultMidRate, *req.DefaultOffpeakRate); err != nil {
			if errors.Is(err, service.ErrInvalidDefaultRates) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update default rates"})
			return
		}
	}

	settings, err := h.sysSet.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "settings updated but failed to read back"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"otp_delivery_method":  settings.OTPDeliveryMethod,
		"default_peak_rate":    settings.DefaultPeakRate,
		"default_mid_rate":     settings.DefaultMidRate,
		"default_offpeak_rate": settings.DefaultOffpeakRate,
	})
}
