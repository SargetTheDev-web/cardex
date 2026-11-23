// internal/handler/top_student_borrowers_handler.go

package handler

import (
	"net/http"
	"time"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetTopStudentBorrowersHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		fromStr := c.Query("from")
		toStr := c.Query("to")

		if fromStr == "" || toStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "Both from and to date parameters are required",
			})
			return
		}

		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "Invalid from date. Expected YYYY-MM-DD",
			})
			return
		}

		to, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "Invalid to date. Expected YYYY-MM-DD",
			})
			return
		}

		if from.After(to) {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "The from date must not be later than the to date",
			})
			return
		}

		report, err := service.GetTopStudentBorrowersReport(
			db.WithContext(c.Request.Context()),
			from,
			to,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "Failed to retrieve top student borrowers report",
			})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}
