// internal/service/reservation_service.go

package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrInvalidReservationRequest = errors.New("invalid reservation request")
	ErrReservationNotFound       = errors.New("reservation not found")
	ErrReservationNotActive      = errors.New("reservation is not active")
	ErrReservationConflict       = errors.New("user already has an active reservation for this book")
	ErrNoAvailableCopy           = errors.New("no available copy for this book")
	ErrReservationNotNextInQueue = errors.New("reservation is not next in the queue")
)

type CreateReservationInput struct {
	BookID              int    `json:"book_id" binding:"required,gt=0"`
	PreferredPickupDate string `json:"preferred_pickup_date" binding:"required"`
	Notes               string `json:"notes"`
}

type ReservationResult struct {
	Reservation model.Reservation `json:"reservation"`
	Message     string            `json:"message"`
}

func CreateReservation(
	db *gorm.DB,
	userID int,
	input CreateReservationInput,
	ip string,
	userAgent string,
) (*ReservationResult, error) {
	notes := strings.TrimSpace(input.Notes)

	if userID <= 0 || input.BookID <= 0 {
		return nil, ErrInvalidReservationRequest
	}

	pickupDate, err := time.Parse("2006-01-02", strings.TrimSpace(input.PreferredPickupDate))
	if err != nil {
		return nil, ErrInvalidReservationRequest
	}

	var result model.Reservation

	err = db.Transaction(func(tx *gorm.DB) error {
		account, err := repository.GetBorrowerAccount(tx, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBorrowerNotEligible
			}
			return err
		}

		if account.StatusCode != "ACTIVE" ||
			(account.ExpirationDate != nil && !account.ExpirationDate.After(time.Now())) {
			return ErrBorrowerNotEligible
		}

		switch account.RoleCode {
		case "STUDENT", "FACULTY":
		default:
			return ErrBorrowerNotEligible
		}

		var bookExists bool
		if err := tx.Raw(
			`SELECT EXISTS (SELECT 1 FROM book WHERE book_id = ?)`,
			input.BookID,
		).Scan(&bookExists).Error; err != nil {
			return err
		}
		if !bookExists {
			return ErrInvalidReservationRequest
		}

		var today time.Time
		if err := tx.Raw(`SELECT CURRENT_DATE`).Scan(&today).Error; err != nil {
			return err
		}
		if pickupDate.Before(today) {
			return ErrInvalidReservationRequest
		}

		var queuePosition int
		if err := tx.Raw(`
			SELECT COUNT(*) + 1
			FROM reservation
			WHERE book_id = ?
			  AND status = 'PENDING'
		`, input.BookID).Scan(&queuePosition).Error; err != nil {
			return err
		}

		reservation := model.Reservation{
			BookID:              input.BookID,
			UserID:              userID,
			PreferredPickupDate: pickupDate,
			Status:              "PENDING",
			QueuePosition:       &queuePosition,
		}
		if notes != "" {
			reservation.Reason = &notes
		}

		if err := repository.CreateReservation(tx, &reservation); err != nil {
			return fmt.Errorf("%w: %v", ErrReservationConflict, err)
		}

		if err := writeCirculationAudit(
			tx,
			userID,
			"CREATE",
			fmt.Sprintf(
				"Created reservation ID %d for book ID %d; preferred pickup date %s",
				reservation.ReservationID,
				reservation.BookID,
				pickupDate.Format("2006-01-02"),
			),
			"", // No client IP available in this service call yet.
			"", // No User-Agent available in this service call yet.
		); err != nil {
			return err
		}

		result = reservation
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ReservationResult{
		Reservation: result,
		Message:     "Reservation submitted successfully",
	}, nil
}

func ApproveReservation(
	db *gorm.DB,
	reservationID int64,
	staffID int,
	ip string,
	userAgent string,
) (*ReservationResult, error) {
	if reservationID <= 0 || staffID <= 0 {
		return nil, ErrInvalidReservationRequest
	}

	var result model.Reservation

	err := db.Transaction(func(tx *gorm.DB) error {
		reservation, err := repository.LockReservationByID(tx, reservationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrReservationNotFound
			}
			return err
		}

		if reservation.Status != "PENDING" {
			return ErrReservationNotActive
		}

		// Enforce FIFO among pending requests for the same book.
		var earlierPending bool
		if err := tx.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM reservation
				WHERE book_id = ?
				  AND status = 'PENDING'
				  AND (
						reservation_date < ?
						OR (
							reservation_date = ?
							AND reservation_id < ?
						)
				  )
			)
		`,
			reservation.BookID,
			reservation.ReservationDate,
			reservation.ReservationDate,
			reservation.ReservationID,
		).Scan(&earlierPending).Error; err != nil {
			return err
		}
		if earlierPending {
			return ErrReservationNotNextInQueue
		}

		copy, err := repository.LockAvailableCopyForBook(tx, reservation.BookID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoAvailableCopy
			}
			return err
		}

		if err := repository.MarkCopyReserved(tx, copy.CopyID); err != nil {
			return err
		}

		if err := repository.ApproveReservation(
			tx,
			reservation.ReservationID,
			copy.CopyID,
		); err != nil {
			return err
		}

		if err := writeCirculationAudit(
			tx,
			staffID,
			"APPROVE",
			fmt.Sprintf(
				"Approved reservation ID %d for user ID %d; assigned copy ID %d",
				reservation.ReservationID,
				reservation.UserID,
				copy.CopyID,
			),
			ip,
			userAgent,
		); err != nil {
			return err
		}

		reservation.CopyID = &copy.CopyID
		reservation.Status = "NOTIFIED"
		result = reservation

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &ReservationResult{
		Reservation: result,
		Message:     "Reservation approved; copy is ready for pickup",
	}, nil
}

type FulfillReservationInput struct {
	DueDate     string `json:"due_date" binding:"required"`
	BorrowNotes string `json:"borrow_notes"`
}

func GetMyReservations(db *gorm.DB, userID int) ([]model.Reservation, error) {
	if userID <= 0 {
		return nil, ErrInvalidReservationRequest
	}

	var reservations []model.Reservation
	err := db.
		Where("user_id = ?", userID).
		Order("reservation_date DESC, reservation_id DESC").
		Find(&reservations).Error
	if reservations == nil {
		reservations = []model.Reservation{}
	}
	return reservations, err
}

func GetReservationQueue(db *gorm.DB) ([]model.Reservation, error) {
	// Expire reservations before returning the queue.
	if err := ExpireReservations(db); err != nil {
		return nil, err
	}

	var reservations []model.Reservation
	err := db.
		Where("status IN ('PENDING', 'NOTIFIED')").
		Order("reservation_date ASC, reservation_id ASC").
		Find(&reservations).Error
	if reservations == nil {
		reservations = []model.Reservation{}
	}
	return reservations, err
}

func CancelMyReservation(
	db *gorm.DB,
	reservationID int64,
	userID int,
	ip string,
	userAgent string,
) error {
	if reservationID <= 0 || userID <= 0 {
		return ErrInvalidReservationRequest
	}

	return db.Transaction(func(tx *gorm.DB) error {
		reservation, err := repository.LockReservationByID(tx, reservationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReservationNotFound
		}
		if err != nil {
			return err
		}
		if reservation.UserID != userID {
			return ErrReservationNotFound
		}
		if reservation.Status != "PENDING" && reservation.Status != "NOTIFIED" {
			return ErrReservationNotActive
		}

		if reservation.CopyID != nil {
			if err := repository.MarkCopyAvailable(tx, *reservation.CopyID); err != nil {
				return err
			}
		}
		if err := repository.CancelReservation(tx, reservationID); err != nil {
			return err
		}

		return writeCirculationAudit(
			tx,
			userID,
			"UPDATE",
			fmt.Sprintf("Cancelled reservation ID %d", reservationID),
			ip,
			userAgent,
		)
	})
}

func RejectReservation(
	db *gorm.DB,
	reservationID int64,
	staffID int,
	ip string,
	userAgent string,
) error {
	if reservationID <= 0 || staffID <= 0 {
		return ErrInvalidReservationRequest
	}

	return db.Transaction(func(tx *gorm.DB) error {
		reservation, err := repository.LockReservationByID(tx, reservationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReservationNotFound
		}
		if err != nil {
			return err
		}
		if reservation.Status != "PENDING" {
			return ErrReservationNotActive
		}

		if err := repository.CancelReservation(tx, reservationID); err != nil {
			return err
		}

		return writeCirculationAudit(
			tx,
			staffID,
			"UPDATE",
			fmt.Sprintf(
				"Rejected reservation ID %d for user ID %d",
				reservationID,
				reservation.UserID,
			),
			ip,
			userAgent,
		)
	})
}

func FulfillReservation(
	db *gorm.DB,
	reservationID int64,
	staffID int,
	input FulfillReservationInput,
	ip string,
	userAgent string,
) (*BorrowResult, error) {
	if reservationID <= 0 || staffID <= 0 {
		return nil, ErrInvalidReservationRequest
	}

	dueDateText := strings.TrimSpace(input.DueDate)
	parsedDueDate, err := time.Parse("2006-01-02", dueDateText)
	if err != nil {
		return nil, ErrInvalidBorrowRequest
	}

	notes := strings.TrimSpace(input.BorrowNotes)
	var result BorrowResult

	err = db.Transaction(func(tx *gorm.DB) error {
		// Release any reservations whose pickup windows have expired.
		if err := expireReservationsTx(tx); err != nil {
			return err
		}

		reservation, err := repository.LockReservationByID(tx, reservationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReservationNotFound
		}
		if err != nil {
			return err
		}
		if reservation.Status != "NOTIFIED" || reservation.CopyID == nil {
			return ErrReservationNotActive
		}

		var copy model.BookCopy
		err = tx.Raw(`
			SELECT *
			FROM book_copy
			WHERE copy_id = ?
			FOR UPDATE
		`, *reservation.CopyID).Scan(&copy).Error
		if err != nil {
			return err
		}
		if copy.CopyID == 0 || copy.CopyStatus != "RESERVED" {
			return ErrCopyUnavailable
		}

		account, err := repository.GetBorrowerAccount(tx, reservation.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBorrowerNotEligible
			}
			return err
		}
		if account.StatusCode != "ACTIVE" ||
			(account.ExpirationDate != nil && !account.ExpirationDate.After(time.Now())) {
			return ErrBorrowerNotEligible
		}
		if account.RoleCode != "STUDENT" && account.RoleCode != "FACULTY" {
			return ErrBorrowerNotEligible
		}

		loanDate, err := repository.GetDatabaseCurrentDate(tx)
		if err != nil {
			return err
		}
		loanDate = time.Date(
			loanDate.Year(), loanDate.Month(), loanDate.Day(),
			0, 0, 0, 0, time.UTC,
		)
		dueDate := time.Date(
			parsedDueDate.Year(), parsedDueDate.Month(), parsedDueDate.Day(),
			0, 0, 0, 0, time.UTC,
		)
		if dueDate.Before(loanDate) {
			return ErrInvalidBorrowRequest
		}

		loan := model.Loan{
			CopyID:       copy.CopyID,
			UserID:       reservation.UserID,
			LoanDate:     loanDate,
			DueDate:      dueDate,
			RenewalCount: 0,
			Status:       "BORROWED",
		}
		if notes != "" {
			loan.BorrowNotes = &notes
		}
		if err := repository.CreateLoan(tx, &loan); err != nil {
			return err
		}

		updated := tx.Model(&model.BookCopy{}).
			Where("copy_id = ? AND copy_status = 'RESERVED'", copy.CopyID).
			Update("copy_status", "BORROWED")
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrCopyUnavailable
		}

		updated = tx.Model(&model.Reservation{}).
			Where("reservation_id = ? AND status = 'NOTIFIED'", reservationID).
			Updates(map[string]interface{}{
				"status":      "COMPLETED",
				"expiry_date": nil,
			})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrReservationNotActive
		}

		barcode := ""
		if copy.Barcode != nil {
			barcode = *copy.Barcode
		}

		if err := writeCirculationAudit(
			tx,
			staffID,
			"CREATE",
			fmt.Sprintf(
				"Fulfilled reservation ID %d; created loan ID %d for user ID %d; copy ID %d",
				reservationID,
				loan.LoanID,
				reservation.UserID,
				copy.CopyID,
			),
			ip,
			userAgent,
		); err != nil {
			return err
		}

		result = BorrowResult{
			Loan:    loan,
			BookID:  copy.BookID,
			Barcode: barcode,
			Message: "Reservation fulfilled; book borrowed successfully",
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Call this from a scheduled job as well as queue/fulfillment operations.
func ExpireReservations(db *gorm.DB) error {
	return db.Transaction(expireReservationsTx)
}

func expireReservationsTx(tx *gorm.DB) error {
	return tx.Exec(`
		WITH expired AS (
			UPDATE reservation
			SET status = 'CANCELLED'
			WHERE status = 'NOTIFIED'
			  AND expiry_date IS NOT NULL
			  AND expiry_date <= CURRENT_TIMESTAMP
			RETURNING copy_id
		)
		UPDATE book_copy AS c
		SET copy_status = 'AVAILABLE'
		FROM expired AS e
		WHERE c.copy_id = e.copy_id
		  AND c.copy_status = 'RESERVED'
	`).Error
}
