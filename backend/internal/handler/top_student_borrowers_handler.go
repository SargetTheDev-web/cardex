// internal/handler/top_student_borrowers_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetTopStudentBorrowersHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		report, err := service.GetTopStudentBorrowersReport(
			db.WithContext(c.Request.Context()),
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
