// internal/handler/book_copy_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	model "backend/internal/models"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateBookCopyRequest struct {
	BookID        int     `json:"book_id" binding:"required,gt=0"`
	Barcode       *string `json:"barcode"`
	ShelfLocation *string `json:"shelf_location"`
	CopyStatus    string  `json:"copy_status"`
}

type UpdateBookCopyRequest struct {
	BookID        int     `json:"book_id" binding:"required,gt=0"`
	Barcode       *string `json:"barcode"`
	ShelfLocation *string `json:"shelf_location"`
}

type UpdateBookCopyStatusRequest struct {
	CopyStatus string `json:"copy_status" binding:"required"`
}

func respondBookCopyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidBookCopy):
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})

	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Book copy not found",
		})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "An internal server error occurred",
		})
	}
}

func parseBookCopyID(c *gin.Context) (int, bool) {
	copyID, err := strconv.Atoi(c.Param("id"))
	if err != nil || copyID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "A valid book copy ID is required",
		})
		return 0, false
	}

	return copyID, true
}

func GetBookCopiesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		copies, err := service.GetAllBookCopies(
			db,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondBookCopyError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Book copies retrieved successfully",
			"data":    copies,
		})
	}
}

func GetBookCopyHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		copyID, valid := parseBookCopyID(c)
		if !valid {
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		copy, err := service.GetBookCopyByID(
			db,
			copyID,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondBookCopyError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Book copy retrieved successfully",
			"data":    copy,
		})
	}
}

func CreateBookCopyHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateBookCopyRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid request body",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		copy := model.BookCopy{
			BookID:        req.BookID,
			Barcode:       req.Barcode,
			ShelfLocation: req.ShelfLocation,
			CopyStatus:    req.CopyStatus,
		}

		created, err := service.CreateBookCopy(
			db,
			&copy,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondBookCopyError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"message": "Book copy created successfully",
			"data":    created,
		})
	}
}

func UpdateBookCopyHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		copyID, valid := parseBookCopyID(c)
		if !valid {
			return
		}

		var req UpdateBookCopyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid request body",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		updated := &model.BookCopy{
			BookID:        req.BookID,
			Barcode:       req.Barcode,
			ShelfLocation: req.ShelfLocation,
		}

		result, err := service.UpdateBookCopy(
			db,
			copyID,
			updated,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondBookCopyError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Book copy updated successfully",
			"data":    result,
		})
	}
}

func UpdateBookCopyStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		copyID, valid := parseBookCopyID(c)
		if !valid {
			return
		}

		var req UpdateBookCopyStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "copy_status is required",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Unauthorized",
			})
			return
		}

		result, err := service.UpdateBookCopyStatus(
			db,
			copyID,
			req.CopyStatus,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondBookCopyError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Book copy status updated successfully",
			"data":    result,
		})
	}
}
