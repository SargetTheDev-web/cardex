// internal/handler/publisher_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PublisherRequest struct {
	PublisherName string `json:"publisher_name" binding:"required"`
	Address       string `json:"address"`
	ContactNumber string `json:"contact_number"`
}

func publisherIDFromContext(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": "Invalid publisher ID",
		})
		return 0, false
	}
	return id, true
}

func CreatePublisherHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PublisherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  400,
				"message": "Invalid request body",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  401,
				"message": "Unauthorized",
			})
			return
		}

		publisher, err := service.CreatePublisher(
			db, req.PublisherName, req.Address, req.ContactNumber,
			userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, service.ErrInvalidPublisher) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  400,
					"message": err.Error(),
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Failed to create publisher",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  201,
			"message": "Publisher created successfully",
			"data":    publisher,
		})
	}
}

func GetPublishersHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  401,
				"message": "Unauthorized",
			})
			return
		}

		publishers, err := service.GetAllPublishers(db, userID, ip, userAgent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Failed to retrieve publishers",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  200,
			"message": "Publishers retrieved successfully",
			"data":    publishers,
		})
	}
}

func GetPublisherHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := publisherIDFromContext(c)
		if !ok {
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  401,
				"message": "Unauthorized",
			})
			return
		}

		publisher, err := service.GetPublisherByID(
			db, id, userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  404,
					"message": "Publisher not found",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Failed to retrieve publisher",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  200,
			"message": "Publisher retrieved successfully",
			"data":    publisher,
		})
	}
}

func UpdatePublisherHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := publisherIDFromContext(c)
		if !ok {
			return
		}

		var req PublisherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  400,
				"message": "Invalid request body",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  401,
				"message": "Unauthorized",
			})
			return
		}

		publisher, err := service.UpdatePublisher(
			db, id, req.PublisherName, req.Address, req.ContactNumber,
			userID, ip, userAgent,
		)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidPublisher):
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  400,
					"message": err.Error(),
				})
			case errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusNotFound, gin.H{
					"status":  404,
					"message": "Publisher not found",
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  500,
					"message": "Failed to update publisher",
				})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  200,
			"message": "Publisher updated successfully",
			"data":    publisher,
		})
	}
}

func DeletePublisherHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := publisherIDFromContext(c)
		if !ok {
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  401,
				"message": "Unauthorized",
			})
			return
		}

		err := service.DeletePublisher(db, id, userID, ip, userAgent)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  404,
					"message": "Publisher not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Failed to delete publisher",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  200,
			"message": "Publisher deleted successfully",
		})
	}
}
