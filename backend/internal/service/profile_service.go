// internal/service/profile_service.go

package service

import (
	"errors"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

type ProfileUpdateRequest struct {
	Username     *string `json:"username"`
	EmailAddress *string `json:"email_address"`

	InstitutionalID *string `json:"institutional_id"`
	LastName        *string `json:"last_name"`
	FirstName       *string `json:"first_name"`
	MiddleName      *string `json:"middle_name"`
	SuffixExtension *string `json:"suffix_extension"`
	MobileNumber    *string `json:"mobile_number"`
	BirthDate       *string `json:"birth_date"`
}

func RequestProfileUpdate(
	db *gorm.DB,
	userID int,
	req ProfileUpdateRequest,
	ip string,
	userAgent string,
) error {

	return db.Transaction(func(tx *gorm.DB) error {

		// --------------------------------------------------
		// Get user
		// --------------------------------------------------

		user, err := repository.GetUserByID(tx, userID)
		if err != nil {
			return errors.New("user not found")
		}

		// --------------------------------------------------
		// Get existing profile
		//
		// Profile may not exist yet.
		// This is allowed because the first approved
		// change request creates the profile.
		// --------------------------------------------------

		profile, profileErr := repository.GetUserProfileByUserID(
			tx,
			userID,
		)

		if profileErr != nil &&
			!errors.Is(profileErr, gorm.ErrRecordNotFound) {

			return errors.New("failed to retrieve profile")
		}

		// --------------------------------------------------
		// Prevent multiple pending requests
		// --------------------------------------------------

		existing, err := repository.GetPendingProfileChange(
			tx,
			userID,
		)

		if err == nil && existing != nil {
			return errors.New(
				"you already have a pending profile change request",
			)
		}

		// --------------------------------------------------
		// Prepare change request
		// --------------------------------------------------

		change := model.ProfileChangeRequest{
			UserID: userID,
			Status: "PENDING",
		}

		hasChanges := false

		// ==================================================
		// ACCOUNT INFORMATION
		// ==================================================

		// --------------------------------------------------
		// Username
		// --------------------------------------------------

		if req.Username != nil {

			value := strings.TrimSpace(*req.Username)

			if value == "" {
				return errors.New("username cannot be empty")
			}

			if value != user.Username {

				exists, err := repository.UsernameExists(
					tx,
					value,
					userID,
				)

				if err != nil {
					return errors.New(
						"failed to validate username",
					)
				}

				if exists {
					return errors.New(
						"username already exists",
					)
				}

				change.Username = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// Email
		// --------------------------------------------------

		if req.EmailAddress != nil {

			value := strings.ToLower(
				strings.TrimSpace(*req.EmailAddress),
			)

			if value == "" {
				return errors.New("email cannot be empty")
			}

			if value != user.EmailAddress {

				exists, err := repository.EmailExists(
					tx,
					value,
					userID,
				)

				if err != nil {
					return errors.New(
						"failed to validate email",
					)
				}

				if exists {
					return errors.New(
						"email address already exists",
					)
				}

				change.EmailAddress = &value
				hasChanges = true
			}
		}

		// ==================================================
		// PROFILE INFORMATION
		// ==================================================

		// --------------------------------------------------
		// Institutional ID
		// --------------------------------------------------

		if req.InstitutionalID != nil {

			value := strings.TrimSpace(
				*req.InstitutionalID,
			)

			if value == "" {
				return errors.New(
					"institutional ID cannot be empty",
				)
			}

			current := ""

			if profile != nil {
				current = strings.TrimSpace(
					profile.InstitutionalID,
				)
			}

			if value != current {

				exists, err := repository.InstitutionalIDExists(
					tx,
					value,
					userID,
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

				change.InstitutionalID = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// Last Name
		// --------------------------------------------------

		if req.LastName != nil {

			value := strings.TrimSpace(*req.LastName)

			if value == "" {
				return errors.New(
					"last name cannot be empty",
				)
			}

			current := ""

			if profile != nil {
				current = strings.TrimSpace(
					profile.LastName,
				)
			}

			if value != current {

				change.LastName = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// First Name
		// --------------------------------------------------

		if req.FirstName != nil {

			value := strings.TrimSpace(*req.FirstName)

			if value == "" {
				return errors.New(
					"first name cannot be empty",
				)
			}

			current := ""

			if profile != nil {
				current = strings.TrimSpace(
					profile.FirstName,
				)
			}

			if value != current {

				change.FirstName = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// Middle Name
		// --------------------------------------------------

		if req.MiddleName != nil {

			value := strings.TrimSpace(
				*req.MiddleName,
			)

			current := ""

			if profile != nil &&
				profile.MiddleName != nil {

				current = strings.TrimSpace(
					*profile.MiddleName,
				)
			}

			if value != current {

				change.MiddleName = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// Suffix Extension
		// --------------------------------------------------

		if req.SuffixExtension != nil {

			value := strings.TrimSpace(
				*req.SuffixExtension,
			)

			current := ""

			if profile != nil &&
				profile.SuffixExtension != nil {

				current = strings.TrimSpace(
					*profile.SuffixExtension,
				)
			}

			if value != current {

				change.SuffixExtension = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// Mobile Number
		// --------------------------------------------------

		if req.MobileNumber != nil {

			value := strings.TrimSpace(
				*req.MobileNumber,
			)

			current := ""

			if profile != nil &&
				profile.MobileNumber != nil {

				current = strings.TrimSpace(
					*profile.MobileNumber,
				)
			}

			if value != current {

				change.MobileNumber = &value
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// Birth Date
		// --------------------------------------------------

		if req.BirthDate != nil {

			value := strings.TrimSpace(
				*req.BirthDate,
			)

			if value == "" {
				return errors.New(
					"birth date cannot be empty",
				)
			}

			parsedDate, err := time.Parse(
				"2006-01-02",
				value,
			)

			if err != nil {
				return errors.New(
					"birth date must use YYYY-MM-DD format",
				)
			}

			currentDate := ""

			if profile != nil &&
				profile.BirthDate != nil {

				currentDate = profile.BirthDate.Format(
					"2006-01-02",
				)
			}

			newDate := parsedDate.Format(
				"2006-01-02",
			)

			if newDate != currentDate {

				change.BirthDate = &parsedDate
				hasChanges = true
			}
		}

		// --------------------------------------------------
		// No changes
		// --------------------------------------------------

		if !hasChanges {
			return errors.New("no changes detected")
		}

		// --------------------------------------------------
		// Save pending request
		// --------------------------------------------------

		if err := repository.CreateProfileChangeRequest(
			tx,
			&change,
		); err != nil {
			return err
		}

		// --------------------------------------------------
		// Audit Trail
		// --------------------------------------------------

		if err := repository.InsertAuditLog(
			tx,
			&user.UserID,
			3, // UPDATE
			2, // USER_MANAGEMENT
			"Profile change request submitted",
			ip,
			userAgent,
		); err != nil {
			return err
		}

		return nil
	})
}
