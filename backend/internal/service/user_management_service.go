// internal/service/user_management_service.go

package service

import (
	"errors"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"
	"backend/pkg/hash"

	"gorm.io/gorm"
)

type CreateUserInput struct {
	Username        string
	EmailAddress    string
	Password        string
	RoleID          int
	InstitutionalID string
	LastName        string
	FirstName       string
	MiddleName      *string
	SuffixExtension *string
	Course          *string
	MobileNumber    *string
	BirthDate       *string
}

type UpdateUserInput struct {
	Username        *string
	EmailAddress    *string
	InstitutionalID *string
	LastName        *string
	FirstName       *string
	MiddleName      *string
	SuffixExtension *string
	Course          *string
	MobileNumber    *string
	BirthDate       *string
}

func GetUsers(db *gorm.DB) ([]repository.UserListRow, error) {
	return repository.GetUserList(db)
}

func CreateUser(
	db *gorm.DB,
	adminID int,
	input CreateUserInput,
	ip string,
	userAgent string,
) error {

	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.EmailAddress)
	institutionalID := strings.TrimSpace(input.InstitutionalID)
	lastName := strings.TrimSpace(input.LastName)
	firstName := strings.TrimSpace(input.FirstName)

	if username == "" {
		return errors.New("username is required")
	}

	if email == "" {
		return errors.New("email address is required")
	}

	if input.Password == "" {
		return errors.New("password is required")
	}

	if len(input.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	if input.RoleID <= 0 {
		return errors.New("role is required")
	}

	if institutionalID == "" {
		return errors.New("institutional ID is required")
	}

	if lastName == "" {
		return errors.New("last name is required")
	}

	if firstName == "" {
		return errors.New("first name is required")
	}

	usernameExists, err := repository.UsernameExists(db, username, 0)
	if err != nil {
		return err
	}

	if usernameExists {
		return errors.New("username already exists")
	}

	emailExists, err := repository.EmailExists(db, email, 0)
	if err != nil {
		return err
	}

	if emailExists {
		return errors.New("email address already exists")
	}

	institutionalIDExists, err :=
		repository.InstitutionalIDExists(db, institutionalID, 0)

	if err != nil {
		return err
	}

	if institutionalIDExists {
		return errors.New("institutional ID already exists")
	}

	role, err := repository.GetUserRoleByID(db, input.RoleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("role not found")
		}

		return err
	}

	if !role.IsActive {
		return errors.New("selected role is inactive")
	}

	var birthDate *time.Time

	if input.BirthDate != nil {
		value := strings.TrimSpace(*input.BirthDate)

		if value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				return errors.New("birth date must use YYYY-MM-DD format")
			}

			birthDate = &parsed
		}
	}

	passwordHash, err := hash.HashPassword(input.Password)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {

		user := &model.User{
			Username:        username,
			EmailAddress:    email,
			PasswordHash:    passwordHash,
			StatusID:        3,
			RoleID:          input.RoleID,
			LoginRetryCount: 0,
			ExpirationDate:  time.Now().AddDate(1, 0, 0),
			CreatedBy:       &adminID,
			UpdatedBy:       &adminID,
		}

		if err := repository.CreateUser(tx, user); err != nil {
			return err
		}

		profile := &model.UserProfile{
			UserID:          user.UserID,
			InstitutionalID: institutionalID,
			LastName:        lastName,
			FirstName:       firstName,
			MiddleName:      input.MiddleName,
			SuffixExtension: input.SuffixExtension,
			Course:          input.Course,
			MobileNumber:    input.MobileNumber,
			BirthDate:       birthDate,
			CreatedBy:       &adminID,
			UpdatedBy:       &adminID,
		}

		if err := repository.CreateUserProfile(tx, profile); err != nil {
			return err
		}

		/*
			TEMPORARY AUDIT MAPPING

			Action/module IDs need to be matched against your
			audit_action and audit_module tables.

			Do not change these blindly if your existing audit
			records use different IDs.
		*/
		if err := repository.InsertAuditLog(
			tx,
			&user.UserID,
			8,
			2,
			"User account created by administrator",
			ip,
			userAgent,
		); err != nil {
			return err
		}

		return nil
	})
}

