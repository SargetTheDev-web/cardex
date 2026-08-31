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

func UsernameExists(
	db *gorm.DB,
	username string,
	excludeUserID int,
) (bool, error) {

	var count int64

	err := db.
		Table(`"user"`).
		Where(
			"username = ? AND user_id <> ?",
			username,
			excludeUserID,
		).
		Count(&count).Error

	return count > 0, err
}

func EmailExists(
	db *gorm.DB,
	email string,
	excludeUserID int,
) (bool, error) {

	var count int64

	err := db.
		Table(`"user"`).
		Where(
			"email_address = ? AND user_id <> ?",
			email,
			excludeUserID,
		).
		Count(&count).Error

	return count > 0, err
}

func InstitutionalIDExists(
	db *gorm.DB,
	institutionalID string,
	excludeUserID int,
) (bool, error) {

	var count int64

	err := db.
		Table("user_profile").
		Where(
			"institutional_id = ? AND user_id <> ?",
			institutionalID,
			excludeUserID,
		).
		Count(&count).Error

	return count > 0, err
}

func GetUserRoleID(
	db *gorm.DB,
	userID int,
) (int, error) {

	var user model.User

	err := db.
		Table(`"user"`).
		Select("user_id, role_id").
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		return 0, err
	}

	return user.RoleID, nil
}
