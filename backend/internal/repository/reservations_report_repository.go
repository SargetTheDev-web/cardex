// internal/repository/reservations_report_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type ReservationReportRow struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"`
	Book     string `json:"book"`
	Borrower string `json:"borrower"`
	Type     string `json:"type"`
	Status   string `json:"status"`

	UserID   int    `json:"-"`
	Category string `json:"-"`
}

func GetReservationsReportRows(
	db *gorm.DB,
	fromDate time.Time,
	toDate time.Time,
	categoryID *int,
	roleID *int,
) ([]ReservationReportRow, error) {
	var rows []ReservationReportRow

	query := `
		SELECT
			r.reservation_id AS id,
			r.reservation_date::text AS date,
			b.title AS book,
			COALESCE(
				NULLIF(
					TRIM(CONCAT_WS(' ',
						up.first_name,
						up.middle_name,
						up.last_name,
						up.suffix_extension
					)),
					''
				),
				u.username
			) AS borrower,
			'Reservation' AS type,
			CASE r.status
				WHEN 'PENDING' THEN 'Pending'
				WHEN 'NOTIFIED' THEN 'Ready'
				WHEN 'FULFILLED' THEN 'Fulfilled'
				ELSE INITCAP(LOWER(r.status))
			END AS status,
			u.user_id AS user_id,
			COALESCE(c.category_name, 'Uncategorized') AS category
		FROM reservation r
		JOIN "user" u ON u.user_id = r.user_id
		LEFT JOIN user_profile up ON up.user_id = u.user_id
		LEFT JOIN book b ON b.book_id = r.book_id
		LEFT JOIN book_copy bc ON bc.copy_id = r.copy_id
		LEFT JOIN category c
			ON c.category_id = COALESCE(b.category_id, (
				SELECT b2.category_id
				FROM book b2
				WHERE b2.book_id = bc.book_id
			))
		WHERE r.reservation_date >= ?
		  AND r.reservation_date < (?::date + INTERVAL '1 day')
		  AND (?::int IS NULL OR c.category_id = ?)
		  AND (?::int IS NULL OR u.role_id = ?)
		ORDER BY r.reservation_date DESC, r.reservation_id DESC
	`

	err := db.Raw(
		query,
		fromDate,
		toDate,
		categoryID,
		categoryID,
		roleID,
		roleID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []ReservationReportRow{}
	}

	return rows, nil
}
