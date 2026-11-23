// internal/handler/top_borrowed_books_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetTopBorrowedBooksHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		report, err := service.GetTopBorrowedBooksReport(
			db.WithContext(c.Request.Context()),
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "Failed to retrieve top borrowed books report",
			})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}
