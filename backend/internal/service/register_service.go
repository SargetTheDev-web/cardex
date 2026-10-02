// internal/service/register_service.go

// internal/service/register_service.go

package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"backend/internal/mail"
	model "backend/internal/models"
	"backend/internal/repository"
	"backend/pkg/hash"

	"gorm.io/gorm"
)

const (
	RoleFaculty = 4
	RoleStudent = 5
	RoleGuest   = 6
)

func RequestRegistration(
	db *gorm.DB,
	email string,
	ip string,
) error {

	_ = ip

	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" {
		return errors.New("email is required")
	}

	/*
		RECAPTCHA CHECK (DISABLED FOR POSTMAN)

		isHuman := VerifyRecaptcha(token)

		err := repository.InsertRecaptchaLog(
			db,
			email,
			token,
			isHuman,
			ip,
		)

		if err != nil {
			return err
		}

		if !isHuman {
			return errors.New("captcha verification failed")
		}
	*/

	// --------------------------------------------------
	// Check if email is already registered
	// --------------------------------------------------

	fmt.Println("Checking registration email:", email)

	existingUser, err := repository.GetUserByIdentifier(
		db,
		email,
	)

	fmt.Println("Existing user:", existingUser)
	fmt.Println("Lookup error:", err)

	if err == nil && existingUser != nil {
		return errors.New("email already in use")
	}

	if err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {

		return err
	}

	// --------------------------------------------------
	// Check existing verification code
	// --------------------------------------------------

	existing, err := repository.GetLatestVerificationByEmail(
		db,
		email,
	)

	if err == nil {

		if !existing.IsVerified &&
			time.Now().Before(existing.ExpiresAt) {

			return errors.New(
				"verification already sent. check your email",
			)
		}

		if !existing.IsVerified &&
			time.Now().After(existing.ExpiresAt) {

			if err := repository.DeleteVerificationByEmail(
				db,
				email,
			); err != nil {
				return err
			}
		}
	}

	// --------------------------------------------------
	// Generate 6-digit verification code
	// --------------------------------------------------

	num, err := rand.Int(
		rand.Reader,
		big.NewInt(1000000),
	)

	if err != nil {
		return err
	}

	code := fmt.Sprintf(
		"%06d",
		num.Int64(),
	)

	expiry := time.Now().Add(
		10 * time.Minute,
	)

	// --------------------------------------------------
	// Store verification code
	// --------------------------------------------------

	if err := repository.CreateVerificationCode(
		db,
		email,
		code,
		expiry,
	); err != nil {
		return err
	}

	// --------------------------------------------------
	// Send verification email
	// --------------------------------------------------

	if err := mail.SendVerificationCode(
		email,
		code,
	); err != nil {
		return err
	}

	return nil
}

func VerifyRegistrationCode(
	db *gorm.DB,
	email string,
	code string,
) error {

	email = strings.ToLower(
		strings.TrimSpace(email),
	)

	code = strings.TrimSpace(code)

	verification, err := repository.GetVerificationByEmail(
		db,
		email,
	)

	if err != nil {
		return errors.New("verification not found")
	}

	if verification.IsVerified {
		return errors.New("email already verified")
	}

	if time.Now().After(verification.ExpiresAt) {
		return errors.New("verification code expired")
	}

	if verification.VerificationCode != code {
		return errors.New("invalid verification code")
	}

	if err := repository.MarkVerificationAsVerified(
		db,
		verification.VerificationID,
	); err != nil {
		return err
	}

	return nil
}

