// internal/handler/loan_handler.go

package handler

import (
	"errors"
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type borrowBookRequest struct {
	UserID  int    `json:"user_id" binding:"required,gt=0"`
	Barcode string `json:"barcode" binding:"required"`
}

func BorrowBookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req borrowBookRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"error":   "invalid_request",
				"message": "A valid user_id and barcode are required",
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

		result, err := service.BorrowBook(
			db,
			req.UserID,
			req.Barcode,
			staffID,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)

		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidBorrowRequest):
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"error":   "invalid_request",
					"message": err.Error(),
				})
			case errors.Is(err, service.ErrBorrowerNotEligible):
				c.JSON(http.StatusForbidden, gin.H{
					"status":  http.StatusForbidden,
					"error":   "borrower_not_eligible",
					"message": "The borrower account is not eligible to borrow",
				})
			case errors.Is(err, service.ErrCopyUnavailable):
				c.JSON(http.StatusConflict, gin.H{
					"status":  http.StatusConflict,
					"error":   "copy_unavailable",
					"message": "The copy does not exist or is not available",
				})
			case errors.Is(err, service.ErrInvalidLoanPolicy):
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  http.StatusInternalServerError,
					"error":   "invalid_loan_policy",
					"message": "The borrowing policy is missing or invalid",
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  http.StatusInternalServerError,
					"error":   "borrow_failed",
					"message": "Unable to process the borrowing transaction",
				})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"message": result.Message,
			"data":    result,
		})
	}
}
