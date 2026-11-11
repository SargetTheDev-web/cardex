// internal/service/profile_rejection_service.go

package service

import (
	"encoding/json"
	"errors"
	"fmt"
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
		request, err := repository.GetProfileChangeRequest(tx, changeRequestID)
		if err != nil {
			return errors.New("profile change request not found")
		}

		if request.Status != "PENDING" {
			return errors.New("profile change request has already been processed")
		}

		// Load the request owner's current account details for notification.
		user, err := repository.GetUserByID(tx, request.UserID)
		if err != nil {
			return errors.New("user not found")
		}

		if strings.TrimSpace(user.EmailAddress) == "" {
			return errors.New("user email is required to send the rejection notification")
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
			return errors.New("profile change request was already processed")
		}

		// Create the in-app notification.
		referenceType := "PROFILE_CHANGE"
		referenceID := changeRequestID

		notification := model.Notification{
			UserID:        request.UserID,
			Type:          "PROFILE_CHANGE",
			Channel:       repository.NotificationChannelInApp,
			Subject:       "Profile Change Rejected",
			Message:       "Your profile change request was rejected. Reason: " + reason,
			IsRead:        false,
			ReferenceType: &referenceType,
			ReferenceID:   &referenceID,
		}

		if err := tx.Create(&notification).Error; err != nil {
			return fmt.Errorf("create rejection notification: %w", err)
		}

		// Queue the email; the existing outbox worker sends it.
		details, err := json.Marshal([]model.EmailOutboxDetail{
			{Label: "Request ID", Value: fmt.Sprintf("%d", changeRequestID)},
			{Label: "Status", Value: "Rejected"},
			{Label: "Reason", Value: reason},
		})

		recipientName := "Library Borrower"

		profile, err := repository.GetUserProfileByUserID(
			tx,
			request.UserID,
		)
		if err == nil {
			nameParts := []string{
				profile.FirstName,
				profile.LastName,
			}

			if profile.MiddleName != nil {
				nameParts = append(nameParts, *profile.MiddleName)
			}

			if profile.SuffixExtension != nil {
				nameParts = append(nameParts, *profile.SuffixExtension)
			}

			var nonEmptyParts []string

			for _, part := range nameParts {
				if part = strings.TrimSpace(part); part != "" {
					nonEmptyParts = append(nonEmptyParts, part)
				}
			}

			if len(nonEmptyParts) > 0 {
				recipientName = strings.Join(nonEmptyParts, " ")
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("retrieve user profile: %w", err)
		}

		if err != nil {
			return fmt.Errorf("marshal rejection email details: %w", err)
		}

		email := model.EmailOutbox{
			RecipientEmail: user.EmailAddress,
			RecipientName:  recipientName,
			Subject:        "CARDex Profile Change Rejected",
			Heading:        "Profile Change Rejected",
			Message:        "We regret to inform you that we are unable to approve your registration at this time. Please log in to the student portal to update your details, or visit the library desk for assistance. Your profile change request was rejected. Reason: " + reason,
			Details:        details,
			Status:         "PENDING",
		}

		if err := repository.EnqueueEmail(tx, &email); err != nil {
			return fmt.Errorf("enqueue rejection email: %w", err)
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
