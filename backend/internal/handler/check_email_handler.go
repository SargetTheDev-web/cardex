// internal/handler/check_email_handler.go

package handler

import (
	"net/http"
	"net/mail"
	"strings"

	"backend/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CheckEmailRequest struct {
	Email string `json:"email" binding:"required"`
}

func CheckEmailHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CheckEmailRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "A valid email is required",
			})
			return
		}

		email := strings.ToLower(strings.TrimSpace(req.Email))

		parsedEmail, err := mail.ParseAddress(email)
		if err != nil || parsedEmail.Address != email {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid email format",
			})
			return
		}

		user, err := repository.GetUserByIdentifier(db, email)

		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to check email availability",
			})
			return
		}

		emailExists := err == nil && user != nil

		c.JSON(http.StatusOK, gin.H{
			"status":       http.StatusOK,
			"email_exists": emailExists,
			"available":    !emailExists,
			"message": func() string {
				if emailExists {
					return "This email is already registered"
				}
				return "This email is available"
			}(),
		})
	}
}
