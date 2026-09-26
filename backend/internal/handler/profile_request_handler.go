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
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"requests": requests,
			"count":    len(requests),
		})
	}
}
