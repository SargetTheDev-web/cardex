// internal/service/email_outbox_runner.go

package service

import (
	"context"
	"errors"
	"log"
	"time"

	"backend/internal/repository"

	"gorm.io/gorm"
)

func (s *NotificationService) RunEmailOutboxWorker(ctx context.Context) {
	const (
		pollInterval = 10 * time.Second
		staleAfter   = 15 * time.Minute
	)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
		log.Printf("email outbox recovery failed: %v", err)
	}

	for {
		// Process queued emails until the queue is empty.
		for {
			if ctx.Err() != nil {
				return
			}

			err := s.ProcessNextOutboxEmail(ctx)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break // Nothing currently due.
			}
			if err != nil {
				log.Printf("email outbox worker: %v", err)
				break // Avoid a tight retry loop on persistent errors.
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
				log.Printf("email outbox recovery failed: %v", err)
			}
		}
	}
}
