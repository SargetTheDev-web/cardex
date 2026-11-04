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
) ([]model.Fine, error) {
	fines := make([]model.Fine, 0)

	query := db.Model(&model.Fine{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if loanID > 0 {
		query = query.Where("loan_id = ?", loanID)
	}

	err := query.Order("created_at DESC, fine_id DESC").Find(&fines).Error
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
