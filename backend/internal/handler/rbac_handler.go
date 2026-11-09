// internal/handler/rbac_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateRolePermissionMatrixRequest struct {
	Roles []service.RolePermissionUpdate `json:"roles" binding:"required"`
}

func GetRBACMatrixHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matrix, err := service.GetRBACMatrix(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve RBAC matrix",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"data":   matrix,
		})
	}
}

func UpdateRBACMatrixHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request UpdateRolePermissionMatrixRequest

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid RBAC matrix request",
			})
			return
		}

		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		userID, ok := userIDValue.(int)
		if !ok || userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user identity",
			})
			return
		}

		if err := service.UpdateRBACMatrix(
			db,
			userID,
			request.Roles,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "RBAC permission matrix updated successfully",
		})
	}
}
