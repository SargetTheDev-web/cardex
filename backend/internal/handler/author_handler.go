// internal/handler/author_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthorRequest struct {
	AuthorName string `json:"author_name" binding:"required"`
	Biography  string `json:"biography"`
}

func CreateAuthorHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AuthorRequest
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

		author, err := service.CreateAuthor(
			db, req.AuthorName, req.Biography, userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, service.ErrInvalidAuthor) {
				c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": err.Error()})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "failed to create author"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"message": "author created successfully",
			"author":  author,
		})
	}
}

func GetAuthorsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		authors, err := service.GetAllAuthors(db, userID, ip, userAgent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "failed to retrieve authors"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"authors": authors,
		})
	}
}

func GetAuthorHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorID, err := strconv.Atoi(c.Param("id"))
		if err != nil || authorID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": "invalid author ID"})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		author, err := service.GetAuthorByID(db, authorID, userID, ip, userAgent)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "author not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "failed to retrieve author"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": http.StatusOK,
			"author": author,
		})
	}
}

func UpdateAuthorHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorID, err := strconv.Atoi(c.Param("id"))
		if err != nil || authorID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": "invalid author ID"})
			return
		}

		var req AuthorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": "invalid request body"})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		author, err := service.UpdateAuthor(
			db, authorID, req.AuthorName, req.Biography, userID, ip, userAgent,
		)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidAuthor):
				c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": err.Error()})
			case errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "author not found"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "failed to update author"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "author updated successfully",
			"author":  author,
		})
	}
}

func DeleteAuthorHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorID, err := strconv.Atoi(c.Param("id"))
		if err != nil || authorID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": "invalid author ID"})
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			return
		}

		err = service.DeleteAuthor(db, authorID, userID, ip, userAgent)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "author not found"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "failed to delete author"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "author deleted successfully",
		})
	}
}
