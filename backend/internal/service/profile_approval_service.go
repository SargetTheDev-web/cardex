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

		// ==================================================
		// INITIAL PROFILE CREATION
		// ==================================================

		if !profileExists {

			// --------------------------------------------------
			// Validate initial registration data
			// --------------------------------------------------

			if user.RoleID != RoleGuest {
				return errors.New(
					"invalid role for initial profile approval",
				)
			}

			if request.RequestedRoleID == nil {
				return errors.New(
					"requested role is missing",
				)
			}

			if *request.RequestedRoleID != RoleStudent &&
				*request.RequestedRoleID != RoleFaculty {

				return errors.New(
					"invalid requested role",
				)
			}

			if request.InstitutionalID == nil ||
				request.LastName == nil ||
				request.FirstName == nil {

				return errors.New(
					"initial profile is incomplete",
				)
			}

			if request.PINHash == nil {
				return errors.New(
					"PIN is missing",
				)
			}

			// --------------------------------------------------
			// Course validation
			// --------------------------------------------------

			// Students must have a course.
			if *request.RequestedRoleID == RoleStudent &&
				request.Course == nil {

				return errors.New(
					"course is required for student registration",
				)
			}

			// Faculty members do not require a course.
			if *request.RequestedRoleID == RoleFaculty {
				request.Course = nil
			}

			// --------------------------------------------------
			// Create user profile
			// --------------------------------------------------

			profile = &model.UserProfile{
				UserID:          request.UserID,
				InstitutionalID: *request.InstitutionalID,
				LastName:        *request.LastName,
				FirstName:       *request.FirstName,
				MiddleName:      request.MiddleName,
				SuffixExtension: request.SuffixExtension,
				Course:          request.Course,
				MobileNumber:    request.MobileNumber,
				BirthDate:       request.BirthDate,
				CreatedBy:       &adminID,
			}

			if err := repository.CreateUserProfile(
				tx,
				profile,
			); err != nil {
				return err
			}

			// --------------------------------------------------
			// Activate requested role and save PIN
			// --------------------------------------------------

			if err := repository.UpdateUserAccount(
				tx,
				request.UserID,
				map[string]interface{}{
					"role_id":    *request.RequestedRoleID,
					"pin_hash":   *request.PINHash,
					"updated_by": adminID,
				},
			); err != nil {
				return err
			}

		} else {

			// ==================================================
			// EXISTING PROFILE UPDATE
			// ==================================================

			updates := make(map[string]interface{})

			// --------------------------------------------------
			// Profile fields
			// --------------------------------------------------

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

			if request.Course != nil {
				updates["course"] =
					*request.Course
			}

			if request.MobileNumber != nil {
				updates["mobile_number"] =
					*request.MobileNumber
			}

			if request.BirthDate != nil {
				updates["birth_date"] =
					*request.BirthDate
			}

			// --------------------------------------------------
			// Apply profile updates
			// --------------------------------------------------

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

		// ==================================================
		// ACCOUNT INFORMATION
		// ==================================================

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

		// ==================================================
		// MARK REQUEST AS APPROVED
		// ==================================================

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

		// ==================================================
		// AUDIT LOG
		// ==================================================

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
