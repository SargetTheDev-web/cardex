// internal/repository/inventory_report_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type InventoryCategoryRow struct {
	CategoryID   *int   `gorm:"column:category_id"`
	CategoryName string `gorm:"column:category_name"`
	Count        int64  `gorm:"column:count"`
}

type InventoryReportRows struct {
	TitlesRepresented int64
	Borrowed          int64
	Returned          int64
	OnHold            int64
	Categories        []InventoryCategoryRow
}

func GetInventoryReport(
	db *gorm.DB,
	fromDate time.Time,
	toDate time.Time,
	categoryID *int,
	roleID *int,
) (*InventoryReportRows, error) {
	result := &InventoryReportRows{}

	// Current inventory metrics. Category filtering applies to all three.
	err := db.Raw(`
		SELECT
			COUNT(DISTINCT b.book_id) AS titles_represented,
			COUNT(*) FILTER (WHERE bc.copy_status = 'BORROWED') AS borrowed,
			COUNT(*) FILTER (WHERE bc.copy_status = 'RESERVED') AS on_hold
		FROM book_copy bc
		JOIN book b ON b.book_id = bc.book_id
		WHERE (?::int IS NULL OR b.category_id = ?)
	`,
		categoryID,
		categoryID,
	).Row().Scan(
		&result.TitlesRepresented,
		&result.Borrowed,
		&result.OnHold,
	)
	if err != nil {
		return nil, err
	}

	// Returned copies during the selected period.
	err = db.Raw(`
		SELECT COUNT(*)
		FROM loan l
		JOIN book_copy bc ON bc.copy_id = l.copy_id
		JOIN book b ON b.book_id = bc.book_id
		JOIN "user" u ON u.user_id = l.user_id
		WHERE l.status = 'RETURNED'
		  AND l.return_date >= ?::date
		  AND l.return_date < (?::date + INTERVAL '1 day')
		  AND (?::int IS NULL OR b.category_id = ?)
		  AND (?::int IS NULL OR u.role_id = ?)
	`,
		fromDate,
		toDate,
		categoryID,
		categoryID,
		roleID,
		roleID,
	).Row().Scan(&result.Returned)
	if err != nil {
		return nil, err
	}

	// Holdings by category. Include categories with zero copies.
	err = db.Raw(`
		SELECT
			c.category_id,
			c.category_name,
			COUNT(bc.copy_id) AS count
		FROM category c
		LEFT JOIN book b ON b.category_id = c.category_id
		LEFT JOIN book_copy bc ON bc.book_id = b.book_id
		WHERE (?::int IS NULL OR c.category_id = ?)
		GROUP BY c.category_id, c.category_name
		ORDER BY c.category_name
	`,
		categoryID,
		categoryID,
	).Scan(&result.Categories).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}
