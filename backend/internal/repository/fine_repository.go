// internal/repository/fine_repository.go

package repository

import (
	"errors"
	"strings"
	"time"

	model "backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetFines(
	db *gorm.DB,
	status string,
	loanID int64,
) ([]model.FineResponse, error) {
	fines := make([]model.FineResponse, 0)

	query := db.Table("fine f").
		Select(`
			f.fine_id,
			f.loan_id,
			l.user_id,
			TRIM(CONCAT_WS(' ',
				NULLIF(up.first_name, ''),
				NULLIF(up.middle_name, ''),
				NULLIF(up.last_name, ''),
				NULLIF(up.suffix_extension, '')
			)) AS name,
			ur.role_code AS role_code,
			up.institutional_id,
			f.amount,
			f.reason,
			f.status,
			f.paid_at,
			f.created_at
		`).
		Joins("JOIN loan l ON l.loan_id = f.loan_id").
		Joins(`JOIN "user" u ON u.user_id = l.user_id`).
		Joins("LEFT JOIN user_profile up ON up.user_id = u.user_id").
		Joins("LEFT JOIN user_role ur ON ur.role_id = u.role_id")

	if status != "" {
		query = query.Where("f.status = ?", status)
	}

	if loanID > 0 {
		query = query.Where("f.loan_id = ?", loanID)
	}

	err := query.
		Order("f.created_at DESC, f.fine_id DESC").
		Scan(&fines).Error

	return fines, err
}

func LockFineByID(tx *gorm.DB, fineID int64) (model.Fine, error) {
	var fine model.Fine

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("fine_id = ?", fineID).
		First(&fine).Error

	return fine, err
}

func UpdateFineStatus(
	tx *gorm.DB,
	fineID int64,
	status string,
	paidAt *time.Time,
) error {
	result := tx.Model(&model.Fine{}).
		Where("fine_id = ?", fineID).
		Updates(map[string]interface{}{
			"status":  status,
			"paid_at": paidAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// Keep this helper available if you prefer repository-level not-found checks.
func IsFineNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func NormalizeFineStatus(status string) string {
	return strings.ToUpper(strings.TrimSpace(status))
}
