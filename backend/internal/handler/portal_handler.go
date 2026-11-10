// internal/handler/portal_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPortalHandler(db *gorm.DB) gin.HandlerFunc {
	portalService := service.NewPortalService(db)

	return func(c *gin.Context) {
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Authentication required",
			})
			return
		}

		userID, ok := userIDValue.(int)
		if !ok || userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid authenticated user",
			})
			return
		}

		dashboard, err := portalService.GetDashboard(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to retrieve portal dashboard",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Portal dashboard retrieved successfully",
			"data":    dashboard,
		})
	}
}
