// internal/service/profile_approval_service.go

package service

import (
	"errors"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

func ApproveProfileChange(
	db *gorm.DB,
	changeRequestID int64,
	adminID int,
	ip string,
	userAgent string,
) error {

	return db.Transaction(func(tx *gorm.DB) error {

		// --------------------------------------------------
		// Get pending request
		// --------------------------------------------------

		request, err := repository.GetProfileChangeRequest(
			tx,
			changeRequestID,
		)

		if err != nil {
			return errors.New(
				"profile change request not found",
			)
		}

		if request.Status != "PENDING" {
			return errors.New(
				"profile change request has already been processed",
			)
		}

		// --------------------------------------------------
		// Get user
		// --------------------------------------------------

		user, err := repository.GetUserByID(
			tx,
			request.UserID,
		)

		if err != nil {
			return errors.New("user not found")
		}

		// --------------------------------------------------
		// Check existing profile
		// --------------------------------------------------

		profile, err := repository.GetUserProfileByUserID(
			tx,
			request.UserID,
		)

		profileExists := err == nil

		if err != nil &&
			!errors.Is(err, gorm.ErrRecordNotFound) {

			return errors.New(
				"failed to retrieve user profile",
			)
		}

		// --------------------------------------------------
		// First profile creation
		// --------------------------------------------------

		if !profileExists {

			if request.InstitutionalID == nil ||
				request.LastName == nil ||
				request.FirstName == nil {

				return errors.New(
					"initial profile is incomplete",
				)
			}

			profile = &model.UserProfile{
				UserID:          request.UserID,
				InstitutionalID: *request.InstitutionalID,
				LastName:        *request.LastName,
				FirstName:       *request.FirstName,
				MiddleName:      request.MiddleName,
				SuffixExtension: request.SuffixExtension,
				MobileNumber:    request.MobileNumber,
				BirthDate:       request.BirthDate,
			}

			if err := repository.CreateUserProfile(
				tx,
				profile,
			); err != nil {
				return err
			}

		} else {

			// --------------------------------------------------
			// Existing profile update
			// --------------------------------------------------

			updates := make(map[string]interface{})

			if request.InstitutionalID != nil {
				updates["institutional_id"] =
					*request.InstitutionalID
			}

			if request.LastName != nil {
				updates["last_name"] =
					*request.LastName
			}

			if request.FirstName != nil {
				updates["first_name"] =
					*request.FirstName
			}

			if request.MiddleName != nil {
				updates["middle_name"] =
					*request.MiddleName
			}

			if request.SuffixExtension != nil {
				updates["suffix_extension"] =
					*request.SuffixExtension
			}

			if request.MobileNumber != nil {
				updates["mobile_number"] =
					*request.MobileNumber
			}

			if request.BirthDate != nil {
				updates["birth_date"] =
					*request.BirthDate
			}

			if len(updates) > 0 {
				updates["updated_by"] = adminID

				if err := repository.UpdateUserProfile(
					tx,
					request.UserID,
					updates,
				); err != nil {
					return err
				}
			}
		}

		// --------------------------------------------------
		// Update account information
		// --------------------------------------------------

		accountUpdates := make(map[string]interface{})

		if request.Username != nil {
			accountUpdates["username"] =
				*request.Username
		}

		if request.EmailAddress != nil {
			accountUpdates["email_address"] =
				*request.EmailAddress
		}

		if len(accountUpdates) > 0 {
			accountUpdates["updated_by"] = adminID

			if err := repository.UpdateUserAccount(
				tx,
				request.UserID,
				accountUpdates,
			); err != nil {
				return err
			}
		}

		// --------------------------------------------------
		// Mark request approved
		// --------------------------------------------------

		now := time.Now()

		result := tx.
			Model(&model.ProfileChangeRequest{}).
			Where(
				"change_request_id = ? AND status = ?",
				changeRequestID,
				"PENDING",
			).
			Updates(map[string]interface{}{
				"status":      "APPROVED",
				"reviewed_by": adminID,
				"reviewed_at": now,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return errors.New(
				"profile change request was already processed",
			)
		}

		// --------------------------------------------------
		// Audit
		// --------------------------------------------------

		if err := repository.InsertAuditLog(
			tx,
			&user.UserID,
			8, // APPROVE
			2, // USER_MANAGEMENT
			"Profile change request approved",
			ip,
			userAgent,
		); err != nil {
			return err
		}

		return nil
	})
}
