// internal/repository/reservation_repository.go

package repository

import (
	"errors"

	model "backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateReservation(
	tx *gorm.DB,
	reservation *model.Reservation,
) error {
	return tx.Create(reservation).Error
}

func GetMyReservations(
	db *gorm.DB,
	userID int,
) ([]model.Reservation, error) {
	var reservations []model.Reservation

	err := db.
		Where("user_id = ?", userID).
		Order("reservation_date DESC, reservation_id DESC").
		Find(&reservations).Error
	if err != nil {
		return nil, err
	}

	if reservations == nil {
		reservations = []model.Reservation{}
	}

	return reservations, nil
}

// LockReservationByID must be called inside a transaction.
func LockReservationByID(
	tx *gorm.DB,
	reservationID int64,
) (model.Reservation, error) {
	var reservation model.Reservation

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("reservation_id = ?", reservationID).
		First(&reservation).Error

	return reservation, err
}

// LockAvailableCopyForBook picks one available copy and locks it.
// SKIP LOCKED prevents concurrent approvals from selecting the same copy.
func LockAvailableCopyForBook(
	tx *gorm.DB,
	bookID int,
) (model.BookCopy, error) {
	var copy model.BookCopy

	err := tx.
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "SKIP LOCKED",
		}).
		Where("book_id = ? AND copy_status = ?", bookID, "AVAILABLE").
		Order("copy_id ASC").
		First(&copy).Error

	return copy, err
}

func MarkCopyReserved(
	tx *gorm.DB,
	copyID int,
) error {
	result := tx.
		Model(&model.BookCopy{}).
		Where("copy_id = ? AND copy_status = ?", copyID, "AVAILABLE").
		Update("copy_status", "RESERVED")

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("copy is no longer available")
	}

	return nil
}

func MarkCopyAvailable(
	tx *gorm.DB,
	copyID int,
) error {
	result := tx.
		Model(&model.BookCopy{}).
		Where("copy_id = ? AND copy_status = ?", copyID, "RESERVED").
		Update("copy_status", "AVAILABLE")

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("reserved copy could not be released")
	}

	return nil
}

// ApproveReservation assigns the copy and starts the 3-day pickup window.
// Call inside the same transaction that marks the copy RESERVED.
func ApproveReservation(
	tx *gorm.DB,
	reservationID int64,
	copyID int,
) error {
	result := tx.Exec(`
		UPDATE reservation
		SET status = 'NOTIFIED',
		    copy_id = ?,
		    expiry_date = CURRENT_TIMESTAMP + INTERVAL '3 days'
		WHERE reservation_id = ?
		  AND status = 'PENDING'
	`, copyID, reservationID)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("reservation is no longer pending")
	}

	return nil
}

func CancelReservation(
	tx *gorm.DB,
	reservationID int64,
) error {
	result := tx.Exec(`
		UPDATE reservation
		SET status = 'CANCELLED'
		WHERE reservation_id = ?
		  AND status IN ('PENDING', 'NOTIFIED')
	`, reservationID)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("reservation is not active")
	}

	return nil
}
