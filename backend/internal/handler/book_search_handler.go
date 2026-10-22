// internal/handler/book_search_handler.go

package handler

import (
	"net/http"
	"strings"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PublicBookSearchRequest struct {
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"top_k"`
}

func PublicBookSearchHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request PublicBookSearchRequest

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "A valid JSON body with a query is required",
			})
			return
		}

		request.Query = strings.TrimSpace(request.Query)
		if request.Query == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Search query cannot be empty",
			})
			return
		}

		if request.TopK == 0 {
			request.TopK = 10
		}

		if request.TopK < 1 || request.TopK > 10 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  http.StatusBadRequest,
				"message": "top_k must be between 1 and 10",
			})
			return
		}

		result, err := service.SearchPublicBooks(
			c.Request.Context(),
			db,
			request.Query,
			request.TopK,
		)
		if err != nil {
			// Do not expose internal database or AI service details.
			c.JSON(http.StatusBadGateway, gin.H{
				"status":  http.StatusBadGateway,
				"message": "Book search service is temporarily unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"message": "Book search completed",
			"data":    result,
		})
	}
}
