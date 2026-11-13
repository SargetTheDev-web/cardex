package service

import (
	"context"
	"errors"
	"log"
	"time"
	_ "time/tzdata"

	"backend/internal/repository"

	"gorm.io/gorm"
)

func (s *NotificationService) RunEmailOutboxWorker(ctx context.Context) {
	const (
		pollInterval         = 10 * time.Second
		recoveryInterval     = 5 * time.Minute
		staleAfter           = 15 * time.Minute
		notificationInterval = time.Hour
	)

	// Use Philippine local time for notification scheduling.
	location, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		log.Printf("failed to load Asia/Manila timezone: %v", err)
		return
	}

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	recoveryTicker := time.NewTicker(recoveryInterval)
	defer recoveryTicker.Stop()

	hourlyTicker := time.NewTicker(notificationInterval)
	defer hourlyTicker.Stop()

	// Recover emails left in processing state.
	if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
		log.Printf("email outbox recovery failed: %v", err)
	}

	// Tracks the last attempted hourly notification run.
	lastNotificationAttemptHour := ""

	runHourlyLoanNotifications := func() {
		now := time.Now().In(location)
		currentHour := now.Format("2006-01-02-15")

		// Prevent multiple executions within the same Philippine hour.
		if lastNotificationAttemptHour == currentHour {
			return
		}

		// Record the attempt before processing to avoid repeatedly
		// executing failed jobs during the same hour.
		lastNotificationAttemptHour = currentHour

		log.Printf(
			"starting hourly loan notification processing for %s",
			now.Format("2006-01-02 15:04:05 MST"),
		)

		success := true

		// Process upcoming due dates.
		if err := s.ProcessDueSoonLoans(); err != nil {
			log.Printf("due soon notification job failed: %v", err)
			success = false
		}

		// Process overdue loans.
		if err := s.ProcessOverdueLoans(); err != nil {
			log.Printf("overdue notification job failed: %v", err)
			success = false
		}

		if success {
			log.Printf(
				"hourly loan notification processing completed for %s",
				currentHour,
			)
		} else {
			log.Printf(
				"hourly loan notification processing incomplete for %s; will retry next hour",
				currentHour,
			)
		}
	}

	for {
		if ctx.Err() != nil {
			return
		}

		// Check whether hourly notification processing is due.
		// Executes immediately when the worker starts.
		runHourlyLoanNotifications()

		// Drain pending emails until the queue is empty or an error occurs.
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
			// Check and process pending emails.

		case <-hourlyTicker.C:
			// Return to the top and execute hourly notification processing.

		case <-recoveryTicker.C:
			if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
				log.Printf("email outbox recovery failed: %v", err)
			}
		}
	}
}
