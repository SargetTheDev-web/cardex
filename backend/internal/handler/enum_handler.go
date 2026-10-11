// internal/handler/enum_handler.go

package handler

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetCourseProgramsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		courses, err := service.GetCoursePrograms(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"message": "Failed to retrieve course programs",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  200,
			"message": "Course programs retrieved successfully",
			"data":    courses,
		})
	}
}
