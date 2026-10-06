// internal/handler/change_pin_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChangePINRequest struct {
	CurrentPIN string `json:"current_pin" binding:"required"`
	NewPIN     string `json:"new_pin" binding:"required"`
	ConfirmPIN string `json:"confirm_pin" binding:"required"`
}

func ChangePINHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get authenticated user ID
		userIDValue, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		userID, ok := userIDValue.(int)

		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "invalid user identity",
			})
			return
		}

		// Parse request
		var req ChangePINRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request body",
			})
			return
		}

		// Change PIN
		err := service.ChangePIN(
			db,
			userID,
			req.CurrentPIN,
			req.NewPIN,
			req.ConfirmPIN,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "PIN changed successfully",
		})
	}
}
