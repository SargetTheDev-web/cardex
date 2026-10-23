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
	router.POST("/auth/google", handler.GoogleAuthHandler(db))
	router.POST("/auth/google/register/complete", handler.GoogleRegisterCompleteHandler(db))
	router.POST("/auth/email/check", handler.CheckEmailHandler(db))
	router.POST("/login/qr", handler.QRLoginHandler(db))
	router.POST("/register/request", handler.RegisterRequestHandler(db))
	router.POST("/register/verify", handler.VerifyCodeHandler(db))
	router.POST("/register/complete", handler.CompleteRegistrationHandler(db))
	router.GET("/enums/courses", handler.GetCourseProgramsHandler(db))
	router.GET("/enums/suffixes", handler.GetSuffixExtensionsHandler(db))
	router.GET("/public/books", handler.GetPublicBooksHandler(db))
	router.POST("/public/search", handler.PublicBookSearchHandler(db))

	// Kiosk scanning: authenticates using the kiosk credential.
	router.POST("/attendance/kiosk/scan", handler.AttendanceScanHandler(db))

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
	protected.GET(
		"/books", handler.GetBooksHandler(db),
	)
	protected.GET(
		"/books/:id", handler.GetBookHandler(db),
	)

	protected.GET(
		"/book-copies",
		handler.GetBookCopiesHandler(db),
	)

	protected.GET(
		"/book-copies/:id", handler.GetBookCopyHandler(db),
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

	// --------------------------------------------------
	// Book Management - ADMIN ONLY
	// --------------------------------------------------

	admin.GET(
		"/books/isbn/validate",
		handler.ValidateBookISBNHandler(db),
	)

	admin.POST(
		"/books",
		handler.CreateBookHandler(db),
	)

	admin.PUT(
		"/books/:id",
		handler.UpdateBookHandler(db),
	)

	admin.DELETE(
		"/books/:id", handler.DeleteBookHandler(db),
	)

	// --------------------------------------------------
	// Book Copy Management - ADMIN ONLY
	// --------------------------------------------------

	admin.POST(
		"/book-copies", handler.CreateBookCopyHandler(db),
	)

	admin.PUT(
		"/book-copies/:id", handler.UpdateBookCopyHandler(db),
	)

	admin.PATCH(
		"/book-copies/:id/status", handler.UpdateBookCopyStatusHandler(db),
	)

	// --------------------------------------------------
	// System Parameter - ADMIN
	// --------------------------------------------------

	admin.GET(
		"/system-parameters/:key",
		handler.GetSystemParameterHandler(db),
	)

	admin.PUT(
		"/system-parameters/:key",
		handler.UpdateSystemParameterHandler(db),
	)

	admin.GET(
		"/system-parameters",
		handler.GetAllSystemParametersHandler(db),
	)

	// --------------------------------------------------
	// Attendance kiosk - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	kiosk := protected.Group("/attendance/kiosk")
	kiosk.Use(middleware.RequireLibrarianOrAdmin(db))

	kiosk.POST(
		"/activate",
		handler.ActivateAttendanceKioskHandler(db),
	)

	kiosk.POST(
		"/deactivate",
		handler.DeactivateAttendanceKioskHandler(db),
	)
}
