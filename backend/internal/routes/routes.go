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

	// --------------------------------------------------
	// Public authentication
	// --------------------------------------------------

	router.POST("/login", handler.LoginHandler(db))
	router.POST("/register/request", handler.RegisterRequestHandler(db))
	router.POST("/register/verify", handler.VerifyCodeHandler(db))
	router.POST("/register/complete", handler.CompleteRegistrationHandler(db))

	router.POST(
		"/forgot-password",
		handler.ForgotPasswordHandler(db),
	)

	router.POST(
		"/reset-password",
		handler.ResetPasswordHandler(db),
	)

	// --------------------------------------------------
	// Authenticated routes
	// --------------------------------------------------

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(db))

	protected.GET("/profile", func(c *gin.Context) {

		userID, exists := c.Get("user_id")

		if !exists {
			c.JSON(500, gin.H{
				"error": "user_id not found in context",
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "authorized",
			"user_id": userID,
		})
	})

	// --------------------------------------------------
	// User profile
	// --------------------------------------------------

	protected.POST(
		"/logout",
		handler.LogoutHandler(db),
	)

	protected.POST(
		"/profile/update",
		handler.RequestProfileUpdateHandler(db),
	)

	protected.GET(
		"/profile/change-request",
		handler.GetProfileChangeRequestHandler(db),
	)

	// ADMIN ONLY
	admin := protected.Group("/")
	admin.Use(middleware.RequireAdmin(db))

	admin.POST(
		"/profile/change-request/:id/approve",
		handler.ApproveProfileChangeHandler(db),
	)

	admin.POST(
		"/profile/change-request/:id/reject",
		handler.RejectProfileChangeHandler(db),
	)

}
