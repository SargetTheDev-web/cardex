// internal/handler/auth_handler.go

package handler

import (
	"errors"
	"net/http"

	"backend/internal/repository"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func LoginHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req LoginRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid input",
			})
			return
		}

		result, err := service.Login(
			db,
			req.Identifier,
			req.Password,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {

			var loginErr *service.LoginError

			if errors.As(err, &loginErr) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":                  http.StatusUnauthorized,
					"error":                   loginErr.Message,
					"login_retry_count":       loginErr.LoginRetryCount,
					"max_login_retry_count":   loginErr.MaxLoginAttempts,
					"remaining_login_retries": loginErr.RemainingLoginRetries,
					"account_locked":          loginErr.AccountLocked,
				})
				return
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// Get user profile
		// --------------------------------------------------

		_, profile, profileErr := repository.GetUserProfile(
			db,
			result.User.UserID,
		)

		// --------------------------------------------------
		// Default profile values
		// --------------------------------------------------

		var (
			firstName       string
			middleName      *string
			lastName        string
			suffixExtension *string
		)

		if profileErr == nil && profile != nil {

			firstName = profile.FirstName
			middleName = profile.MiddleName
			lastName = profile.LastName
			suffixExtension = profile.SuffixExtension
		}

		// --------------------------------------------------
		// Successful login response
		// --------------------------------------------------

		c.JSON(http.StatusOK, gin.H{
			"status":                http.StatusOK,
			"token":                 result.Token,
			"id":                    result.User.UserID,
			"username":              result.User.Username,
			"first_name":            firstName,
			"middle_name":           middleName,
			"last_name":             lastName,
			"suffix_extension":      suffixExtension,
			"email":                 result.User.EmailAddress,
			"sso_provider_id":       result.User.SSOProviderID,
			"role":                  result.Role.RoleCode,
			"permissions":           result.Permissions,
			"login_retry_count":     result.User.LoginRetryCount,
			"max_login_retry_count": result.MaxLoginAttempts,
			"expires_at":            result.TokenExpiresAt,
		})
	}
}

type QRLoginRequest struct {
	UserID int    `json:"user_id" binding:"required,gt=0"`
	PIN    string `json:"pin" binding:"required"`
}

func QRLoginHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req QRLoginRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid input",
			})
			return
		}

		result, err := service.LoginWithPIN(
			db,
			req.UserID,
			req.PIN,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {
			var loginErr *service.LoginError

			if errors.As(err, &loginErr) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":                  http.StatusUnauthorized,
					"error":                   loginErr.Message,
					"login_retry_count":       loginErr.LoginRetryCount,
					"max_login_retry_count":   loginErr.MaxLoginAttempts,
					"remaining_login_retries": loginErr.RemainingLoginRetries,
					"account_locked":          loginErr.AccountLocked,
				})
				return
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "QR login failed",
			})
			return
		}

		_, profile, profileErr := repository.GetUserProfile(
			db,
			result.User.UserID,
		)

		var (
			firstName       string
			middleName      *string
			lastName        string
			suffixExtension *string
		)

		if profileErr == nil && profile != nil {
			firstName = profile.FirstName
			middleName = profile.MiddleName
			lastName = profile.LastName
			suffixExtension = profile.SuffixExtension
		}

		c.JSON(http.StatusOK, gin.H{
			"status":                http.StatusOK,
			"token":                 result.Token,
			"id":                    result.User.UserID,
			"username":              result.User.Username,
			"first_name":            firstName,
			"middle_name":           middleName,
			"last_name":             lastName,
			"suffix_extension":      suffixExtension,
			"email":                 result.User.EmailAddress,
			"sso_provider_id":       result.User.SSOProviderID,
			"role":                  result.Role.RoleCode,
			"permissions":           result.Permissions,
			"login_retry_count":     result.User.LoginRetryCount,
			"max_login_retry_count": result.MaxLoginAttempts,
			"expires_at":            result.TokenExpiresAt,
		})
	}
}