func UpdateUser(
	db *gorm.DB,
	adminID int,
	userID int,
	input UpdateUserInput,
	ip string,
	userAgent string,
) error {

	user, err := repository.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}

		return err
	}

	_ = user

	/*
		User Management edits an existing complete user account.

		A profile-less GUEST belongs to the registration approval
		flow, where the initial profile is created.
	*/
	_, err = repository.GetUserProfileByUserID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user profile not found")
		}

		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {

		accountUpdates := make(map[string]interface{})
		profileUpdates := make(map[string]interface{})

		if input.Username != nil {
			username := strings.TrimSpace(*input.Username)

			if username == "" {
				return errors.New("username cannot be empty")
			}

			exists, err :=
				repository.UsernameExists(tx, username, userID)

			if err != nil {
				return err
			}

			if exists {
				return errors.New("username already exists")
			}

			accountUpdates["username"] = username
		}

		if input.EmailAddress != nil {
			email := strings.TrimSpace(*input.EmailAddress)

			if email == "" {
				return errors.New("email address cannot be empty")
			}

			exists, err :=
				repository.EmailExists(tx, email, userID)

			if err != nil {
				return err
			}

			if exists {
				return errors.New("email address already exists")
			}

			accountUpdates["email_address"] = email
		}

		if input.InstitutionalID != nil {
			institutionalID :=
				strings.TrimSpace(*input.InstitutionalID)

			if institutionalID == "" {
				return errors.New("institutional ID cannot be empty")
			}

			exists, err :=
				repository.InstitutionalIDExists(
					tx,
					institutionalID,
					userID,
				)

			if err != nil {
				return err
			}

			if exists {
				return errors.New("institutional ID already exists")
			}

			profileUpdates["institutional_id"] = institutionalID
		}

		if input.LastName != nil {
			lastName := strings.TrimSpace(*input.LastName)

			if lastName == "" {
				return errors.New("last name cannot be empty")
			}

			profileUpdates["last_name"] = lastName
		}

		if input.FirstName != nil {
			firstName := strings.TrimSpace(*input.FirstName)

			if firstName == "" {
				return errors.New("first name cannot be empty")
			}

			profileUpdates["first_name"] = firstName
		}

		if input.MiddleName != nil {
			profileUpdates["middle_name"] = *input.MiddleName
		}

		if input.SuffixExtension != nil {
			profileUpdates["suffix_extension"] =
				*input.SuffixExtension
		}

		if input.Course != nil {
			profileUpdates["course"] = *input.Course
		}

		if input.MobileNumber != nil {
			profileUpdates["mobile_number"] =
				*input.MobileNumber
		}

		if input.BirthDate != nil {
			value := strings.TrimSpace(*input.BirthDate)

			if value == "" {
				profileUpdates["birth_date"] = nil
			} else {
				parsed, err :=
					time.Parse("2006-01-02", value)

				if err != nil {
					return errors.New(
						"birth date must use YYYY-MM-DD format",
					)
				}

				profileUpdates["birth_date"] = parsed
			}
		}

		if len(accountUpdates) > 0 {
			accountUpdates["updated_by"] = adminID

			if err := repository.UpdateUserAccount(
				tx,
				userID,
				accountUpdates,
			); err != nil {
				return err
			}
		}

		if len(profileUpdates) > 0 {
			profileUpdates["updated_by"] = adminID

			if err := repository.UpdateUserProfile(
				tx,
				userID,
				profileUpdates,
			); err != nil {
				return err
			}
		}

		if err := repository.InsertAuditLog(
			tx,
			&userID,
			8,
			2,
			"User account updated by administrator",
			ip,
			userAgent,
		); err != nil {
			return err
		}

		return nil
	})
}

func ChangeUserStatus(
	db *gorm.DB,
	adminID int,
	userID int,
	statusID int,
	ip string,
	userAgent string,
) error {

	if statusID <= 0 {
		return errors.New("status is required")
	}

	user, err := repository.GetUserByID(db, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}

		return err
	}

	status, err := repository.GetUserStatus(db, statusID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("status not found")
		}

		return err
	}

	if user.StatusID == statusID {
		return errors.New("user already has this status")
	}

	return db.Transaction(func(tx *gorm.DB) error {

		err := repository.UpdateUserAccount(
			tx,
			userID,
			map[string]interface{}{
				"status_id":  statusID,
				"updated_by": adminID,
			},
		)

		if err != nil {
			return err
		}

		if err := repository.InsertAuditLog(
			tx,
			&userID,
			8,
			2,
			"User account status changed to "+status.StatusName,
			ip,
			userAgent,
		); err != nil {
			return err
		}

		return nil
	})
}
