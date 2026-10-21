// internal/handler/google_register_handler.go

package handler

import (
	"net/http"
	"strings"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GoogleRegistrationRequest struct {
	IDToken         string  `json:"id_token" binding:"required"`
	Username        string  `json:"username" binding:"required"`
	Role            string  `json:"role" binding:"required"`
	InstitutionalID string  `json:"institutional_id" binding:"required"`
	LastName        string  `json:"last_name" binding:"required"`
	FirstName       string  `json:"first_name" binding:"required"`
	MiddleName      *string `json:"middle_name"`
	SuffixExtension *string `json:"suffix_extension"`
	Course          string  `json:"course"`
	MobileNumber    *string `json:"mobile_number"`
	PIN             string  `json:"pin" binding:"required"`
}

func GoogleRegisterCompleteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GoogleRegistrationRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"code":    "invalid_request",
				"message": "Required registration fields are missing or invalid",
			})
			return
		}

		err := service.CompleteGoogleRegistration(
			db,
			service.GoogleRegistrationInput{
				IDToken:         req.IDToken,
				Username:        req.Username,
				Role:            req.Role,
				InstitutionalID: req.InstitutionalID,
				LastName:        req.LastName,
				FirstName:       req.FirstName,
				MiddleName:      req.MiddleName,
				SuffixExtension: req.SuffixExtension,
				Course:          req.Course,
				MobileNumber:    req.MobileNumber,
				PIN:             req.PIN,
				IP:              c.ClientIP(),
				UserAgent:       c.Request.UserAgent(),
			},
		)

		if err != nil {
			message := err.Error()
			status := http.StatusBadRequest
			code := "google_registration_failed"

			switch {
			case strings.Contains(message, "already registered"),
				strings.Contains(message, "already in use"),
				strings.Contains(message, "already exists"),
				strings.Contains(message, "account linking is required"):
				status = http.StatusConflict
				code = "registration_conflict"

			case strings.Contains(message, "Google ID token"),
				strings.Contains(message, "verified email"):
				status = http.StatusUnauthorized
				code = "invalid_google_identity"

			case strings.Contains(message, "not configured"),
				strings.Contains(message, "failed to create"),
				strings.Contains(message, "failed to audit"),
				strings.Contains(message, "failed to check"),
				strings.Contains(message, "failed to validate"),
				strings.Contains(message, "failed to secure"):
				status = http.StatusInternalServerError
				code = "google_registration_error"
				message = "Unable to complete Google registration"
			}

			c.JSON(status, gin.H{
				"status":  status,
				"code":    code,
				"message": message,
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"code":    "registration_pending",
			"message": "Registration submitted. Your account is awaiting administrator approval.",
		})
	}
}
