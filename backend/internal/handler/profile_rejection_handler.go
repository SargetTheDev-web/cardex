// internal/handler/profile_rejection_handler.go

package handler

import (
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RejectProfileRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func RejectProfileChangeHandler(
	db *gorm.DB,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		idString := c.Param("id")

		changeRequestID, err := strconv.ParseInt(
			idString,
			10,
			64,
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid change request ID",
			})
			return
		}

		adminIDValue, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		adminID, ok := adminIDValue.(int)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid admin session",
			})
			return
		}

		var req RejectProfileRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "rejection reason is required",
			})
			return
		}

		err = service.RejectProfileChange(
			db,
			changeRequestID,
			adminID,
			req.Reason,
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "profile change request rejected",
		})
	}
}
