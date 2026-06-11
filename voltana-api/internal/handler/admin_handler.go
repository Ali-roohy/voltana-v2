package handler

import (
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

// AdminHandler backs the /v1/admin/* routes (JWT + AdminOnly required).
type AdminHandler struct {
	auth   *service.AuthService
	admin  *service.AdminService
	sysSet *service.SystemSettingsService
}

func NewAdminHandler(auth *service.AuthService, admin *service.AdminService, sysSet *service.SystemSettingsService) *AdminHandler {
	return &AdminHandler{auth: auth, admin: admin, sysSet: sysSet}
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
	c.JSON(http.StatusOK, gin.H{"otp_delivery_method": settings.OTPDeliveryMethod})
}

type updateSystemSettingsReq struct {
	OTPDeliveryMethod string `json:"otp_delivery_method" binding:"required"`
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

	settings, err := h.sysSet.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "settings updated but failed to read back"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"otp_delivery_method": settings.OTPDeliveryMethod})
}
