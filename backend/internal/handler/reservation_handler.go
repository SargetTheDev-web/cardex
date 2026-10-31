// internal/handler/reservation_handler.go

package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func reservationIDParam(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("reservation_id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, service.ErrInvalidReservationRequest
	}
	return id, nil
}

func reservationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidReservationRequest),
		errors.Is(err, service.ErrInvalidBorrowRequest):
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error", "message": "Invalid reservation request",
		})
	case errors.Is(err, service.ErrBorrowerNotEligible):
		c.JSON(http.StatusForbidden, gin.H{
			"status": "error", "message": "Account is not eligible to borrow",
		})
	case errors.Is(err, service.ErrReservationNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error", "message": "Reservation not found",
		})
	case errors.Is(err, service.ErrReservationNotActive),
		errors.Is(err, service.ErrReservationNotNextInQueue):
		c.JSON(http.StatusConflict, gin.H{
			"status": "error", "message": err.Error(),
		})
	case errors.Is(err, service.ErrReservationConflict),
		errors.Is(err, service.ErrNoAvailableCopy),
		errors.Is(err, service.ErrCopyUnavailable):
		c.JSON(http.StatusConflict, gin.H{
			"status": "error", "message": err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error", "message": "Unable to process reservation",
		})
	}
}

func CreateReservationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("user_id")
		if userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error", "message": "Authentication required",
			})
			return
		}

		var input service.CreateReservationInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error", "message": "Invalid reservation request",
			})
			return
		}

		result, err := service.CreateReservation(
			db, userID, input, c.ClientIP(), c.GetHeader("User-Agent"),
		)
		if err != nil {
			reservationError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status": "success", "message": result.Message, "data": result.Reservation,
		})
	}
}

func GetMyReservationsHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("user_id")
		if userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error", "message": "Authentication required",
			})
			return
		}

		reservations, err := service.GetMyReservations(db, userID)
		if err != nil {
			reservationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success", "data": reservations,
		})
	}
}

func CancelMyReservationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt("user_id")
		id, err := reservationIDParam(c)
		if err != nil {
			reservationError(c, err)
			return
		}
		if userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error", "message": "Authentication required",
			})
			return
		}

		if err := service.CancelMyReservation(
			db, id, userID, c.ClientIP(), c.GetHeader("User-Agent"),
		); err != nil {
			reservationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success", "message": "Reservation cancelled",
		})
	}
}

func GetReservationQueueHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reservations, err := service.GetReservationQueue(db)
		if err != nil {
			reservationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success", "data": reservations,
		})
	}
}

func ApproveReservationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		staffID := c.GetInt("user_id")
		id, err := reservationIDParam(c)
		if err != nil {
			reservationError(c, err)
			return
		}

		result, err := service.ApproveReservation(
			db, id, staffID, c.ClientIP(), c.GetHeader("User-Agent"),
		)
		if err != nil {
			reservationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success", "message": result.Message, "data": result.Reservation,
		})
	}
}

func RejectReservationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		staffID := c.GetInt("user_id")
		id, err := reservationIDParam(c)
		if err != nil {
			reservationError(c, err)
			return
		}

		if err := service.RejectReservation(
			db, id, staffID, c.ClientIP(), c.GetHeader("User-Agent"),
		); err != nil {
			reservationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success", "message": "Reservation rejected",
		})
	}
}

func FulfillReservationHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		staffID := c.GetInt("user_id")
		id, err := reservationIDParam(c)
		if err != nil {
			reservationError(c, err)
			return
		}

		var input service.FulfillReservationInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error", "message": "A valid due_date is required",
			})
			return
		}

		result, err := service.FulfillReservation(
			db, id, staffID, input, c.ClientIP(), c.GetHeader("User-Agent"),
		)
		if err != nil {
			reservationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success", "message": result.Message, "data": result,
		})
	}
}
