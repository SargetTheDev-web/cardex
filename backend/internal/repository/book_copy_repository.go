// internal/repository/book_copy_repository.go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreateBookCopy(db *gorm.DB, copy *model.BookCopy) error {
	return db.Create(copy).Error
}

func GetAllBookCopies(db *gorm.DB) ([]model.BookCopy, error) {
	var copies []model.BookCopy

	err := db.
		Order("copy_id ASC").
		Find(&copies).Error
	if err != nil {
		return nil, err
	}

	if copies == nil {
		copies = []model.BookCopy{}
	}

	return copies, nil
}

func GetBookCopyByID(db *gorm.DB, copyID int) (model.BookCopy, error) {
	var copy model.BookCopy

	err := db.First(&copy, "copy_id = ?", copyID).Error
	if err != nil {
		return model.BookCopy{}, err
	}

	return copy, nil
}

func UpdateBookCopy(
	db *gorm.DB,
	copyID int,
	updates map[string]interface{},
) error {
	result := db.
		Model(&model.BookCopy{}).
		Where("copy_id = ?", copyID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func UpdateBookCopyStatus(
	db *gorm.DB,
	copyID int,
	status string,
) error {
	result := db.
		Model(&model.BookCopy{}).
		Where("copy_id = ?", copyID).
		Update("copy_status", status)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
