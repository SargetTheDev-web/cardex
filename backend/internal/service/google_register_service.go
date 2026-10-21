// internal/service/google_register_service.go

package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"
	"backend/pkg/hash"

	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

type GoogleRegistrationInput struct {
	IDToken         string
	Username        string
	Role            string
	InstitutionalID string
	LastName        string
	FirstName       string
	MiddleName      *string
	SuffixExtension *string
	Course          string
	MobileNumber    *string
	PIN             string
	IP              string
	UserAgent       string
}

func CompleteGoogleRegistration(
	db *gorm.DB,
	input GoogleRegistrationInput,
) error {
	// Normalize fields.
	input.IDToken = strings.TrimSpace(input.IDToken)
	input.Username = strings.TrimSpace(input.Username)
	input.Role = strings.ToUpper(strings.TrimSpace(input.Role))
	input.InstitutionalID = strings.TrimSpace(input.InstitutionalID)
	input.LastName = strings.TrimSpace(input.LastName)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.Course = strings.TrimSpace(input.Course)
	input.PIN = strings.TrimSpace(input.PIN)

	if input.IDToken == "" {
		return errors.New("Google ID token is required")
	}

	if input.Username == "" {
		return errors.New("username is required")
	}
	if input.InstitutionalID == "" {
		return errors.New("institutional ID is required")
	}
	if input.FirstName == "" || input.LastName == "" {
		return errors.New("first name and last name are required")
	}

	var requestedRoleID int
	switch input.Role {
	case "STUDENT":
		requestedRoleID = RoleStudent
		if input.Course == "" {
			return errors.New("course is required for student registration")
		}
	case "FACULTY":
		requestedRoleID = RoleFaculty
		input.Course = ""
	default:
		return errors.New("role must be STUDENT or FACULTY")
	}

	if len(input.PIN) != 6 {
		return errors.New("PIN must be exactly 6 digits")
	}
	for _, char := range input.PIN {
		if char < '0' || char > '9' {
			return errors.New("PIN must contain only digits")
		}
	}

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return errors.New("Google sign-in is not configured")
	}

	payload, err := idtoken.Validate(
		context.Background(),
		input.IDToken,
		clientID,
	)
	if err != nil {
		return errors.New("Google ID token is invalid or expired")
	}

	subject, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	picture, _ := payload.Claims["picture"].(string)

	subject = strings.TrimSpace(subject)
	email = strings.ToLower(strings.TrimSpace(email))

	if subject == "" || email == "" || !emailVerified {
		return errors.New("Google must provide a verified email and valid subject")
	}

	var middleName, suffix, mobile, profileImage *string

	if input.MiddleName != nil && strings.TrimSpace(*input.MiddleName) != "" {
		value := strings.TrimSpace(*input.MiddleName)
		middleName = &value
	}
	if input.SuffixExtension != nil && strings.TrimSpace(*input.SuffixExtension) != "" {
		value := strings.TrimSpace(*input.SuffixExtension)
		suffix = &value
	}
	if input.MobileNumber != nil && strings.TrimSpace(*input.MobileNumber) != "" {
		value := strings.TrimSpace(*input.MobileNumber)
		mobile = &value
	}
	if strings.TrimSpace(picture) != "" {
		profileImage = &picture
	}

	hashedPIN, err := hash.HashPassword(input.PIN)
	if err != nil {
		return errors.New("failed to secure PIN")
	}

	provider := "google"
	providerID := subject

	return db.Transaction(func(tx *gorm.DB) error {
		// Prevent duplicate registration for an already linked identity.
		_, err := repository.GetUserBySSO(tx, provider, providerID)
		if err == nil {
			return errors.New("this Google account is already registered")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("failed to check Google account")
		}

		// Never auto-link an existing account based on email.
		_, err = repository.GetUserByIdentifier(tx, email)
		if err == nil {
			return errors.New("email already belongs to a CARDex account; account linking is required")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("failed to check email")
		}

		_, err = repository.GetUserByIdentifier(tx, input.Username)
		if err == nil {
			return errors.New("username already in use")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("failed to check username")
		}

		exists, err := repository.InstitutionalIDExists(
			tx,
			input.InstitutionalID,
			0,
		)
		if err != nil {
			return errors.New("failed to validate institutional ID")
		}
		if exists {
			return errors.New("institutional ID already exists")
		}

		expirationDate := time.Now().AddDate(1, 0, 0)

		user := model.User{
			Username:       input.Username,
			EmailAddress:   email,
			StatusID:       3,
			RoleID:         RoleGuest,
			ExpirationDate: expirationDate,
			SSOProvider:    &provider,
			SSOProviderID:  &providerID,
		}

		if err := repository.CreateUser(tx, &user); err != nil {
			return errors.New("failed to create CARDex account")
		}

		profileChange := model.ProfileChangeRequest{
			UserID:          user.UserID,
			RequestedRoleID: &requestedRoleID,
			InstitutionalID: &input.InstitutionalID,
			LastName:        &input.LastName,
			FirstName:       &input.FirstName,
			MiddleName:      middleName,
			SuffixExtension: suffix,
			MobileNumber:    mobile,
			ProfileImageURL: profileImage,
			PINHash:         &hashedPIN,
			Status:          "PENDING",
		}

		if requestedRoleID == RoleStudent {
			profileChange.Course = &input.Course
		}

		if err := repository.CreateProfileChangeRequest(
			tx,
			&profileChange,
		); err != nil {
			return errors.New("failed to create pending profile")
		}

		if err := repository.InsertAuditLog(
			tx,
			&user.UserID,
			2,
			2,
			"Google registration completed; profile pending approval",
			input.IP,
			input.UserAgent,
		); err != nil {
			return errors.New("failed to audit Google registration")
		}

		return nil
	})
}
