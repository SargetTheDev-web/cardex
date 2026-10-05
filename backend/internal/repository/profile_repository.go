// internal/repository/profile_repository.go

package repository

import (
	model "backend/internal/models"
	"errors"

	"gorm.io/gorm"
)

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

func GetUserProfile(
	db *gorm.DB,
	userID int,
) (*model.User, *model.UserProfile, error) {

	var user model.User

	// --------------------------------------------------
	// Get user account
	// --------------------------------------------------

	if err := db.
		Table(`"user"`).
		Where("user_id = ?", userID).
		First(&user).Error; err != nil {

		return nil, nil, err
	}

	// --------------------------------------------------
	// Try to get approved/current profile
	// --------------------------------------------------

	var profile model.UserProfile

	err := db.
		Table("user_profile").
		Where("user_id = ?", userID).
		First(&profile).Error

	if err == nil {
		// Approved profile exists.
		return &user, &profile, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Actual database error.
		return &user, nil, err
	}

	// --------------------------------------------------
	// No approved profile yet.
	//
	// Check for a pending registration/profile request.
	// --------------------------------------------------

	request, requestErr := GetPendingProfileChange(
		db,
		userID,
	)

	if requestErr != nil {
		// No profile and no pending request.
		return &user, nil, gorm.ErrRecordNotFound
	}

	// --------------------------------------------------
	// Build temporary profile from pending request
	// --------------------------------------------------

	profile = model.UserProfile{
		UserID: userID,
	}

	if request.InstitutionalID != nil {
		profile.InstitutionalID = *request.InstitutionalID
	}

	if request.LastName != nil {
		profile.LastName = *request.LastName
	}

	if request.FirstName != nil {
		profile.FirstName = *request.FirstName
	}

	profile.MiddleName = request.MiddleName
	profile.SuffixExtension = request.SuffixExtension
	profile.Course = request.Course
	profile.MobileNumber = request.MobileNumber
	profile.BirthDate = request.BirthDate

	return &user, &profile, nil
}
