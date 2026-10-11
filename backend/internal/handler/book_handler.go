// internal/handler/book_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	model "backend/internal/models"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func bookIDFromContext(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  http.StatusBadRequest,
			"message": "Invalid book ID",
		})
		return 0, false
	}

	return id, true
}

func CreateBookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "Unauthorized",
			})
			return
		}

		var book model.Book
		if err := c.ShouldBindJSON(&book); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
			})
			return
		}

		createdBook, err := service.CreateBook(
			db, &book, userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, service.ErrInvalidBook) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"message": err.Error(),
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to create book",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  http.StatusCreated,
			"message": "Book created successfully",
			"data":    createdBook,
		})
	}
}

func GetBooksHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "Unauthorized",
			})
			return
		}

		books, err := service.GetAllBooks(
			db, userID, ip, userAgent,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to retrieve books",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Books retrieved successfully",
			"data":    books,
		})
	}
}

func GetBookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, ok := bookIDFromContext(c)
		if !ok {
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "Unauthorized",
			})
			return
		}

		book, err := service.GetBookByID(
			db, bookID, userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  http.StatusNotFound,
					"message": "Book not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to retrieve book",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Book retrieved successfully",
			"data":    book,
		})
	}
}

func UpdateBookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, ok := bookIDFromContext(c)
		if !ok {
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "Unauthorized",
			})
			return
		}

		var book model.Book
		if err := c.ShouldBindJSON(&book); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
			})
			return
		}

		updatedBook, err := service.UpdateBook(
			db, bookID, &book, userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, service.ErrInvalidBook) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  http.StatusBadRequest,
					"message": err.Error(),
				})
				return
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  http.StatusNotFound,
					"message": "Book not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to update book",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Book updated successfully",
			"data":    updatedBook,
		})
	}
}

func DeleteBookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookID, ok := bookIDFromContext(c)
		if !ok {
			return
		}

		userID, ip, userAgent, ok := getCategoryAuditMetadata(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "Unauthorized",
			})
			return
		}

		err := service.DeleteBook(
			db, bookID, userID, ip, userAgent,
		)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"status":  http.StatusNotFound,
					"message": "Book not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to delete book",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Book deleted successfully",
		})
	}
}
