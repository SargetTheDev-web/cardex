// internal/repository/overdue_report_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type OverdueReportRow struct {
	ID        string    `json:"id"`
	Date      time.Time `json:"date"`
	BookTitle string    `json:"book_title"`
	Borrower  string    `json:"borrower"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	UserID    int       `json:"-"`
	Category  string    `json:"-"`
}

func GetOverdueReportRows(
	db *gorm.DB,
	fromDate time.Time,
	toDate time.Time,
	categoryID *int,
	roleID *int,
) ([]OverdueReportRow, error) {
	var rows []OverdueReportRow

	query := `
		SELECT
			'O-' || l.loan_id::text AS id,
			l.due_date::timestamp AS date,
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
			'Overdue' AS type,
			CASE
				WHEN l.return_date IS NOT NULL THEN 'Resolved'
				ELSE 'Open'
			END AS status,
			u.user_id AS user_id,
			COALESCE(c.category_name, 'Uncategorized') AS category
		FROM loan l
		JOIN book_copy bc ON bc.copy_id = l.copy_id
		JOIN book b ON b.book_id = bc.book_id
		LEFT JOIN category c ON c.category_id = b.category_id
		JOIN "user" u ON u.user_id = l.user_id
		LEFT JOIN user_profile up ON up.user_id = u.user_id
		WHERE l.due_date >= ?
		  AND l.due_date < (?::date + INTERVAL '1 day')
		  AND l.due_date < CURRENT_DATE
		  AND (?::int IS NULL OR c.category_id = ?)
		  AND (?::int IS NULL OR u.role_id = ?)
		ORDER BY l.due_date DESC, l.loan_id DESC
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
		rows = []OverdueReportRow{}
	}

	return rows, nil
}
