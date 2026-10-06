// internal/service/change_pin_service.go

package service

import (
	"errors"
	"regexp"
	"strings"

	"backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var sixDigitPIN = regexp.MustCompile(`^\d{6}$`)

func ChangePIN(
	db *gorm.DB,
	userID int,
	currentPIN string,
	newPIN string,
	confirmPIN string,
	ip string,
	userAgent string,
) error {

	currentPIN = strings.TrimSpace(currentPIN)
	newPIN = strings.TrimSpace(newPIN)
	confirmPIN = strings.TrimSpace(confirmPIN)

	// Required fields
	if currentPIN == "" || newPIN == "" || confirmPIN == "" {
		return errors.New("all PIN fields are required")
	}

	// PIN format
	if !sixDigitPIN.MatchString(currentPIN) {
		return errors.New("current PIN must be exactly 6 digits")
	}

	if !sixDigitPIN.MatchString(newPIN) {
		return errors.New("new PIN must be exactly 6 digits")
	}

	// Confirmation
	if newPIN != confirmPIN {
		return errors.New("new PIN and confirmation do not match")
	}

	// Prevent same PIN
	if currentPIN == newPIN {
		return errors.New("new PIN must be different from current PIN")
	}

	// Get user
	user, err := repository.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}

		return err
	}

	// Make sure the account has a PIN
	if user.PINHash == nil || *user.PINHash == "" {
		return errors.New("PIN change is unavailable for this account")
	}

	// Verify current PIN
	if err := bcrypt.CompareHashAndPassword(
		[]byte(*user.PINHash),
		[]byte(currentPIN),
	); err != nil {
		return errors.New("current PIN is incorrect")
	}

	// Hash new PIN
	newPINHash, err := bcrypt.GenerateFromPassword(
		[]byte(newPIN),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	// Update PIN
	if err := repository.UpdatePIN(
		db,
		userID,
		string(newPINHash),
	); err != nil {
		return err
	}

	// Audit
	if err := repository.InsertAuditLog(
		db,
		&userID,
		3,
		1,
		"PIN changed successfully",
		ip,
		userAgent,
	); err != nil {
		return err
	}

	return nil
}
