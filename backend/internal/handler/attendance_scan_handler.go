// internal/handler/attendance_scan_handler.go

package handler

import (
	"errors"
	"log"
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AttendanceScanRequest struct {
	UserID int `json:"user_id" binding:"required,gt=0"`
}

func AttendanceScanHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		credential := c.GetHeader("X-Kiosk-Credential")
		if credential == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "kiosk credential required",
			})
			return
		}

		var req AttendanceScanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "valid user_id is required",
			})
			return
		}

		result, err := service.RecordAttendanceScan(
			db,
			credential,
			req.UserID,
		)
		if err != nil {
			log.Printf("RecordAttendanceScan failed: user_id=%d error=%v", req.UserID, err)
		}

		switch {
		case errors.Is(err, service.ErrInvalidKioskCredential):
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "invalid or expired kiosk credential",
			})
		case errors.Is(err, service.ErrAttendanceUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  http.StatusNotFound,
				"message": "user not found",
			})
		case errors.Is(err, service.ErrAttendanceUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"status":  http.StatusForbidden,
				"message": "user account is not active",
			})
		case errors.Is(err, service.ErrDuplicateAttendanceScan):
			c.JSON(http.StatusConflict, gin.H{
				"status":  http.StatusConflict,
				"message": err.Error(),
			})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "failed to record attendance",
			})
		default:
			c.JSON(http.StatusCreated, gin.H{
				"status":  http.StatusCreated,
				"message": "attendance recorded",
				"data":    result,
			})
		}
	}
}
