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
	router.GET("/enums/courses", handler.GetCourseProgramsHandler(db))
	router.GET("/enums/suffixes", handler.GetSuffixExtensionsHandler(db))

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

	protected.GET(
		"/authors",
		handler.GetAuthorsHandler(db),
	)

	protected.GET(
		"/authors/:id",
		handler.GetAuthorHandler(db),
	)

	protected.GET(
		"/publishers",
		handler.GetPublishersHandler(db),
	)

	protected.GET(
		"/publishers/:id",
		handler.GetPublisherHandler(db),
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

	// --------------------------------------------------
	// Author Management - ADMIN ONLY
	// --------------------------------------------------

	admin.POST(
		"/authors",
		handler.CreateAuthorHandler(db),
	)

	admin.PUT("/authors/:id",
		handler.UpdateAuthorHandler(db),
	)

	admin.DELETE("/authors/:id",
		handler.DeleteAuthorHandler(db),
	)

	// --------------------------------------------------
	// Publisher Management - ADMIN ONLY
	// --------------------------------------------------

	admin.POST(
		"/publishers",
		handler.CreatePublisherHandler(db),
	)

	admin.PUT(
		"/publishers/:id",
		handler.UpdatePublisherHandler(db),
	)

	admin.DELETE(
		"/publishers/:id",
		handler.DeletePublisherHandler(db),
	)
}
