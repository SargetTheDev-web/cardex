// internal/handler/audit_handler.go

package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/internal/repository"
)

func GetAuditDashboardHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		location, err := time.LoadLocation("Asia/Manila")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Unable to determine dashboard date",
			})
			return
		}

		now := time.Now().In(location)
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
		end := start.AddDate(0, 0, 1)

		summary, err := repository.GetAuditDashboardSummary(db, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Unable to load audit dashboard summary",
			})
			return
		}

		logs, err := repository.GetRecentAuditDashboardLogs(db, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Unable to load audit logs",
			})
			return
		}

		showAlert := summary.SecurityEvents > 0

		message := ""
		if showAlert {
			message = "Security events were recorded today. Review the audit logs."
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  200,
			"summary": summary,
			"alert": gin.H{
				"show_alert":    showAlert,
				"flagged_count": summary.SecurityEvents,
				"message":       message,
				"action_label":  "Review logs",
				"action_url":    "/audit/dashboard",
			},
			"logs": logs,
		})
	}
}
