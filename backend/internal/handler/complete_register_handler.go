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

	Role            string `json:"role"`
	InstitutionalID string `json:"institutional_id"`

	LastName        string  `json:"last_name"`
	FirstName       string  `json:"first_name"`
	MiddleName      *string `json:"middle_name"`
	SuffixExtension *string `json:"suffix_extension"`

	Course       string  `json:"course"`
	MobileNumber *string `json:"mobile_number"`
	PIN          string  `json:"pin"`
}

func CompleteRegistrationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CompleteRegistrationRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request",
			})
			return
		}

		err := service.CompleteRegistration(
			db,
			req.Email,
			req.Username,
			req.Password,
			req.ConfirmPassword,
			req.Role,
			req.InstitutionalID,
			req.LastName,
			req.FirstName,
			req.MiddleName,
			req.SuffixExtension,
			req.Course,
			req.MobileNumber,
			req.PIN,
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

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"message": "registration successful",
		})
	}
}
