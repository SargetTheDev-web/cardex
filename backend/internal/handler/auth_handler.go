// internal/handler/auth_handler.go

package handler

import (
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
				"error": "invalid input",
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Get user profile
		_, profile, profileErr := repository.GetUserProfile(
			db,
			result.User.UserID,
		)

		name := result.User.Username

		if profileErr == nil && profile != nil {

			name = profile.FirstName

			if profile.MiddleName != nil &&
				*profile.MiddleName != "" {
				name += " " + *profile.MiddleName
			}

			if profile.LastName != "" {
				name += " " + profile.LastName
			}

			if profile.SuffixExtension != nil &&
				*profile.SuffixExtension != "" {
				name += " " + *profile.SuffixExtension
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"token":       result.Token,
			"id":          result.User.UserID,
			"username":    result.User.Username,
			"name":        name,
			"email":       result.User.EmailAddress,
			"role":        result.Role.RoleCode,
			"permissions": result.Permissions,
		})
	}
}
