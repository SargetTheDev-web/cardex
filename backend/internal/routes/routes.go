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

	auth := router.Group("/auth")
	{
		auth.POST(
			"/forgot-password",
			handler.ForgotPasswordHandler(db),
		)

		auth.POST(
			"/reset-password",
			handler.ResetPasswordHandler(db),
		)
	}

	// --------------------------------------------------
	// Authenticated routes
	// --------------------------------------------------

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(db))
	protected.GET("/profile", handler.GetProfileHandler(db))

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

	protected.POST(
		"/auth/change-password",
		handler.ChangePasswordHandler(db),
	)

	protected.POST("/auth/change-pin",
		handler.ChangePINHandler(db),
	)

	protected.GET(
		"/categories",
		handler.GetCategoriesHandler(db),
	)

	protected.GET(
		"/categories/:id",
		handler.GetCategoryHandler(db),
	)

	// ADMIN ONLY
	admin := protected.Group("/")
	admin.Use(middleware.RequireAdmin(db))

	// --------------------------------------------------
	// User Management - ADMIN ONLY
	// --------------------------------------------------

	admin.GET(
		"/users",
		handler.GetUsersHandler(db),
	)

	admin.POST(
		"/users",
		handler.CreateUserHandler(db),
	)

	admin.PUT(
		"/users/:id",
		handler.UpdateUserHandler(db),
	)

	admin.PATCH(
		"/users/:id/status",
		handler.ChangeUserStatusHandler(db),
	)

	admin.POST(
		"/profile/change-request/:id/approve",
		handler.ApproveProfileChangeHandler(db),
	)

	admin.POST(
		"/profile/change-request/:id/reject",
		handler.RejectProfileChangeHandler(db),
	)

	admin.GET(
		"/profile/change-requests",
		handler.GetPendingProfileChangeRequestsHandler(db),
	)

	// --------------------------------------------------
	// Category Management - ADMIN ONLY
	// --------------------------------------------------

	admin.POST(
		"/categories",
		handler.CreateCategoryHandler(db),
	)

	admin.PUT(
		"/categories/:id",
		handler.UpdateCategoryHandler(db),
	)

	admin.DELETE(
		"/categories/:id",
		handler.DeleteCategoryHandler(db),
	)
}
