// internal/repository/user_status_repository.go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func GetUserStatus(
	db *gorm.DB,
	statusID int,
) (*model.UserStatus, error) {
	var status model.UserStatus

	err := db.
		Table("user_status").
		Where("status_id = ?", statusID).
		First(&status).Error

	if err != nil {
		return nil, err
	}

	return &status, nil
}
