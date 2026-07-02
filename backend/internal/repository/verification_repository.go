// internal/repository/verification_repository.go

package repository

import (
	model "backend/internal/models"
	"time"

	"gorm.io/gorm"
)

func CreateVerificationCode(
	db *gorm.DB,
	email string,
	code string,
	expiryTime time.Time,
) error {
	return db.Create(&model.EmailVerification{
		EmailAddress:     email,
		VerificationCode: code,
		ExpiresAt:        expiryTime,
	}).Error
}

func GetVerificationByEmail(
	db *gorm.DB,
	email string,
) (*model.EmailVerification, error) {
	var verification model.EmailVerification

	err := db.
		Where("email_address = ?", email).
		Order("created_at DESC").
		First(&verification).Error

	if err != nil {
		return nil, err
	}

	return &verification, nil
}

func MarkVerificationAsVerified(
	db *gorm.DB,
	verificationID int,
) error {
	return db.
		Model(&model.EmailVerification{}).
		Where("verification_id = ?", verificationID).
		Update("is_verified", true).Error
}

func InvalidateOldVerifications(
	db *gorm.DB,
	email string,
) error {
	return db.
		Model(&model.EmailVerification{}).
		Where(
			"email_address = ? AND is_verified = false",
			email,
		).
		Update("expires_at", time.Now()).Error
}

func GetLatestVerificationByEmail(
	db *gorm.DB,
	email string,
) (*model.EmailVerification, error) {

	var verification model.EmailVerification

	err := db.
		Where("email_address = ?", email).
		Order("created_at DESC").
		First(&verification).Error

	if err != nil {
		return nil, err
	}

	return &verification, nil
}

func DeleteVerificationByEmail(
	db *gorm.DB,
	email string,
) error {

	return db.
		Where("email_address = ?", email).
		Delete(&model.EmailVerification{}).Error
}
