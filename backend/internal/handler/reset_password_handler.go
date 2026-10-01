package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

func ResetPasswordHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req ResetPasswordRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid request body",
			})
			return
		}

		err := service.ResetPassword(
			db,
			req.Token,
			req.Password,
			req.ConfirmPassword,
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
			"message": "password reset successfully",
		})
	}
}
