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

		category, err := service.CreateCategory(
			db,
			req.CategoryName,
			req.Description,
		)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":   http.StatusCreated,
			"message":  "category created successfully",
			"category": category,
		})
	}
}

func GetCategoriesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		categories, err := service.GetAllCategories(db)

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

		category, err := service.GetCategoryByID(db, categoryID)

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

		category, err := service.UpdateCategory(
			db,
			categoryID,
			req.CategoryName,
			req.Description,
		)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status": http.StatusNotFound,
					"error":  "category not found",
				})
				return
			}

			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   http.StatusOK,
			"message":  "category updated successfully",
			"category": category,
		})
	}
}

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

		err = service.DeleteCategory(db, categoryID)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status": http.StatusNotFound,
					"error":  "category not found",
				})
				return
			}

			c.JSON(http.StatusBadRequest, gin.H{
				"status": http.StatusBadRequest,
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "category deleted successfully",
		})
	}
}
