// internal/handler/get_pending_profile_change_requests_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPendingProfileChangeRequestsHandler(
	db *gorm.DB,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		requests, err := service.GetPendingProfileChangeRequests(
			db,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   http.StatusOK,
			"requests": requests,
			"count":    len(requests),
		})
	}
}
