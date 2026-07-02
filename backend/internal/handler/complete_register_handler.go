// internal/handler/complete_register_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CompleteRegistrationRequest struct {
	Email           string `json:"email"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func CompleteRegistrationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CompleteRegistrationRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid request",
			})
			return
		}

		err := service.CompleteRegistration(
			db,
			req.Email,
			req.Username,
			req.Password,
			req.ConfirmPassword,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "registration successful",
		})
	}
}
