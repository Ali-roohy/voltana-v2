package handler

import (
	"errors"
	"log"
	"net/http"

	"voltana-api/internal/middleware"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PushHandler backs the web-push routes (TASK-0039). All routes JWT-protected;
// the admin test route additionally sits behind AdminOnly.
type PushHandler struct {
	push *service.PushService
}

func NewPushHandler(push *service.PushService) *PushHandler {
	return &PushHandler{push: push}
}

// VAPIDKey handles GET /v1/push/vapid-key.
func (h *PushHandler) VAPIDKey(c *gin.Context) {
	key, err := h.push.VAPIDPublicKey()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "push notifications are not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vapid_public_key": key})
}

type pushSubscribeReq struct {
	Endpoint string `json:"endpoint" binding:"required,max=2048"`
	Keys     struct {
		P256dh string `json:"p256dh" binding:"required,max=512"`
		Auth   string `json:"auth"   binding:"required,max=512"`
	} `json:"keys" binding:"required"`
}

// Subscribe handles POST /v1/account/push-subscription (the browser's
// PushSubscription.toJSON() shape).
func (h *PushHandler) Subscribe(c *gin.Context) {
	var req pushSubscribeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint and keys are required"})
		return
	}
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)

	err := h.push.Subscribe(c.Request.Context(), userID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth)
	switch {
	case errors.Is(err, service.ErrPushDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "push notifications are not configured"})
	case errors.Is(err, service.ErrInvalidEndpoint):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err != nil:
		log.Printf("push: subscribe userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store subscription"})
	default:
		c.Status(http.StatusCreated)
	}
}

type pushUnsubscribeReq struct {
	Endpoint string `json:"endpoint" binding:"required,max=2048"`
}

// Unsubscribe handles DELETE /v1/account/push-subscription.
func (h *PushHandler) Unsubscribe(c *gin.Context) {
	var req pushUnsubscribeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint is required"})
		return
	}
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	if err := h.push.Unsubscribe(c.Request.Context(), userID, req.Endpoint); err != nil {
		log.Printf("push: unsubscribe userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove subscription"})
		return
	}
	c.Status(http.StatusNoContent)
}

// AdminTestPush handles POST /v1/admin/test-push — sends a test notification
// to the calling admin's own subscriptions (mirrors the OTP test panel).
func (h *PushHandler) AdminTestPush(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	sent, err := h.push.SendToUser(c.Request.Context(), userID, service.PushPayload{
		Title: "🔔 تست اعلان ولتانا",
		Body:  "اعلان‌های وب به درستی کار می‌کنند",
		URL:   "/settings",
	})
	switch {
	case errors.Is(err, service.ErrPushDisabled):
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "push notifications are not configured"})
	case err != nil:
		log.Printf("push: test userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "send failed"})
	case sent == 0:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "no active push subscriptions for this account — enable notifications first"})
	default:
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "sent", "sent": sent})
	}
}
