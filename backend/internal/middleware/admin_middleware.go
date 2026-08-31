package middleware

import (
	"net/http"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RequireAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// AuthMiddleware must run first.
		userIDValue, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		userID, ok := userIDValue.(int)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user identity",
			})
			c.Abort()
			return
		}

		// Get the user's role directly from the database.
		roleID, err := repository.GetUserRoleID(
			db,
			userID,
		)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user account not found",
			})
			c.Abort()
			return
		}

		// Role ID 1 = ADMIN.
		if roleID != 1 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "administrator privileges required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
