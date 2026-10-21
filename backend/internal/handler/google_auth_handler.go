// internal/handler/google_auth_handler.go

package handler

import (
	"errors"
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GoogleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func GoogleAuthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GoogleAuthRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"code":    "invalid_request",
				"message": "A valid id_token is required",
			})
			return
		}

		result, err := service.GoogleAuthenticate(
			db,
			req.IDToken,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {
			var authErr *service.GoogleAuthError
			if errors.As(err, &authErr) {
				c.JSON(authErr.HTTPStatus, gin.H{
					"status":  authErr.HTTPStatus,
					"code":    authErr.Code,
					"message": authErr.Message,
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"code":    "google_auth_failed",
				"message": "Google authentication failed",
			})
			return
		}

		if result.RegistrationRequired {
			c.JSON(http.StatusOK, gin.H{
				"status":                http.StatusOK,
				"code":                  "registration_required",
				"registration_required": true,
				"email":                 result.Email,
				"name":                  result.Name,
				"picture":               result.Picture,
			})
			return
		}

		login := result.Login

		c.JSON(http.StatusOK, gin.H{
			"status":            http.StatusOK,
			"message":           "Google login successful",
			"token":             login.Token,
			"expires_at":        login.TokenExpiresAt,
			"user_id":           login.User.UserID,
			"username":          login.User.Username,
			"email":             login.User.EmailAddress,
			"role_id":           login.User.RoleID,
			"permissions":       login.Permissions,
			"profile_image_url": result.ProfileImageURL,
		})
	}
}
