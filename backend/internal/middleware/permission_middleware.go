// internal/middleware/permission_middleware.go

package middleware

import (
	"net/http"
	"strings"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RequirePermission(
	db *gorm.DB,
	permissionCode string,
) gin.HandlerFunc {
	permissionCode = strings.TrimSpace(permissionCode)

	return func(c *gin.Context) {
		// AuthMiddleware must execute before this middleware.
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		userID, ok := userIDValue.(int)
		if !ok || userID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user identity",
			})
			return
		}

		// An empty permission code must never grant access.
		if permissionCode == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "permission configuration error",
			})
			return
		}

		allowed, err := repository.UserHasPermission(
			db,
			userID,
			permissionCode,
		)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to validate permission",
			})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status": http.StatusForbidden,
				"error":  "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}
