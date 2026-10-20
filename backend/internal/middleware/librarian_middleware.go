// internal/middleware/librarian_middleware.go

package middleware

import (
	"net/http"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireLibrarianOrAdmin allows ADMIN (role 1)
// and LIBRARIAN (role 2) to access a protected route.
//
// AuthMiddleware must run first.
func RequireLibrarianOrAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			c.Abort()
			return
		}

		userID, ok := userIDValue.(int)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user identity",
			})
			c.Abort()
			return
		}

		roleID, err := repository.GetUserRoleID(db, userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "user account not found",
			})
			c.Abort()
			return
		}

		// Role 1 = ADMIN; role 2 = LIBRARIAN.
		if roleID != 1 && roleID != 2 {
			c.JSON(http.StatusForbidden, gin.H{
				"status": http.StatusForbidden,
				"error":  "librarian or administrator privileges required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
