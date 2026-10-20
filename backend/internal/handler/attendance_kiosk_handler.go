// internal/handler/attendance_kiosk_handler.go

package handler

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ActivateAttendanceKioskRequest struct {
	Campus        string `json:"campus"`
	DurationHours int    `json:"duration_hours"`
}

type DeactivateAttendanceKioskRequest struct {
	SessionID int64 `json:"session_id" binding:"required,gt=0"`
}

func ActivateAttendanceKioskHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ActivateAttendanceKioskRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request body",
			})
			return
		}

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
				"error":  "invalid user identity",
			})
			return
		}

		// Default to a 12-hour operating window.
		if req.DurationHours == 0 {
			req.DurationHours = 12
		}

		activation, err := service.ActivateAttendanceKiosk(
			db,
			userID,
			req.Campus,
			time.Duration(req.DurationHours)*time.Hour,
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  err.Error(),
			})
			return
		}

		// The raw credential is returned only at activation.
		// Never log this response or store the credential in the database.
		c.JSON(http.StatusCreated, gin.H{
			"status":     http.StatusCreated,
			"message":    "attendance kiosk activated",
			"session_id": activation.SessionID,
			"campus":     activation.Campus,
			"expires_at": activation.ExpiresAt,
			"credential": activation.Credential,
		})
	}
}

func DeactivateAttendanceKioskHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req DeactivateAttendanceKioskRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "a valid session_id is required",
			})
			return
		}

		err := service.DeactivateAttendanceKiosk(db, req.SessionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status": http.StatusNotFound,
					"error":  "active kiosk session not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to deactivate kiosk",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "attendance kiosk deactivated",
		})
	}
}
