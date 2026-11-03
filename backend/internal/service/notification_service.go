// internal/service/notification_service.go

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

type NotificationEmail struct {
	RecipientName string
	Recipient     string
	Subject       string
	Message       string
	ActionURL     string
}

type NotificationEmailSender interface {
	SendNotificationEmail(
		ctx context.Context,
		email NotificationEmail,
	) error
}

type NotificationService struct {
	db     *gorm.DB
	mailer NotificationEmailSender
}

func NewNotificationService(
	db *gorm.DB,
	mailer NotificationEmailSender,
) *NotificationService {
	return &NotificationService{
		db:     db,
		mailer: mailer,
	}
}

type CreateNotificationInput struct {
	UserID        int
	Type          string
	Subject       string
	Message       string
	ReferenceType *string
	ReferenceID   *int64
}

func (s *NotificationService) CreateInApp(
	tx *gorm.DB,
	input CreateNotificationInput,
) error {
	if err := validateNotificationInput(input); err != nil {
		return err
	}

	db := s.db
	if tx != nil {
		db = tx
	}

	notification := model.Notification{
		UserID:        input.UserID,
		Type:          input.Type,
		Channel:       repository.NotificationChannelInApp,
		Subject:       strings.TrimSpace(input.Subject),
		Message:       strings.TrimSpace(input.Message),
		IsRead:        false,
		ReferenceType: input.ReferenceType,
		ReferenceID:   input.ReferenceID,
	}

	return repository.CreateNotification(db, &notification)
}

func (s *NotificationService) CreateInAppForUsers(
	tx *gorm.DB,
	userIDs []int,
	input CreateNotificationInput,
) error {
	if len(userIDs) == 0 {
		return errors.New("at least one recipient is required")
	}

	db := s.db
	if tx != nil {
		db = tx
	}

	return db.Transaction(func(transaction *gorm.DB) error {
		for _, userID := range userIDs {
			recipientInput := input
			recipientInput.UserID = userID

			if err := s.CreateInApp(transaction, recipientInput); err != nil {
				return fmt.Errorf(
					"create notification for user %d: %w",
					userID,
					err,
				)
			}
		}

		return nil
	})
}

func validateNotificationInput(
	input CreateNotificationInput,
) error {
	if input.UserID <= 0 {
		return errors.New("user_id must be positive")
	}

	if strings.TrimSpace(input.Type) == "" {
		return errors.New("notification type is required")
	}

	if strings.TrimSpace(input.Subject) == "" {
		return errors.New("notification subject is required")
	}

	if strings.TrimSpace(input.Message) == "" {
		return errors.New("notification message is required")
	}

	if (input.ReferenceType == nil) != (input.ReferenceID == nil) {
		return errors.New(
			"reference_type and reference_id must be provided together",
		)
	}

	if input.ReferenceType != nil {
		referenceType := strings.TrimSpace(*input.ReferenceType)
		if referenceType == "" {
			return errors.New("reference_type cannot be empty")
		}

		if *input.ReferenceID <= 0 {
			return errors.New("reference_id must be positive")
		}
	}

	return nil
}
