// internal/service/email_outbox_worker.go

package service

import (
	model "backend/internal/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"backend/internal/mail"
	"backend/internal/repository"

	"gorm.io/gorm"
)

const (
	maxEmailAttempts = 8
)

func (s *NotificationService) ProcessNextOutboxEmail(ctx context.Context) error {
	email, err := repository.ClaimEmail(s.db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		return fmt.Errorf("claim email outbox item: %w", err)
	}

	var details []model.EmailOutboxDetail
	if err := json.Unmarshal(email.Details, &details); err != nil {
		return s.failOutboxEmail(email, fmt.Errorf("decode email details: %w", err))
	}

	err = mail.SendNotificationEmail(email.RecipientEmail, mail.NotificationEmailData{
		RecipientName: email.RecipientName,
		Subject:       email.Subject,
		Heading:       email.Heading,
		Message:       email.Message,
		Details:       toMailDetails(details),
	})
	if err != nil {
		return s.failOutboxEmail(email, err)
	}

	if err := repository.MarkEmailSent(s.db, email.EmailOutboxID); err != nil {
		return fmt.Errorf("mark email %d as sent: %w", email.EmailOutboxID, err)
	}

	return nil
}

func (s *NotificationService) failOutboxEmail(
	email *model.EmailOutbox,
	sendErr error,
) error {
	if email.AttemptCount >= maxEmailAttempts {
		// Leave it FAILED for inspection; do not keep retrying indefinitely.
		if err := repository.MarkEmailFailed(
			s.db,
			email.EmailOutboxID,
			sendErr.Error(),
			time.Now().AddDate(100, 0, 0),
		); err != nil {
			return fmt.Errorf("record final email failure: %w", err)
		}
		return fmt.Errorf("email %d reached retry limit: %w", email.EmailOutboxID, sendErr)
	}

	// Exponential backoff: 1, 2, 4, 8... minutes, capped at 6 hours.
	delay := time.Minute * time.Duration(1<<min(email.AttemptCount-1, 8))
	if delay > 6*time.Hour {
		delay = 6 * time.Hour
	}

	if err := repository.MarkEmailFailed(
		s.db,
		email.EmailOutboxID,
		sendErr.Error(),
		time.Now().Add(delay),
	); err != nil {
		return fmt.Errorf("record email failure: %w", err)
	}

	return fmt.Errorf("send email %d: %w", email.EmailOutboxID, sendErr)
}

func toMailDetails(details []model.EmailOutboxDetail) []mail.NotificationEmailDetail {
	result := make([]mail.NotificationEmailDetail, 0, len(details))
	for _, detail := range details {
		result = append(result, mail.NotificationEmailDetail{
			Label: detail.Label,
			Value: detail.Value,
		})
	}
	return result
}
