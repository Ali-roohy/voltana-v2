package handler

import (
	"net/http"
	"strings"

	"voltana-api/internal/middleware"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler backs the /v1/admin/* routes (JWT + AdminOnly required).
type AdminHandler struct {
	auth *service.AuthService
}

func NewAdminHandler(auth *service.AuthService) *AdminHandler {
	return &AdminHandler{auth: auth}
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
