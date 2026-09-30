// internal/handler/forgot_password_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func ForgotPasswordHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ForgotPasswordRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request",
			})
			return
		}

		err := service.RequestPasswordReset(
			db,
			req.Email,
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
			"message": "password reset link sent",
		})
	}
}
