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
	BookID  int    `json:"book_id" binding:"required,gt=0"`
	DueDate string `json:"due_date" binding:"required"`
	Notes   string `json:"notes"`
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
			req.BookID,
			req.DueDate,
			req.Notes,
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

type returnBookRequest struct {
	LoanID    int64   `json:"loan_id" binding:"required,gt=0"`
	Amount    float64 `json:"amount"`
	Condition string  `json:"condition" binding:"required,oneof=GOOD DAMAGED"`
	Notes     string  `json:"notes"`
}

func ReturnBookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req returnBookRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"error":   "invalid_request",
				"message": "A barcode and condition (GOOD or DAMAGED) are required",
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

		result, err := service.ReturnBook(
			db,
			req.LoanID,
			req.Condition,
			req.Notes,
			staffID,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)

		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidReturnRequest):
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"error":   "invalid_request",
					"message": "A valid barcode and condition are required",
				})
			case errors.Is(err, service.ErrLoanNotFound):
				c.JSON(http.StatusConflict, gin.H{
					"status":  http.StatusConflict,
					"error":   "active_loan_not_found",
					"message": "No active loan was found for this copy",
				})
			case errors.Is(err, service.ErrInvalidFinePolicy):
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  http.StatusInternalServerError,
					"error":   "invalid_fine_policy",
					"message": "The fine policy is missing or invalid",
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  http.StatusInternalServerError,
					"error":   "return_failed",
					"message": "Unable to process the return transaction",
				})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": result.Message,
			"data":    result,
		})
	}
}

func GetRecentTransactionsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		transactions, err := service.GetRecentTransactions(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"error":   "transactions_fetch_failed",
				"message": "Unable to retrieve recent transactions",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"data":   transactions,
		})
	}
}
