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

	if newPassword == "" || confirmPassword == "" {
		return errors.New("new password and confirmation are required")
	}

	if newPassword != confirmPassword {
		return errors.New("new password and confirmation do not match")
	}

	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	user, err := repository.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}

		return err
	}

	// An SSO-only account has no local password but has both
	// SSO provider identifiers.
	ssoOnlyAccount :=
		user.PasswordHash == "" &&
			user.SSOProvider != nil &&
			strings.TrimSpace(*user.SSOProvider) != "" &&
			user.SSOProviderID != nil &&
			strings.TrimSpace(*user.SSOProviderID) != ""

	// Accounts with local passwords must verify their current password.
	if !ssoOnlyAccount {
		if user.PasswordHash == "" {
			return errors.New("password change is unavailable for this account")
		}

		if currentPassword == "" {
			return errors.New("current password is required")
		}

		if currentPassword == newPassword {
			return errors.New("new password must be different from current password")
		}

		if err := bcrypt.CompareHashAndPassword(
			[]byte(user.PasswordHash),
			[]byte(currentPassword),
		); err != nil {
			return errors.New("current password is incorrect")
		}
	}

	// Check recent password history for both account types.
	history, err := repository.GetRecentPasswordHistory(db, userID, 5)
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

	// Only save the previous password if one actually exists.
	if user.PasswordHash != "" {
		if err := repository.CreatePasswordHistory(
			db,
			userID,
			user.PasswordHash,
		); err != nil {
			return err
		}
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
