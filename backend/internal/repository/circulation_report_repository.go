// internal/repository/circulation_report_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type CirculationReportRow struct {
	ID         string    `json:"id"`
	Date       time.Time `json:"date"`
	BookTitle  string    `json:"book_title"`
	Borrower   string    `json:"borrower"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	CategoryID *int      `json:"-"`
	Category   string    `json:"-"`
	Overdue    bool      `json:"-"`
}

type CirculationCategoryCount struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Count      int64   `json:"count"`
}

func GetCirculationReportRows(
	db *gorm.DB,
	fromDate time.Time,
	toDate time.Time,
	categoryID *int,
	roleID *int,
) ([]CirculationReportRow, error) {
	var rows []CirculationReportRow

	query := `
		SELECT
			'B-' || l.loan_id::text AS id,
			l.loan_date::timestamp AS date,
			b.title AS book_title,
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
			'Borrow' AS type,
			CASE
				WHEN l.status = 'RETURNED' THEN 'Returned'
				WHEN l.status = 'OVERDUE'
					OR (l.return_date IS NULL AND l.due_date < CURRENT_DATE)
					THEN 'Overdue'
				ELSE 'Active'
			END AS status,
			c.category_id,
			COALESCE(c.category_name, 'Uncategorized') AS category,
			FALSE AS overdue
		FROM loan l
		JOIN book_copy bc ON bc.copy_id = l.copy_id
		JOIN book b ON b.book_id = bc.book_id
		LEFT JOIN category c ON c.category_id = b.category_id
		JOIN "user" u ON u.user_id = l.user_id
		LEFT JOIN user_profile up ON up.user_id = u.user_id
		WHERE l.loan_date >= ?
		  AND l.loan_date <= ?
		  AND (?::int IS NULL OR c.category_id = ?)
		  AND (?::int IS NULL OR u.role_id = ?)

		UNION ALL

		SELECT
			'R-' || l.loan_id::text AS id,
			l.return_date::timestamp AS date,
			b.title AS book_title,
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
			'Return' AS type,
			CASE
				WHEN l.return_date > l.due_date THEN 'Overdue'
				ELSE 'Returned'
			END AS status,
			c.category_id,
			COALESCE(c.category_name, 'Uncategorized') AS category,
			(l.return_date > l.due_date) AS overdue
		FROM loan l
		JOIN book_copy bc ON bc.copy_id = l.copy_id
		JOIN book b ON b.book_id = bc.book_id
		LEFT JOIN category c ON c.category_id = b.category_id
		JOIN "user" u ON u.user_id = l.user_id
		LEFT JOIN user_profile up ON up.user_id = u.user_id
		WHERE l.return_date >= ?
		  AND l.return_date < (?::date + INTERVAL '1 day')
		  AND (?::int IS NULL OR c.category_id = ?)
		  AND (?::int IS NULL OR u.role_id = ?)

		ORDER BY date DESC, id DESC
	`

	err := db.Raw(
		query,
		fromDate, toDate,
		categoryID, categoryID,
		roleID, roleID,

		fromDate, toDate,
		categoryID, categoryID,
		roleID, roleID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	if rows == nil {
		rows = []CirculationReportRow{}
	}

	return rows, nil
}