func CompleteRegistration(
	db *gorm.DB,
	email string,
	username string,
	password string,
	confirmPassword string,
	role string,
	institutionalID string,
	lastName string,
	firstName string,
	middleName *string,
	suffixExtension *string,
	course string,
	mobileNumber *string,
	pin string,
	ip string,
	userAgent string,
) error {

	// --------------------------------------------------
	// Normalize input
	// --------------------------------------------------

	email = strings.ToLower(
		strings.TrimSpace(email),
	)

	username = strings.TrimSpace(username)

	role = strings.ToUpper(
		strings.TrimSpace(role),
	)

	institutionalID = strings.TrimSpace(
		institutionalID,
	)

	lastName = strings.TrimSpace(
		lastName,
	)

	firstName = strings.TrimSpace(
		firstName,
	)

	course = strings.TrimSpace(course)

	pin = strings.TrimSpace(pin)

	// --------------------------------------------------
	// Validate account information
	// --------------------------------------------------

	if email == "" {
		return errors.New("email is required")
	}

	if username == "" {
		return errors.New("username is required")
	}

	if password == "" {
		return errors.New("password is required")
	}

	if password != confirmPassword {
		return errors.New("passwords do not match")
	}

	if len(password) < 8 {
		return errors.New(
			"password must be at least 8 characters",
		)
	}

	// --------------------------------------------------
	// Validate profile information
	// --------------------------------------------------

	if institutionalID == "" {
		return errors.New(
			"institutional ID is required",
		)
	}

	if lastName == "" {
		return errors.New(
			"last name is required",
		)
	}

	if firstName == "" {
		return errors.New(
			"first name is required",
		)
	}

	// --------------------------------------------------
	// Validate requested role
	// --------------------------------------------------

	var requestedRoleID int

	switch role {

	case "STUDENT":

		requestedRoleID = RoleStudent

		if course == "" {
			return errors.New(
				"course is required for student registration",
			)
		}

	case "FACULTY":

		requestedRoleID = RoleFaculty

		// Faculty does not require a course.
		course = ""

	default:

		return errors.New(
			"role must be STUDENT or FACULTY",
		)
	}

	// --------------------------------------------------
	// Validate PIN
	// --------------------------------------------------

	if len(pin) != 6 {
		return errors.New(
			"PIN must be exactly 6 digits",
		)
	}

	for _, char := range pin {

		if char < '0' || char > '9' {
			return errors.New(
				"PIN must contain only digits",
			)
		}
	}

	// --------------------------------------------------
	// Normalize optional fields
	// --------------------------------------------------

	if middleName != nil {

		value := strings.TrimSpace(
			*middleName,
		)

		if value == "" {
			middleName = nil
		} else {
			middleName = &value
		}
	}

	if suffixExtension != nil {

		value := strings.TrimSpace(
			*suffixExtension,
		)

		if value == "" {
			suffixExtension = nil
		} else {
			suffixExtension = &value
		}
	}

	if mobileNumber != nil {
		value := strings.TrimSpace(*mobileNumber)

		if value == "" {
			mobileNumber = nil
		} else {
			mobileNumber = &value
		}
	}

	// --------------------------------------------------
	// Database transaction
	// --------------------------------------------------

	return db.Transaction(func(tx *gorm.DB) error {

		// --------------------------------------------------
		// Verify email verification
		// --------------------------------------------------

		verification, err := repository.GetVerificationByEmail(
			tx,
			email,
		)

		if err != nil {
			return errors.New(
				"verification not found",
			)
		}

		if !verification.IsVerified {
			return errors.New(
				"email not verified",
			)
		}

		// --------------------------------------------------
		// Check email
		// --------------------------------------------------

		existingEmail, err := repository.GetUserByIdentifier(
			tx,
			email,
		)

		if err == nil && existingEmail != nil {
			return errors.New(
				"email already in use",
			)
		}

		if err != nil &&
			!errors.Is(err, gorm.ErrRecordNotFound) {

			return err
		}

		// --------------------------------------------------
		// Check username
		// --------------------------------------------------

		existingUsername, err := repository.GetUserByIdentifier(
			tx,
			username,
		)

		if err == nil && existingUsername != nil {
			return errors.New(
				"username already in use",
			)
		}

		if err != nil &&
			!errors.Is(err, gorm.ErrRecordNotFound) {

			return err
		}

		// --------------------------------------------------
		// Check institutional ID
		// --------------------------------------------------

		exists, err := repository.InstitutionalIDExists(
			tx,
			institutionalID,
			0,
		)

		if err != nil {
			return errors.New(
				"failed to validate institutional ID",
			)
		}

		if exists {
			return errors.New(
				"institutional ID already exists",
			)
		}

		// --------------------------------------------------
		// Hash password
		// --------------------------------------------------

		hashedPassword, err := hash.HashPassword(
			password,
		)

		if err != nil {
			return err
		}

		// --------------------------------------------------
		// Hash PIN
		// --------------------------------------------------

		hashedPIN, err := hash.HashPassword(
			pin,
		)

		if err != nil {
			return err
		}

		// --------------------------------------------------
		// Create GUEST account
		// --------------------------------------------------

		expirationDate := time.Now().AddDate(
			1,
			0,
			0,
		)

		user := model.User{
			Username:       username,
			EmailAddress:   email,
			PasswordHash:   hashedPassword,
			StatusID:       3,
			RoleID:         RoleGuest,
			ExpirationDate: expirationDate,
		}

		if err := repository.CreateUser(
			tx,
			&user,
		); err != nil {
			return err
		}

		// --------------------------------------------------
		// Create pending profile change request
		// --------------------------------------------------

		profileChange := model.ProfileChangeRequest{
			UserID:          user.UserID,
			RequestedRoleID: &requestedRoleID,
			InstitutionalID: &institutionalID,
			LastName:        &lastName,
			FirstName:       &firstName,
			MiddleName:      middleName,
			SuffixExtension: suffixExtension,
			MobileNumber:    mobileNumber,
			PINHash:         &hashedPIN,
			Status:          "PENDING",
		}

		// Course applies only to students.
		if requestedRoleID == RoleStudent {
			profileChange.Course = &course
		}

		if err := repository.CreateProfileChangeRequest(
			tx,
			&profileChange,
		); err != nil {
			return err
		}

		// --------------------------------------------------
		// Audit registration
		// --------------------------------------------------

		if err := repository.InsertAuditLog(
			tx,
			&user.UserID,
			2,
			2,
			"User registered; profile pending approval",
			ip,
			userAgent,
		); err != nil {
			return err
		}

		// --------------------------------------------------
		// Delete verification code
		// --------------------------------------------------

		if err := repository.DeleteVerificationByEmail(
			tx,
			email,
		); err != nil {
			return err
		}

		return nil
	})
}
