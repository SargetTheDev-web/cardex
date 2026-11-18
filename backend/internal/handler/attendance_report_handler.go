// internal/handler/attendance_report_handler.go

package handler

import (
	"log"
	"net/http"
	"time"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AttendanceReportHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromDate := c.Query("from_date")
		toDate := c.Query("to_date")
		courseProgram := c.DefaultQuery(
			"course_program",
			"All courses / departments",
		)

		if fromDate == "" || toDate == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "from_date and to_date are required",
			})
			return
		}

		// Reject malformed dates at the handler level.
		for _, date := range []string{fromDate, toDate} {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"message": "dates must use YYYY-MM-DD format",
				})
				return
			}
		}

		report, err := service.GetAttendanceReport(
			db,
			fromDate,
			toDate,
			courseProgram,
		)

		if err != nil {
			log.Printf("GetAttendanceReport failed: %v", err)

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
