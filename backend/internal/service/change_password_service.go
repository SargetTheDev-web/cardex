// internal/service/change_password_service.go

package service

import (
	"errors"
	"strings"

	"backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func ChangePassword(
	db *gorm.DB,
	userID int,
	currentPassword string,
	newPassword string,
	confirmPassword string,
	ip string,
	userAgent string,
) error {

	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	confirmPassword = strings.TrimSpace(confirmPassword)

	if currentPassword == "" ||
		newPassword == "" ||
		confirmPassword == "" {
		return errors.New("all password fields are required")
	}

	if newPassword != confirmPassword {
		return errors.New("new password and confirmation do not match")
	}

	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	if currentPassword == newPassword {
		return errors.New("new password must be different from current password")
	}

	user, err := repository.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}

		return err
	}

	if user.PasswordHash == "" {
		return errors.New("password change is unavailable for this account")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(currentPassword),
	); err != nil {
		return errors.New("current password is incorrect")
	}

	history, err := repository.GetRecentPasswordHistory(
		db,
		userID,
		5,
	)
	if err != nil {
		return err
	}

	for _, oldPassword := range history {
		if bcrypt.CompareHashAndPassword(
			[]byte(oldPassword.PasswordHash),
			[]byte(newPassword),
		) == nil {
			return errors.New("new password was recently used")
		}
	}

	if err := repository.CreatePasswordHistory(
		db,
		userID,
		user.PasswordHash,
	); err != nil {
		return err
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	if err := repository.UpdatePassword(
		db,
		userID,
		string(newPasswordHash),
	); err != nil {
		return err
	}

	if err := repository.InsertAuditLog(
		db,
		&userID,
		3,
		1,
		"Password changed successfully",
		ip,
		userAgent,
	); err != nil {
		return err
	}

	return nil
}
