package middleware

import (
	"net/http"
	"strings"

	"voltana-api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserIDKey is the Gin context key set by the Auth middleware.
const UserIDKey = "user_id"

// Auth validates the Bearer access token and sets UserIDKey on the context.
// Downstream handlers retrieve the user ID with:
//
//	userID := c.MustGet(middleware.UserIDKey).(uuid.UUID)
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "authorization header required", "code": "UNAUTHORIZED"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := authSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token", "code": "UNAUTHORIZED"})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}
