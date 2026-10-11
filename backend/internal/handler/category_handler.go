// internal/handler/category_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CategoryRequest struct {
	CategoryName string `json:"category_name" binding:"required"`
	Description  string `json:"description"`
}

// Extract authenticated user ID and request metadata for auditing.
func getCategoryAuditMetadata(c *gin.Context) (int, string, string, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": http.StatusUnauthorized,
			"error":  "authenticated user ID not found",
		})
		return 0, "", "", false
	}

	userID, ok := userIDValue.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": http.StatusInternalServerError,
			"error":  "invalid authenticated user ID",
		})
		return 0, "", "", false
	}

	return userID, c.ClientIP(), c.Request.UserAgent(), true
}

// Convert service errors into appropriate HTTP responses.
func respondCategoryMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCategory):
		c.JSON(http.StatusBadRequest, gin.H{
			"status": http.StatusBadRequest,
			"error":  err.Error(),
		})

	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status": http.StatusNotFound,
			"error":  "category not found",
		})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": http.StatusInternalServerError,
			"error":  "category operation failed",
		})
	}
}

// POST /categories
func CreateCategoryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CategoryRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request body",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		category, err := service.CreateCategory(
			db,
			req.CategoryName,
			req.Description,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondCategoryMutationError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":   http.StatusCreated,
			"message":  "category created successfully",
			"category": category,
		})
	}
}

// GET /categories

func GetCategoriesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		categories, err := service.GetAllCategories(
			db,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve categories",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     http.StatusOK,
			"categories": categories,
		})
	}
}

// GET /categories/:id

func GetCategoryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID, err := strconv.Atoi(c.Param("id"))
		if err != nil || categoryID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid category ID",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		category, err := service.GetCategoryByID(
			db,
			categoryID,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status": http.StatusNotFound,
					"error":  "category not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve category",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   http.StatusOK,
			"category": category,
		})
	}
}

// PUT /categories/:id
func UpdateCategoryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID, err := strconv.Atoi(c.Param("id"))
		if err != nil || categoryID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid category ID",
			})
			return
		}

		var req CategoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid request body",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		category, err := service.UpdateCategory(
			db,
			categoryID,
			req.CategoryName,
			req.Description,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondCategoryMutationError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   http.StatusOK,
			"message":  "category updated successfully",
			"category": category,
		})
	}
}

// DELETE /categories/:id
func DeleteCategoryHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID, err := strconv.Atoi(c.Param("id"))
		if err != nil || categoryID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  "invalid category ID",
			})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		err = service.DeleteCategory(
			db,
			categoryID,
			userID,
			ip,
			userAgent,
		)
		if err != nil {
			respondCategoryMutationError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "category deleted successfully",
		})
	}
}
