// internal/service/profile_rejection_service.go

package service

import (
	"errors"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

func RejectProfileChange(
	db *gorm.DB,
	changeRequestID int64,
	adminID int,
	reason string,
	ip string,
	userAgent string,
) error {

	reason = strings.TrimSpace(reason)

	if reason == "" {
		return errors.New("rejection reason is required")
	}

	return db.Transaction(func(tx *gorm.DB) error {

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

		now := time.Now()

		result := tx.
			Model(&model.ProfileChangeRequest{}).
			Where(
				"change_request_id = ? AND status = ?",
				changeRequestID,
				"PENDING",
			).
			Updates(map[string]interface{}{
				"status":           "REJECTED",
				"reviewed_by":      adminID,
				"reviewed_at":      now,
				"rejection_reason": reason,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return errors.New(
				"profile change request was already processed",
			)
		}

		if err := repository.InsertAuditLog(
			tx,
			&request.UserID,
			3, // UPDATE
			2, // USER_MANAGEMENT
			"Profile change request rejected: "+reason,
			ip,
			userAgent,
		); err != nil {
			return err
		}

		return nil
	})
}
