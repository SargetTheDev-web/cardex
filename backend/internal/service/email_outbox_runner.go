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
		pollInterval     = 10 * time.Second
		recoveryInterval = 5 * time.Minute
		staleAfter       = 15 * time.Minute
	)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	recoveryTicker := time.NewTicker(recoveryInterval)
	defer recoveryTicker.Stop()

	if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
		log.Printf("email outbox recovery failed: %v", err)
	}

	for {
		for {
			if ctx.Err() != nil {
				return
			}

			err := s.ProcessNextOutboxEmail(ctx)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			if err != nil {
				log.Printf("email outbox worker: %v", err)
				break
			}
		}

		select {
		case <-ctx.Done():
			return

		case <-pollTicker.C:
			// Queue polling is handled at the top of the loop.

		case <-recoveryTicker.C:
			if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
				log.Printf("email outbox recovery failed: %v", err)
			}
		}
	}
}
