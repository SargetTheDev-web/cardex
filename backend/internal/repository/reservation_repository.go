// internal/repository/reservation_repository.go

package repository

import (
	"errors"
	"time"

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

type ReservationCreatedDetails struct {
	ReservationID       int64   `gorm:"column:reservation_id"`
	BookID              int     `gorm:"column:book_id"`
	BookTitle           string  `gorm:"column:book_title"`
	UserID              int     `gorm:"column:user_id"`
	RealName            string  `gorm:"column:real_name"`
	PreferredPickupDate string  `gorm:"column:preferred_pickup_date"`
	Notes               *string `gorm:"column:notes"`
	Status              string  `gorm:"column:status"`
	QueuePosition       *int    `gorm:"column:queue_position"`
}

func GetCreatedReservationDetails(
	db *gorm.DB,
	reservationID int64,
) (ReservationCreatedDetails, error) {
	var details ReservationCreatedDetails

	err := db.Raw(`
		SELECT
			r.reservation_id,
			r.book_id,
			b.title AS book_title,
			r.user_id,
			TRIM(CONCAT_WS(' ',
				NULLIF(TRIM(up.first_name), ''),
				NULLIF(TRIM(up.middle_name), ''),
				NULLIF(TRIM(up.last_name), ''),
				NULLIF(TRIM(up.suffix_extension), '')
			)) AS real_name,
			r.preferred_pickup_date,
			r.reason AS notes,
			r.status,
			r.queue_position
		FROM reservation r
		JOIN book b ON b.book_id = r.book_id
		JOIN user_profile up ON up.user_id = r.user_id
		WHERE r.reservation_id = ?
	`, reservationID).Scan(&details).Error

	return details, err
}

type ReservationQueueItem struct {
	ReservationID       int64      `json:"reservation_id" gorm:"column:reservation_id"`
	BookID              int        `json:"book_id" gorm:"column:book_id"`
	BookTitle           string     `json:"book_title" gorm:"column:book_title"`
	UserID              int        `json:"user_id" gorm:"column:user_id"`
	BorrowerName        string     `json:"borrower_name" gorm:"column:borrower_name"`
	CopyID              *int       `json:"copy_id" gorm:"column:copy_id"`
	ReservationDate     time.Time  `json:"reservation_date" gorm:"column:reservation_date"`
	PreferredPickupDate time.Time  `json:"preferred_pickup_date" gorm:"column:preferred_pickup_date"`
	Status              string     `json:"status" gorm:"column:status"`
	Reason              *string    `json:"reason" gorm:"column:reason"`
	QueuePosition       *int       `json:"queue_position" gorm:"column:queue_position"`
	ExpiryDate          *time.Time `json:"expiry_date" gorm:"column:expiry_date"`
	CreatedAt           time.Time  `json:"created_at" gorm:"column:created_at"`
	Role                string     `json:"role" gorm:"column:role"`
	InstitutionalID     *string    `json:"institutional_id" gorm:"column:institutional_id"`
}

func GetReservationQueue(db *gorm.DB) ([]ReservationQueueItem, error) {
	var reservations []ReservationQueueItem

	err := db.Raw(`
		SELECT
    r.reservation_id,
    r.book_id,
    b.title AS book_title,
    r.user_id,
    u.role_id AS role_id,
    up.institutional_id AS institutional_id,
    TRIM(CONCAT_WS(' ',
        NULLIF(TRIM(up.first_name), ''),
        NULLIF(TRIM(up.middle_name), ''),
        NULLIF(TRIM(up.last_name), ''),
        NULLIF(TRIM(up.suffix_extension), '')
    )) AS borrower_name,
    r.copy_id,
    r.reservation_date,
    r.preferred_pickup_date,
    r.status,
    r.reason,
    r.queue_position,
    r.expiry_date,
    r.created_at
FROM reservation r
JOIN book b ON b.book_id = r.book_id
JOIN "user" u ON u.user_id = r.user_id
LEFT JOIN user_profile up ON up.user_id = r.user_id
WHERE r.status IN ('PENDING', 'NOTIFIED')
ORDER BY r.reservation_date ASC, r.reservation_id ASC
	`).Scan(&reservations).Error

	if reservations == nil {
		reservations = []ReservationQueueItem{}
	}

	return reservations, err
}
