package handler

import (
	"log"
	"net/http"

	"voltana-api/internal/middleware"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AccountHandler backs the /v1/account/* routes (JWT-protected).
type AccountHandler struct {
	auth *service.AuthService
}

func NewAccountHandler(auth *service.AuthService) *AccountHandler {
	return &AccountHandler{auth: auth}
}

// BotLink godoc
// POST /v1/account/bot-link
// Mints a short-lived linking token and returns deep links for whichever bot
// platforms are configured. The user opens the link in Bale or Telegram → the
// bot captures their verified phone number → the account is linked.
func (h *AccountHandler) BotLink(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)

	baleURL, telegramURL, err := h.auth.InitiateBotLink(c.Request.Context(), userID)
	if err != nil {
		if err == service.ErrNoBotConfig {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bot integration not configured"})
			return
		}
		log.Printf("bot-link: userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate link"})
		return
	}

	resp := gin.H{}
	if baleURL != "" {
		resp["bale_url"] = baleURL
	}
	if telegramURL != "" {
		resp["telegram_url"] = telegramURL
	}
	c.JSON(http.StatusOK, resp)
}
