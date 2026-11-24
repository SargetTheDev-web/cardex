package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

	// --------------------------------------------------
	// Normalize input
	// --------------------------------------------------

	input.IDToken = strings.TrimSpace(input.IDToken)
	input.Username = strings.TrimSpace(input.Username)
	input.Role = strings.ToUpper(strings.TrimSpace(input.Role))
	input.InstitutionalID = strings.TrimSpace(input.InstitutionalID)
	input.LastName = strings.ToUpper(strings.TrimSpace(input.LastName))
	input.FirstName = strings.ToUpper(strings.TrimSpace(input.FirstName))
	input.Course = strings.TrimSpace(input.Course)
	input.PIN = strings.TrimSpace(input.PIN)

	if db == nil {
		return errors.New("database connection is required")
	}

	if input.IDToken == "" {
		return errors.New("Google ID token is required")
	}

	if input.Username == "" {
		return errors.New("username is required")
	}

	if input.InstitutionalID == "" {
		return errors.New("Student/Faculty ID is required")
	}

	if input.FirstName == "" || input.LastName == "" {
		return errors.New("first name and last name are required")
	}

	// --------------------------------------------------
	// Validate requested role
	// --------------------------------------------------

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
		return errors.New("role must be student or faculty")
	}

	// --------------------------------------------------
	// Validate PIN
	// --------------------------------------------------

	if len(input.PIN) != 6 {
		return errors.New("PIN must be exactly 6 digits")
	}

	for _, char := range input.PIN {
		if char < '0' || char > '9' {
			return errors.New("PIN must contain only digits")
		}
	}

	// --------------------------------------------------
	// Validate Google ID token
	// --------------------------------------------------

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
	emailAddress, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	picture, _ := payload.Claims["picture"].(string)

	subject = strings.TrimSpace(subject)
	emailAddress = strings.ToLower(strings.TrimSpace(emailAddress))
	picture = strings.TrimSpace(picture)

	if subject == "" || emailAddress == "" || !emailVerified {
		return errors.New(
			"Google must provide a verified email and valid subject",
		)
	}

	// --------------------------------------------------
	// Normalize optional profile fields
	// --------------------------------------------------

	var middleName *string
	var suffix *string
	var mobile *string
	var profileImage *string

	if input.MiddleName != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.MiddleName))

		if value != "" {
			middleName = &value
		}
	}

	if input.SuffixExtension != nil {
		value := strings.TrimSpace(*input.SuffixExtension)

		if value != "" {
			suffix = &value
		}
	}

	if input.MobileNumber != nil {
		value := strings.TrimSpace(*input.MobileNumber)

		if value != "" {
			mobile = &value
		}
	}

	if picture != "" {
		profileImage = &picture
	}

	// --------------------------------------------------
	// Hash PIN
	// --------------------------------------------------

	hashedPIN, err := hash.HashPassword(input.PIN)
	if err != nil {
		return errors.New("failed to secure PIN")
	}

	provider := "google"
	providerID := subject

	// --------------------------------------------------
	// Database transaction
	// --------------------------------------------------

	return db.Transaction(func(tx *gorm.DB) error {

		// ----------------------------------------------
		// Check Google identity
		// ----------------------------------------------

		_, err := repository.GetUserBySSO(
			tx,
			provider,
			providerID,
		)

		if err == nil {
			return errors.New("this Google account is already registered")
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("failed to check Google account")
		}

		// ----------------------------------------------
		// Check email
		// ----------------------------------------------

		_, err = repository.GetUserByIdentifier(
			tx,
			emailAddress,
		)

		if err == nil {
			return errors.New(
				"email already belongs to a CARDex account; account linking is required",
			)
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("failed to check email")
		}

		// ----------------------------------------------
		// Check username
		// ----------------------------------------------

		usernameExists, err := repository.UsernameExists(
			tx,
			input.Username,
			0,
		)

		if err != nil {
			return errors.New("failed to validate username")
		}

		if usernameExists {
			log.Println("Aha I see you Username has been duplicated in SSO register")
			return errors.New("username already exists")
		}

		// ----------------------------------------------
		// Check institutional ID
		// ----------------------------------------------

		institutionalIDExists, err := repository.InstitutionalIDExists(
			tx,
			input.InstitutionalID,
			0,
		)

		if err != nil {
			return errors.New("failed to validate institutional ID")
		}

		if institutionalIDExists {
			log.Println("Aha I see you Insti ID has been duplicated in SSO register")
			return errors.New("Student/Faculty ID already exists")
		}

		// ----------------------------------------------
		// Create CARDex user
		// ----------------------------------------------

		expirationDate := time.Now().AddDate(1, 0, 0)

		user := model.User{
			Username:       input.Username,
			EmailAddress:   emailAddress,
			StatusID:       3,
			RoleID:         RoleGuest,
			ExpirationDate: expirationDate,
			SSOProvider:    &provider,
			SSOProviderID:  &providerID,
		}

		if err := repository.CreateUser(tx, &user); err != nil {
			return errors.New("failed to create CARDex account")
		}

		// ----------------------------------------------
		// Create pending profile change request
		// ----------------------------------------------

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

		// ----------------------------------------------
		// Build recipient name
		// ----------------------------------------------

		nameParts := []string{
			input.FirstName,
		}

		if middleName != nil {
			nameParts = append(nameParts, *middleName)
		}

		nameParts = append(nameParts, input.LastName)

		if suffix != nil {
			nameParts = append(nameParts, *suffix)
		}

		recipientName := strings.Join(nameParts, " ")

		// ----------------------------------------------
		// Prepare email details
		// ----------------------------------------------

		details, err := json.Marshal(
			[]model.EmailOutboxDetail{
				{
					Label: "Username",
					Value: user.Username,
				},
				{
					Label: "Requested Role",
					Value: input.Role,
				},
				{
					Label: "Status",
					Value: "Pending Approval",
				},
			},
		)

		if err != nil {
			return fmt.Errorf(
				"marshal registration email details: %w",
				err,
			)
		}

		// ----------------------------------------------
		// Queue registration email
		// ----------------------------------------------

		outboxEmail := model.EmailOutbox{
			RecipientEmail: user.EmailAddress,
			RecipientName:  recipientName,
			Subject:        "CARDex Registration Received",
			Heading:        "Registration Received",
			Message:        "Your Google registration has been received. Your profile is pending administrator approval.",
			Details:        details,
			Status:         "PENDING",
		}

		if err := repository.EnqueueEmail(
			tx,
			&outboxEmail,
		); err != nil {
			return fmt.Errorf(
				"enqueue registration email: %w",
				err,
			)
		}

		// ----------------------------------------------
		// Audit log
		// ----------------------------------------------

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
