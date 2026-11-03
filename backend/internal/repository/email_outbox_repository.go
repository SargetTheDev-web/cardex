// internal/repository/email_outbox_repository.go

package repository

import (
	"errors"
	"fmt"
	"time"

	model "backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func EnqueueEmail(db *gorm.DB, email *model.EmailOutbox) error {
	if email == nil {
		return errors.New("email outbox record is required")
	}
	return db.Create(email).Error
}

// ClaimEmail atomically claims one eligible message.
// SKIP LOCKED allows multiple workers without claiming the same row.
func ClaimEmail(db *gorm.DB) (*model.EmailOutbox, error) {
	var email model.EmailOutbox

	err := db.Transaction(func(tx *gorm.DB) error {
		// Empty queue is normal polling behavior; don't log it as an error.
		query := tx.Session(&gorm.Session{
			Logger: tx.Logger.LogMode(logger.Silent),
		})

		if err := query.
			Clauses(clause.Locking{
				Strength: "UPDATE",
				Options:  "SKIP LOCKED",
			}).
			Where("status IN ?", []string{"PENDING", "FAILED"}).
			Where("next_attempt_at <= NOW()").
			Order("next_attempt_at ASC").
			Order("email_outbox_id ASC").
			First(&email).Error; err != nil {
			return err
		}

		result := tx.Model(&model.EmailOutbox{}).
			Where("email_outbox_id = ?", email.EmailOutboxID).
			Updates(map[string]any{
				"status":        "PROCESSING",
				"claimed_at":    gorm.Expr("NOW()"),
				"attempt_count": gorm.Expr("attempt_count + 1"),
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("failed to claim outbox email")
		}

		email.Status = "PROCESSING"
		email.AttemptCount++
		now := time.Now()
		email.ClaimedAt = &now

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &email, nil
}

func MarkEmailSent(db *gorm.DB, emailOutboxID int64) error {
	result := db.Model(&model.EmailOutbox{}).
		Where(
			"email_outbox_id = ? AND status = ?",
			emailOutboxID,
			"PROCESSING",
		).
		Updates(map[string]any{
			"status":     "SENT",
			"sent_at":    gorm.Expr("NOW()"),
			"last_error": nil,
			"claimed_at": nil,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return fmt.Errorf(
			"mark email outbox item %d as sent: expected 1 row, updated %d",
			emailOutboxID,
			result.RowsAffected,
		)
	}

	return nil
}

func MarkEmailFailed(
	db *gorm.DB,
	emailOutboxID int64,
	lastError string,
	nextAttemptAt time.Time,
) error {
	result := db.Model(&model.EmailOutbox{}).
		Where(
			"email_outbox_id = ? AND status = ?",
			emailOutboxID,
			"PROCESSING",
		).
		Updates(map[string]any{
			"status":          "FAILED",
			"last_error":      lastError,
			"next_attempt_at": nextAttemptAt,
			"claimed_at":      nil,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return fmt.Errorf(
			"mark email outbox item %d as failed: expected 1 row, updated %d",
			emailOutboxID,
			result.RowsAffected,
		)
	}

	return nil
}

// RecoverStaleEmailClaims returns emails left in PROCESSING after a worker crash
// to the retry queue.
func RecoverStaleEmailClaims(db *gorm.DB, olderThan time.Duration) error {
	return db.Model(&model.EmailOutbox{}).
		Where("status = ?", "PROCESSING").
		Where("claimed_at IS NOT NULL").
		Where(
			"claimed_at < NOW() - (? * INTERVAL '1 second')",
			olderThan.Seconds(),
		).
		Updates(map[string]any{
			"status":          "FAILED",
			"claimed_at":      nil,
			"next_attempt_at": gorm.Expr("NOW()"),
			"last_error":      "Recovered stale PROCESSING email after worker interruption",
		}).Error
}
