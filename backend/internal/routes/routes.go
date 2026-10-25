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

	// Librarian
	librarian := protected.Group("/")
	librarian.Use(middleware.RequireLibrarianOrAdmin(db))

	// --------------------------------------------------
	// User Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.GET(
		"/users",
		handler.GetUsersHandler(db),
	)

	librarian.POST(
		"/users",
		handler.CreateUserHandler(db),
	)

	librarian.PUT(
		"/users/:id",
		handler.UpdateUserHandler(db),
	)

	librarian.PATCH(
		"/users/:id/status",
		handler.ChangeUserStatusHandler(db),
	)

	librarian.POST(
		"/profile/change-request/:id/approve",
		handler.ApproveProfileChangeHandler(db),
	)

	librarian.POST(
		"/profile/change-request/:id/reject",
		handler.RejectProfileChangeHandler(db),
	)

	librarian.GET(
		"/profile/change-requests",
		handler.GetPendingProfileChangeRequestsHandler(db),
	)

	// --------------------------------------------------
	// Category Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/categories",
		handler.CreateCategoryHandler(db),
	)

	librarian.PUT(
		"/categories/:id",
		handler.UpdateCategoryHandler(db),
	)

	librarian.DELETE(
		"/categories/:id",
		handler.DeleteCategoryHandler(db),
	)

	// --------------------------------------------------
	// Author Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/authors",
		handler.CreateAuthorHandler(db),
	)

	librarian.PUT("/authors/:id",
		handler.UpdateAuthorHandler(db),
	)

	librarian.DELETE("/authors/:id",
		handler.DeleteAuthorHandler(db),
	)

	// --------------------------------------------------
	// Publisher Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/publishers",
		handler.CreatePublisherHandler(db),
	)

	librarian.PUT(
		"/publishers/:id",
		handler.UpdatePublisherHandler(db),
	)

	librarian.DELETE(
		"/publishers/:id",
		handler.DeletePublisherHandler(db),
	)

	// --------------------------------------------------
	// Book Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.GET(
		"/summary", handler.GetBookCatalogSummaryHandler(db),
	)

	librarian.GET(
		"/books/isbn/validate",
		handler.ValidateBookISBNHandler(db),
	)

	librarian.POST(
		"/books",
		handler.CreateBookHandler(db),
	)

	librarian.PUT(
		"/books/:id",
		handler.UpdateBookHandler(db),
	)

	librarian.DELETE(
		"/books/:id", handler.DeleteBookHandler(db),
	)

	// --------------------------------------------------
	// Book Copy Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/book-copies", handler.CreateBookCopyHandler(db),
	)

	librarian.PUT(
		"/book-copies/:id", handler.UpdateBookCopyHandler(db),
	)

	librarian.PATCH(
		"/book-copies/:id/status", handler.UpdateBookCopyStatusHandler(db),
	)

	// --------------------------------------------------
	// System Parameter - ADMIN
	// --------------------------------------------------

	admin := protected.Group("/")
	admin.Use(middleware.RequireAdmin(db))

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

	// --------------------------------------------------
	// Circulation Module - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	circulation := protected.Group("/circulation")
	circulation.Use(middleware.RequireLibrarianOrAdmin(db))

	circulation.POST(
		"/borrow", handler.BorrowBookHandler(db),
	)

	circulation.POST(
		"/return", handler.ReturnBookHandler(db),
	)

	circulation.GET(
		"/transactions/recent",
		handler.GetRecentTransactionsHandler(db),
	)

}
