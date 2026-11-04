// internal/handler/fine_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetFinesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := strings.ToUpper(strings.TrimSpace(c.Query("status")))

		var loanID int64
		if rawLoanID := strings.TrimSpace(c.Query("loan_id")); rawLoanID != "" {
			parsed, err := strconv.ParseInt(rawLoanID, 10, 64)
			if err != nil || parsed <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"error":   "invalid_loan_id",
					"message": "loan_id must be a positive integer",
				})
				return
			}
			loanID = parsed
		}

		fines, err := service.GetFines(db, status, loanID)
		if err != nil {
			if errors.Is(err, service.ErrInvalidFineRequest) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"error":   "invalid_fine_filter",
					"message": "status must be PENDING, PAID, or WAIVED",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"error":   "fines_fetch_failed",
				"message": "Unable to retrieve fines",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"data":   fines,
		})
	}
}

type updateFineStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func UpdateFineStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fineID, err := strconv.ParseInt(c.Param("fine_id"), 10, 64)
		if err != nil || fineID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"error":   "invalid_fine_id",
				"message": "fine_id must be a positive integer",
			})
			return
		}

		var req updateFineStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"error":   "invalid_request",
				"message": "A valid status is required",
			})
			return
		}

		staffIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		staffID, ok := staffIDValue.(int)
		if !ok || staffID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid_user_identity",
			})
			return
		}

		fine, err := service.UpdateFineStatus(
			db,
			fineID,
			req.Status,
			staffID,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidFineRequest):
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"error":   "invalid_fine_status",
					"message": "status must be PENDING, PAID, or WAIVED",
				})
			case errors.Is(err, service.ErrFineNotFound):
				c.JSON(http.StatusNotFound, gin.H{
					"status":  http.StatusNotFound,
					"error":   "fine_not_found",
					"message": "Fine was not found",
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  http.StatusInternalServerError,
					"error":   "fine_update_failed",
					"message": "Unable to update fine status",
				})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Fine status updated successfully",
			"data":    fine,
		})
	}
}
