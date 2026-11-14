// internal/handler/qr_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getAuthenticatedUserID(c *gin.Context) (int, error) {
	value, exists := c.Get("user_id")

	if !exists {
		return 0, errors.New("unauthorized")
	}

	switch id := value.(type) {
	case int:
		if id > 0 {
			return id, nil
		}
	case int64:
		if id > 0 {
			return int(id), nil
		}
	case float64:
		if id > 0 && id == float64(int(id)) {
			return int(id), nil
		}
	case string:
		parsed, err := strconv.Atoi(id)
		if err == nil && parsed > 0 {
			return parsed, nil
		}
	}

	return 0, errors.New("invalid authenticated user")
}

func QRGenerateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, err := getAuthenticatedUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		result, err := service.GenerateQRCode(db, userID)

		if err != nil {
			if errors.Is(err, service.ErrQRAlreadyExists) {
				c.JSON(http.StatusConflict, gin.H{
					"status": http.StatusConflict,
					"error":  err.Error(),
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to generate QR code",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"message": "QR code generated successfully",
			"data":    result,
		})
	}
}

func QRGetHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, err := getAuthenticatedUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		result, err := service.GetQRCode(db, userID)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"status": http.StatusNotFound,
				"error":  "QR code not found",
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve QR code",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"data":   result,
		})
	}
}

func QRRegenerateHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, err := getAuthenticatedUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		result, err := service.RegenerateQRCode(db, userID)

		if errors.Is(err, service.ErrQRNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"status": http.StatusNotFound,
				"error":  err.Error(),
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to regenerate QR code",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "QR code regenerated successfully",
			"data":    result,
		})
	}
}

func QRDeleteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, err := getAuthenticatedUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "unauthorized",
			})
			return
		}

		err = service.RevokeQRCodeByUser(db, userID)

		if errors.Is(err, service.ErrQRNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"status": http.StatusNotFound,
				"error":  err.Error(),
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to revoke QR code",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "QR code revoked successfully",
		})
	}
}
