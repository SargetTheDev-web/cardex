// internal/repository/loan_repository.go

package repository

import (
	model "backend/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BorrowerAccount struct {
	UserID         int        `gorm:"column:user_id"`
	StatusCode     string     `gorm:"column:status_code"`
	RoleCode       string     `gorm:"column:role_code"`
	ExpirationDate *time.Time `gorm:"column:expiration_date"`
}

func GetBorrowerAccount(tx *gorm.DB, userID int) (BorrowerAccount, error) {
	var account BorrowerAccount

	err := tx.Table(`"user" AS u`).
		Select(`
			u.user_id,
			s.status_code,
			r.role_code,
			u.expiration_date
		`).
		Joins("JOIN user_status s ON s.status_id = u.status_id").
		Joins("JOIN user_role r ON r.role_id = u.role_id").
		Where("u.user_id = ?", userID).
		First(&account).Error

	return account, err
}

type LockedBookCopy struct {
	CopyID     int     `gorm:"column:copy_id"`
	Barcode    *string `gorm:"column:barcode"`
	CopyStatus string  `gorm:"column:copy_status"`
}

func LockBookCopyByBarcode(
	tx *gorm.DB,
	barcode string,
) (LockedBookCopy, error) {
	var copy LockedBookCopy

	err := tx.Table("book_copy").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("barcode = ?", barcode).
		First(&copy).Error

	return copy, err
}

func GetDatabaseCurrentDate(tx *gorm.DB) (time.Time, error) {
	var result struct {
		CurrentDate time.Time `gorm:"column:current_date"`
	}

	err := tx.Raw("SELECT CURRENT_DATE AS current_date").Scan(&result).Error
	return result.CurrentDate, err
}

func CreateLoan(tx *gorm.DB, loan *model.Loan) error {
	return tx.Create(loan).Error
}

func MarkCopyBorrowed(tx *gorm.DB, copyID int) error {
	result := tx.Table("book_copy").
		Where("copy_id = ? AND copy_status = 'AVAILABLE'", copyID).
		Update("copy_status", "BORROWED")

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
