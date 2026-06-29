package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func GetUserByIdentifier(
	db *gorm.DB,
	identifier string,
) (*model.User, error) {

	var user model.User

	err := db.
		Where(
			"email_address = ? OR username = ?",
			identifier,
			identifier,
		).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}
