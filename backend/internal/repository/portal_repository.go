// internal/repository/portal_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type PortalStatistics struct {
	BooksBorrowed int64 `json:"books_borrowed"`
	Reservations  int64 `json:"reservations"`
	Overdue       int64 `json:"overdue"`
}

type PortalTransaction struct {
	TransactionID string    `json:"transaction_id" gorm:"column:transaction_id"`
	Type          string    `json:"type" gorm:"column:type"`
	BookTitle     string    `json:"book_title" gorm:"column:book_title"`
	Date          time.Time `json:"date" gorm:"column:transaction_date"`
	Status        string    `json:"status" gorm:"column:status"`
}

// GetPortalStatistics retrieves dashboard statistics for one user.
func GetPortalStatistics(
	db *gorm.DB,
	userID int,
) (PortalStatistics, error) {
	var statistics PortalStatistics

	const query = `
		SELECT
			(
				SELECT COUNT(*)
				FROM loan
				WHERE user_id = ?
				  AND return_date IS NULL
				  AND status::text IN ('BORROWED', 'OVERDUE')
			) AS books_borrowed,

			(
				SELECT COUNT(*)
				FROM reservation
				WHERE user_id = ?
				  AND status::text IN ('PENDING', 'NOTIFIED')
				  AND (
						expiry_date IS NULL
						OR expiry_date > CURRENT_TIMESTAMP
				  )
			) AS reservations,

			(
				SELECT COUNT(*)
				FROM loan
				WHERE user_id = ?
				  AND return_date IS NULL
				  AND (
						status::text = 'OVERDUE'
						OR (
							status::text = 'BORROWED'
							AND due_date < CURRENT_DATE
						)
				  )
			) AS overdue
	`

	err := db.Raw(
		query,
		userID,
		userID,
		userID,
	).Scan(&statistics).Error

	if err != nil {
		return PortalStatistics{}, err
	}

	return statistics, nil
}

// GetPortalRecentTransactions retrieves the latest borrowing,
// return, and reservation activity for one user.
func GetPortalRecentTransactions(
	db *gorm.DB,
	userID int,
	limit int,
) ([]PortalTransaction, error) {
	transactions := make([]PortalTransaction, 0)

	if limit <= 0 {
		limit = 10
	}

	const query = `
		SELECT *
		FROM (
			-- Borrowing activity
			SELECT
				CONCAT('LOAN-', l.loan_id) AS transaction_id,
				'BORROW' AS type,
				b.title AS book_title,
				l.loan_date::timestamp AS transaction_date,
				CASE
					WHEN l.return_date IS NOT NULL THEN 'RETURNED'
					WHEN l.status::text = 'OVERDUE'
						OR (
							l.status::text = 'BORROWED'
							AND l.due_date < CURRENT_DATE
						) THEN 'OVERDUE'
					ELSE l.status::text
				END AS status
			FROM loan l
			INNER JOIN book_copy bc
				ON bc.copy_id = l.copy_id
			INNER JOIN book b
				ON b.book_id = bc.book_id
			WHERE l.user_id = ?

			UNION ALL

			-- Return activity
			SELECT
				CONCAT('RETURN-', l.loan_id) AS transaction_id,
				'RETURN' AS type,
				b.title AS book_title,
				l.return_date::timestamp AS transaction_date,
				'RETURNED' AS status
			FROM loan l
			INNER JOIN book_copy bc
				ON bc.copy_id = l.copy_id
			INNER JOIN book b
				ON b.book_id = bc.book_id
			WHERE l.user_id = ?
			  AND l.return_date IS NOT NULL

			UNION ALL

			-- Reservation creation activity
			SELECT
				CONCAT('RESERVATION-', r.reservation_id) AS transaction_id,
				'RESERVATION' AS type,
				b.title AS book_title,
				r.reservation_date AS transaction_date,
				r.status::text AS status
			FROM reservation r
			INNER JOIN book b
				ON b.book_id = r.book_id
			WHERE r.user_id = ?
		) AS activity
		ORDER BY transaction_date DESC, transaction_id DESC
		LIMIT ?
	`

	err := db.Raw(
		query,
		userID,
		userID,
		userID,
		limit,
	).Scan(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}
