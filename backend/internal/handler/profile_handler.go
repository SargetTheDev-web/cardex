// internal/handler/profile_handler.go

package handler

import (
	"net/http"

	"backend/internal/repository"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RequestProfileUpdateHandler(
	db *gorm.DB,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		userIDValue, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		userID, ok := userIDValue.(int)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user session",
			})
			return
		}

		var req service.ProfileUpdateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request",
			})
			return
		}

		err := service.RequestProfileUpdate(
			db,
			userID,
			req,
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

		c.JSON(http.StatusAccepted, gin.H{
			"status":  http.StatusAccepted,
			"message": "profile changes submitted for admin approval",
		})
	}
}

func GetProfileChangeRequestHandler(
	db *gorm.DB,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		userIDValue, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		userID, ok := userIDValue.(int)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user session",
			})
			return
		}

		request, err := repository.GetPendingProfileChange(
			db,
			userID,
		)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  http.StatusNotFound,
				"message": "no pending profile changes",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"request": request,
		})
	}
}

func GetProfileHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userIDValue, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		userID, ok := userIDValue.(int)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user session",
			})
			return
		}

		user, profile, err := repository.GetUserProfile(
			db,
			userID,
		)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"status": http.StatusNotFound,
				"error":  "profile not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   http.StatusOK,
			"user_id":  user.UserID,
			"username": user.Username,
			"email":    user.EmailAddress,
			"profile":  profile,
		})
	}
}
