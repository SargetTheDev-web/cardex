// internal/repository/profile_repository_go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func GetUserProfile(
	db *gorm.DB,
	userID int,
) (*model.User, *model.UserProfile, error) {

	var user model.User
	var profile model.UserProfile

	if err := db.
		Table(`"user"`).
		Where("user_id = ?", userID).
		First(&user).Error; err != nil {
		return nil, nil, err
	}

	if err := db.
		Table("user_profile").
		Where("user_id = ?", userID).
		First(&profile).Error; err != nil {
		return &user, nil, err
	}

	return &user, &profile, nil
}

func GetUserProfileByUserID(
	db *gorm.DB,
	userID int,
) (*model.UserProfile, error) {

	var profile model.UserProfile

	err := db.
		Table("user_profile").
		Where("user_id = ?", userID).
		First(&profile).Error

	if err != nil {
		return nil, err
	}

	return &profile, nil
}

func UpdateUserProfile(
	db *gorm.DB,
	userID int,
	updates map[string]interface{},
) error {

	return db.
		Table("user_profile").
		Where("user_id = ?", userID).
		Updates(updates).Error
}

func UpdateUserAccount(
	db *gorm.DB,
	userID int,
	updates map[string]interface{},
) error {

	return db.
		Table(`"user"`).
		Where("user_id = ?", userID).
		Updates(updates).Error
}

func CreateUserProfile(
	db *gorm.DB,
	profile *model.UserProfile,
) error {
	return db.Create(profile).Error
}
