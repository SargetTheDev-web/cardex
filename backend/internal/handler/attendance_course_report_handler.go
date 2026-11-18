// internal/handler/attendance_course_report_handler.go

package handler

import (
	"log"
	"net/http"
	"time"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AttendanceCourseReportHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromDate := c.Query("from_date")
		toDate := c.Query("to_date")

		if fromDate == "" || toDate == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "from_date and to_date are required",
			})
			return
		}

		for _, date := range []string{fromDate, toDate} {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"message": "dates must use YYYY-MM-DD format",
				})
				return
			}
		}

		report, err := service.GetAttendanceCourseReport(
			db,
			fromDate,
			toDate,
		)

		if err != nil {
			log.Printf(
				"GetAttendanceCourseReport failed: %v",
				err,
			)

			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"data":   report,
		})
	}
}
