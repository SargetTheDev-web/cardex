package handler

import (
	"net/http"
	"time"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ResetPasswordRequest struct {
	Token           string
	Password        string
	ConfirmPassword string
}

/*
ShowResetPasswordHandler

GET /auth/reset-password?token=...

This is responsible only for displaying
the password-reset page.
*/
func ShowResetPasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.Query("token")

		if token == "" {
			c.HTML(
				http.StatusBadRequest,
				"reset_password.html",
				gin.H{
					"Token":       "",
					"CurrentYear": time.Now().Year(),
					"Error":       "Invalid password reset link.",
				},
			)

			return
		}

		c.HTML(
			http.StatusOK,
			"reset_password.html",
			gin.H{
				"Token":       token,
				"CurrentYear": time.Now().Year(),
				"Error":       "",
			},
		)
	}
}

/*
ResetPasswordHandler

POST /auth/reset-password

The HTML form submits directly to Go.
*/
func ResetPasswordHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.PostForm("token")
		password := c.PostForm("password")
		confirmPassword := c.PostForm("confirm_password")

		err := service.ResetPassword(
			db,
			token,
			password,
			confirmPassword,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {

			c.HTML(
				http.StatusBadRequest,
				"reset_password.html",
				gin.H{
					"Token":       token,
					"CurrentYear": time.Now().Year(),
					"Error":       err.Error(),
				},
			)

			return
		}

		c.HTML(
			http.StatusOK,
			"reset_success.html",
			gin.H{
				"CurrentYear": time.Now().Year(),
			},
		)
	}
}
