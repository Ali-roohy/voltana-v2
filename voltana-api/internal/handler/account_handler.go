package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"voltana-api/internal/domain"
	"voltana-api/internal/middleware"
	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AccountHandler backs the /v1/account/* routes (JWT-protected).
type AccountHandler struct {
	auth   *service.AuthService
	backup *service.BackupService
	admin  *service.AdminService
	isProd bool
}

func NewAccountHandler(auth *service.AuthService, backup *service.BackupService, admin *service.AdminService, isProd bool) *AccountHandler {
	return &AccountHandler{auth: auth, backup: backup, admin: admin, isProd: isProd}
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

type setPasswordReq struct {
	Password string `json:"password" binding:"required"`
}

// SetPassword handles POST /v1/account/set-password.
// Sets or replaces a bcrypt password for the authenticated user.
func (h *AccountHandler) SetPassword(c *gin.Context) {
	var req setPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	if err := h.auth.SetPassword(c.Request.Context(), userID, req.Password); err != nil {
		if errors.Is(err, service.ErrPasswordTooShort) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("set-password: userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set password"})
		return
	}
	c.Status(http.StatusNoContent)
}

// importMaxBodyBytes caps the import upload (TASK-0037 FEAT-4).
const importMaxBodyBytes = 5 << 20 // 5 MB

// Export handles GET /v1/account/export — the authenticated user's own data
// as a downloadable JSON document.
func (h *AccountHandler) Export(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	b, err := h.backup.Export(c.Request.Context(), userID)
	if err != nil {
		log.Printf("account/export: userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export failed"})
		return
	}
	c.Header("Content-Disposition", `attachment; filename="voltana-backup.json"`)
	c.JSON(http.StatusOK, b)
}

// Import handles POST /v1/account/import — replaces the user's own data with
// the uploaded backup. Strictly scoped to the authenticated user; the service
// re-maps every id.
func (h *AccountHandler) Import(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importMaxBodyBytes)
	var b domain.UserBackup
	if err := json.NewDecoder(c.Request.Body).Decode(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	stats, err := h.backup.Import(c.Request.Context(), userID, &b)
	if err != nil {
		var vErr *service.BackupValidationError
		if errors.As(err, &vErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": vErr.Reason})
			return
		}
		log.Printf("account/import: userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "import complete", "imported": stats})
}

// DeleteAccount handles DELETE /v1/account — self-service account deletion
// (TASK-0037 FEAT-5). The permanent first admin is protected server-side by
// the last-admin guard. Owned data cascades; the refresh cookie is revoked
// and cleared.
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)

	if err := h.admin.SelfDelete(c.Request.Context(), userID); err != nil {
		if errors.Is(err, service.ErrLastAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": "the last admin account cannot be deleted", "code": "LAST_ADMIN"})
			return
		}
		log.Printf("account/delete: userID=%s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "account deletion failed"})
		return
	}

	// Best-effort refresh-token revocation + cookie clear (user row is gone,
	// so any surviving refresh token would fail its user lookup anyway).
	if refreshToken, err := c.Cookie(refreshCookieName); err == nil && refreshToken != "" {
		if rErr := h.auth.Logout(c.Request.Context(), refreshToken); rErr != nil {
			log.Printf("account/delete: token revocation: %v", rErr)
		}
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", h.isProd, true)
	c.Status(http.StatusNoContent)
}
