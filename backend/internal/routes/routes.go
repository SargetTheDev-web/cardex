// internal/routes/routes.go

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
	// RBAC Implementation
	// --------------------------------------------------

	require := func(permissionCode string) gin.HandlerFunc {
		return middleware.RequirePermission(db, permissionCode)
	}

	// --------------------------------------------------
	// Authenticated routes
	// --------------------------------------------------

	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware(db))
	protected.GET(
		"/profile",
		require("PROFILE_VIEW_OWN"),
		handler.GetProfileHandler(db),
	)

	// --------------------------------------------------
	// User profile
	// --------------------------------------------------

	protected.POST(
		"/logout",
		handler.LogoutHandler(db),
	)

	protected.POST(
		"/profile/update",
		require("PROFILE_UPDATE_OWN"),
		handler.RequestProfileUpdateHandler(db),
	)

	protected.POST(
		"/auth/change-password",
		require("PROFILE_CHANGE_PASSWORD"),
		handler.ChangePasswordHandler(db),
	)

	protected.POST("/auth/change-pin",
		require("PROFILE_CHANGE_PIN"),
		handler.ChangePINHandler(db),
	)

	protected.GET(
		"/categories",
		require("CATEGORY_READ"),
		handler.GetCategoriesHandler(db),
	)

	protected.GET(
		"/categories/:id",
		require("CATEGORY_READ"),
		handler.GetCategoryHandler(db),
	)

	protected.GET(
		"/authors",
		require("AUTHOR_READ"),
		handler.GetAuthorsHandler(db),
	)

	protected.GET(
		"/authors/:id",
		require("AUTHOR_READ"),
		handler.GetAuthorHandler(db),
	)

	protected.GET(
		"/publishers",
		require("PUBLISHER_READ"),
		handler.GetPublishersHandler(db),
	)

	protected.GET(
		"/publishers/:id",
		require("PUBLISHER_READ"),
		handler.GetPublisherHandler(db),
	)
	protected.GET(
		"/books",
		require("BOOK_READ"),
		handler.GetBooksHandler(db),
	)
	protected.GET(
		"/books/:id",
		require("BOOK_READ"),
		handler.GetBookHandler(db),
	)

	protected.GET(
		"/book-copies",
		require("BOOK_COPY_READ"),
		handler.GetBookCopiesHandler(db),
	)

	protected.GET(
		"/book-copies/:id",
		require("BOOK_COPY_READ"),
		handler.GetBookCopyHandler(db),
	)

	protected.POST(
		"/reservations",
		require("RESERVATION_CREATE"),
		handler.CreateReservationHandler(db),
	)

	protected.GET(
		"/reservations/mine",
		require("RESERVATION_READ"),
		handler.GetMyReservationsHandler(db),
	)

	protected.DELETE(
		"/reservations/:reservation_id",
		require("RESERVATION_CANCEL"),
		handler.CancelMyReservationHandler(db),
	)

	// Librarian
	librarian := protected.Group("/")
	// librarian.Use(middleware.RequireLibrarianOrAdmin(db))

	// --------------------------------------------------
	// User Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.GET(
		"/users",
		require("USER_READ"),
		handler.GetUsersHandler(db),
	)

	librarian.POST(
		"/users",
		require("USER_CREATE"),
		handler.CreateUserHandler(db),
	)

	librarian.PUT(
		"/users/:id",
		require("USER_UPDATE"),
		handler.UpdateUserHandler(db),
	)

	librarian.PATCH(
		"/users/:id/status",
		require("USER_DEACTIVATE"),
		handler.ChangeUserStatusHandler(db),
	)

	librarian.POST(
		"/profile/change-request/:id/approve",
		require("USER_CREATE"),
		handler.ApproveProfileChangeHandler(db),
	)

	librarian.POST(
		"/profile/change-request/:id/reject",
		require("USER_CREATE"),
		handler.RejectProfileChangeHandler(db),
	)

	librarian.GET(
		"/profile/change-requests",
		require("USER_READ"),
		handler.GetPendingProfileChangeRequestsHandler(db),
	)

	// --------------------------------------------------
	// Category Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/categories",
		require("CATEGORY_CREATE"),
		handler.CreateCategoryHandler(db),
	)

	librarian.PUT(
		"/categories/:id",
		require("CATEGORY_UPDATE"),
		handler.UpdateCategoryHandler(db),
	)

	librarian.DELETE(
		"/categories/:id",
		require("CATEGORY_DELETE"),
		handler.DeleteCategoryHandler(db),
	)

	// --------------------------------------------------
	// Author Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/authors",
		require("AUTHOR_CREATE"),
		handler.CreateAuthorHandler(db),
	)

	librarian.PUT(
		"/authors/:id",
		require("AUTHOR_UPDATE"),
		handler.UpdateAuthorHandler(db),
	)

	librarian.DELETE(
		"/authors/:id",
		require("AUTHOR_DELETE"),
		handler.DeleteAuthorHandler(db),
	)

	// --------------------------------------------------
	// Publisher Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/publishers",
		require("PUBLISHER_CREATE"),
		handler.CreatePublisherHandler(db),
	)

	librarian.PUT(
		"/publishers/:id",
		require("PUBLISHER_UPDATE"),
		handler.UpdatePublisherHandler(db),
	)

	librarian.DELETE(
		"/publishers/:id",
		require("PUBLISHER_DELETE"),
		handler.DeletePublisherHandler(db),
	)

	// --------------------------------------------------
	// Book Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.GET(
		"/summary",
		require("DASHBOARD_VIEW_ANALYTICS"),
		handler.GetBookCatalogSummaryHandler(db),
	)

	librarian.GET(
		"/books/isbn/validate",
		handler.ValidateBookISBNHandler(db),
	)

	librarian.POST(
		"/books",
		require("BOOK_CREATE"),
		handler.CreateBookHandler(db),
	)

	librarian.PUT(
		"/books/:id",
		require("BOOK_UPDATE"),
		handler.UpdateBookHandler(db),
	)

	librarian.DELETE(
		"/books/:id",
		require("BOOK_DELETE"),
		handler.DeleteBookHandler(db),
	)

	// --------------------------------------------------
	// Book Copy Management - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	librarian.POST(
		"/book-copies",
		require("BOOK_COPY_CREATE"),
		handler.CreateBookCopyHandler(db),
	)

	librarian.PUT(
		"/book-copies/:id",
		require("BOOK_COPY_UPDATE"),
		handler.UpdateBookCopyHandler(db),
	)

	librarian.PATCH(
		"/book-copies/:id/status",
		require("BOOK_COPY_UPDATE"),
		handler.UpdateBookCopyStatusHandler(db),
	)

	// --------------------------------------------------
	// System Parameter - ADMIN
	// --------------------------------------------------

	admin := protected.Group("/")

	router.GET(
		"/system-parameters/:key",
		handler.GetSystemParameterHandler(db),
	)

	admin.PUT(
		"/system-parameters/:key",
		require("SYSTEM_PARAMETERS_UPDATE"),
		handler.UpdateSystemParameterHandler(db),
	)

	admin.GET(
		"/system-parameters",
		require("SYSTEM_PARAMETERS_VIEW"),
		handler.GetAllSystemParametersHandler(db),
	)

	admin.GET(
		"/audit/dashboard",
		require("AUDIT_VIEW"),
		handler.GetAuditDashboardHandler(db),
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

	circulation.POST(
		"/borrow",
		require("LOAN_CREATE"),
		handler.BorrowBookHandler(db),
	)

	circulation.POST(
		"/return",
		require("LOAN_RETURN"),
		handler.ReturnBookHandler(db),
	)

	circulation.GET(
		"/transactions/recent",
		require("LOAN_READ"),
		handler.GetRecentTransactionsHandler(db),
	)

	circulation.GET("/reservations",
		require("RESERVATION_READ"),
		handler.GetReservationQueueHandler(db),
	)

	circulation.POST(
		"/reservations/:reservation_id/approve",
		require("RESERVATION_APPROVE"),
		handler.ApproveReservationHandler(db),
	)

	circulation.POST(
		"/reservations/:reservation_id/reject",
		require("RESERVATION_REJECT"),
		handler.RejectReservationHandler(db),
	)

	circulation.POST(
		"/reservations/:reservation_id/fulfill",
		require("LOAN_CREATE"),
		handler.FulfillReservationHandler(db),
	)

	circulation.GET("/fines",
		handler.GetFinesHandler(db),
	)

	circulation.PATCH(
		"/fines/:fine_id/status",
		handler.UpdateFineStatusHandler(db),
	)

	// --------------------------------------------------
	// Reports and Analytics Module - LIBRARIAN OR ADMIN
	// --------------------------------------------------

	protected.GET(
		"/reports/circulation-summary",
		require("DASHBOARD_VIEW	_ANALYTICS"),
		handler.GetCirculationSummaryHandler(db),
	)

	protected.GET(
		"/reports/overdue",
		require("DASHBOARD_VIEW_ANALYTICS"),
		handler.GetOverdueReportHandler(db),
	)

	protected.GET(
		"/reports/reservations",
		require("DASHBOARD_VIEW_ANALYTICS"),
		handler.GetReservationsReportHandler(db),
	)

	protected.GET(
		"/reports/inventory",
		require("DASHBOARD_VIEW_ANALYTICS"),
		handler.GetInventoryReportHandler(db),
	)

	librarian.GET(
		"/get-all-notifications",
		handler.GetAllNotificationsHandler(db),
	)

	notifications := protected.Group("/notifications")
	{
		notifications.GET("", require("NOTIFICATION_READ"),
			handler.GetNotificationsHandler(db),
		)
		notifications.GET(
			"/unread",
			require("NOTIFICATION_READ"),
			handler.GetUnreadNotificationsHandler(db),
		)

		notifications.GET(
			"/unread-count",
			require("NOTIFICATION_READ"),
			handler.GetUnreadNotificationCountHandler(db))

		notifications.PATCH(
			"/:id/read",
			require("NOTIFICATION_UPDATE"),
			handler.MarkNotificationAsReadHandler(db))

		notifications.PATCH(
			"/read-all",
			require("NOTIFICATION_UPDATE"),
			handler.MarkAllNotificationsAsReadHandler(db),
		)

		notifications.DELETE(
			"/:id", require("NOTIFICATION_DELETE"),
			handler.DeleteNotificationHandler(db),
		)
	}

	rbac := protected.Group("/rbac")
	{
		rbac.GET(
			"/matrix",
			middleware.RequirePermission(db, "RBAC_VIEW"),
			handler.GetRBACMatrixHandler(db),
		)

		rbac.PUT(
			"/matrix",
			middleware.RequirePermission(db, "RBAC_VIEW"),
			middleware.RequirePermission(db, "RBAC_PERMISSION_ASSIGN"),
			middleware.RequirePermission(db, "RBAC_PERMISSION_REVOKE"),
			handler.UpdateRBACMatrixHandler(db),
		)
	}
}
