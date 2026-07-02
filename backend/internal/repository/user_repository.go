// internal/repository/user_repository.go

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

func CreateUser(
	db *gorm.DB,
	user *model.User,
) error {
	return db.Create(user).Error
}

func GetUserByID(
	db *gorm.DB,
	userID int,
) (*model.User, error) {

	var user model.User

	err := db.
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func UpdatePassword(
	db *gorm.DB,
	userID int,
	newHash string,
) error {
	return db.
		Table("user").
		Where("user_id = ?", userID).
		Update("password_hash", newHash).Error
}
