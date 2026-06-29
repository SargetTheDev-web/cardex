package routes

import (
	"backend/internal/handler"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.POST("/login", handler.LoginHandler(db))

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())

	protected.GET("/profile", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "authorized",
		})
	})
}
