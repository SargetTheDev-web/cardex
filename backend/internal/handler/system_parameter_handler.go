// internal/handlers/system_parameter_handler.go

package handler

import (
	"errors"
	"net/http"
	"strings"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type updateSystemParameterRequest struct {
	ParameterValue *string `json:"parameter_value" binding:"required"`
}

func getSystemParameterActor(c *gin.Context) (*int, error) {
	// Match this key to the one set by your AuthMiddleware.
	value, exists := c.Get("user_id")
	if !exists {
		return nil, errors.New("authenticated user ID missing")
	}

	var userID int

	switch id := value.(type) {
	case int:
		userID = id
	case int64:
		if id <= 0 || int64(int(id)) != id {
			return nil, errors.New("invalid authenticated user ID")
		}
		userID = int(id)
	default:
		return nil, errors.New("invalid authenticated user ID type")
	}

	if userID <= 0 {
		return nil, errors.New("invalid authenticated user ID")
	}

	return &userID, nil
}

func GetAllSystemParametersHandler(db *gorm.DB) gin.HandlerFunc {
	svc := service.NewSystemParameterService(db)

	return func(c *gin.Context) {
		userID, err := getSystemParameterActor(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to identify authenticated administrator",
			})
			return
		}

		params, err := svc.GetAll(
			userID,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to retrieve system parameters",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "System parameters retrieved successfully",
			"data":    params,
		})
	}
}

func GetSystemParameterHandler(db *gorm.DB) gin.HandlerFunc {
	svc := service.NewSystemParameterService(db)

	return func(c *gin.Context) {
		key := strings.TrimSpace(c.Param("key"))

		if key == "" || len(key) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid parameter key",
			})
			return
		}

		userID, err := getSystemParameterActor(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to identify authenticated administrator",
			})
			return
		}

		value, err := svc.Get(
			key,
			userID,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  http.StatusNotFound,
					"message": "System parameter not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to retrieve system parameter",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":          http.StatusOK,
			"message":         "System parameter retrieved successfully",
			"parameter_key":   key,
			"parameter_value": value,
		})
	}
}

func UpdateSystemParameterHandler(db *gorm.DB) gin.HandlerFunc {
	svc := service.NewSystemParameterService(db)

	return func(c *gin.Context) {
		key := strings.TrimSpace(c.Param("key"))

		if key == "" || len(key) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid parameter key",
			})
			return
		}

		var req updateSystemParameterRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
			})
			return
		}

		userID, err := getSystemParameterActor(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to identify authenticated administrator",
			})
			return
		}

		err = svc.Update(
			key,
			*req.ParameterValue,
			userID,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  http.StatusNotFound,
					"message": "System parameter not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Unable to update system parameter",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        http.StatusOK,
			"message":       "System parameter updated successfully",
			"parameter_key": key,
		})
	}
}
