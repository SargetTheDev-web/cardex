// internal/repository/profile_change_repository.go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreateProfileChangeRequest(
	db *gorm.DB,
	request *model.ProfileChangeRequest,
) error {
	return db.Create(request).Error
}

func GetPendingProfileChange(
	db *gorm.DB,
	userID int,
) (*model.ProfileChangeRequest, error) {

	var request model.ProfileChangeRequest

	err := db.
		Where(
			"user_id = ? AND status = ?",
			userID,
			"PENDING",
		).
		Order("created_at DESC").
		First(&request).Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func GetProfileChangeRequest(
	db *gorm.DB,
	changeRequestID int64,
) (*model.ProfileChangeRequest, error) {

	var request model.ProfileChangeRequest

	err := db.
		Where(
			"change_request_id = ?",
			changeRequestID,
		).
		First(&request).Error

	if err != nil {
		return nil, err
	}

	return &request, nil
}

// ApproveProfileChange applies the requested changes
// and marks the request as APPROVED.
//
// Everything happens inside one database transaction.
func ApproveProfileChange(
	db *gorm.DB,
	changeRequestID int64,
	adminID int,
) error {

	return db.Transaction(func(tx *gorm.DB) error {

		// --------------------------------------------------
		// 1. Get pending request
		// --------------------------------------------------

		var request model.ProfileChangeRequest

		err := tx.
			Where(
				"change_request_id = ? AND status = ?",
				changeRequestID,
				"PENDING",
			).
			First(&request).Error

		if err != nil {
			return err
		}

		// --------------------------------------------------
		// 2. Build user updates
		// --------------------------------------------------

		userUpdates := make(map[string]interface{})

		if request.Username != nil {
			userUpdates["username"] = *request.Username
		}

		if request.EmailAddress != nil {
			userUpdates["email_address"] = *request.EmailAddress
		}

		// --------------------------------------------------
		// 3. Build profile updates
		// --------------------------------------------------

		profileUpdates := make(map[string]interface{})

		if request.InstitutionalID != nil {
			profileUpdates["institutional_id"] = *request.InstitutionalID
		}

		if request.LastName != nil {
			profileUpdates["last_name"] = *request.LastName
		}

		if request.FirstName != nil {
			profileUpdates["first_name"] = *request.FirstName
		}

		if request.MiddleName != nil {
			profileUpdates["middle_name"] = *request.MiddleName
		}

		if request.SuffixExtension != nil {
			profileUpdates["suffix_extension"] = *request.SuffixExtension
		}

		if request.MobileNumber != nil {
			profileUpdates["mobile_number"] = *request.MobileNumber
		}

		if request.BirthDate != nil {
			profileUpdates["birth_date"] = *request.BirthDate
		}

		// --------------------------------------------------
		// 4. Apply user changes
		// --------------------------------------------------

		if len(userUpdates) > 0 {

			userUpdates["updated_by"] = adminID

			if err := tx.
				Table(`"user"`).
				Where("user_id = ?", request.UserID).
				Updates(userUpdates).Error; err != nil {

				return err
			}
		}

		// --------------------------------------------------
		// 5. Apply profile changes
		// --------------------------------------------------

		if len(profileUpdates) > 0 {

			profileUpdates["updated_by"] = adminID

			if err := tx.
				Table("user_profile").
				Where("user_id = ?", request.UserID).
				Updates(profileUpdates).Error; err != nil {

				return err
			}
		}

		// --------------------------------------------------
		// 6. Mark request APPROVED
		// --------------------------------------------------

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
				"reviewed_at": gorm.Expr("CURRENT_TIMESTAMP"),
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func RejectProfileChange(
	db *gorm.DB,
	changeRequestID int64,
	adminID int,
	reason string,
) error {

	result := db.
		Model(&model.ProfileChangeRequest{}).
		Where(
			"change_request_id = ? AND status = ?",
			changeRequestID,
			"PENDING",
		).
		Updates(map[string]interface{}{
			"status":           "REJECTED",
			"reviewed_by":      adminID,
			"reviewed_at":      gorm.Expr("CURRENT_TIMESTAMP"),
			"rejection_reason": reason,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
