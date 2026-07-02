package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreatePasswordHistory(
	db *gorm.DB,
	userID int,
	passwordHash string,
) error {
	return db.Table("password_history").Create(map[string]interface{}{
		"user_id":       userID,
		"password_hash": passwordHash,
	}).Error
}

func GetRecentPasswordHistory(
	db *gorm.DB,
	userID int,
	limit int,
) ([]model.PasswordHistory, error) {

	var history []model.PasswordHistory

	err := db.
		Table("password_history").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&history).Error

	return history, err
}
