package middleware

import (
	"net/http"

	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminOnly gates a route to administrators. It MUST run after Auth, which sets
// UserIDKey. The admin flag is checked fresh against the database on every
// request (service.IsAdmin), not read from the access token, so a revoked admin
// loses access immediately.
//
// A non-admin is denied with 403 before any resource lookup happens — so a
// non-admin write to any station id (real or not) returns 403, never 404,
// leaking nothing about which stations exist.
func AdminOnly(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.MustGet(UserIDKey).(uuid.UUID)
		isAdmin, err := authSvc.IsAdmin(c.Request.Context(), userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				gin.H{"error": "internal error", "code": "INTERNAL"})
			return
		}
		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "admin privileges required", "code": "FORBIDDEN"})
			return
		}
		c.Next()
	}
}
