// internal/handler/user_report_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetUserReportHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Respect request cancellation and deadlines.
		ctx := c.Request.Context()
		requestDB := db.WithContext(ctx)

		users, err := service.GetUserReport(requestDB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "Failed to retrieve user report",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "User report retrieved successfully",
			"data":    users,
		})
	}
}
