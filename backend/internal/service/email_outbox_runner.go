// internal/service/email_outbox_runner.go

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
		pollInterval     = 10 * time.Second
		recoveryInterval = 5 * time.Minute
		staleAfter       = 15 * time.Minute
		dailyCheck       = time.Minute
		notificationHour = 8
	)

	// Use Philippine local time for daily loan notification processing.
	location, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		log.Printf("failed to load Asia/Manila timezone: %v", err)
		return
	}

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()

	recoveryTicker := time.NewTicker(recoveryInterval)
	defer recoveryTicker.Stop()

	dailyTicker := time.NewTicker(dailyCheck)
	defer dailyTicker.Stop()

	if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
		log.Printf("email outbox recovery failed: %v", err)
	}

	// Tracks the last successfully completed daily notification run.
	// If the process restarts, the database's unique indexes prevent
	// duplicate notifications from being created.
	lastNotificationRunDate := ""

	runDailyLoanNotifications := func() {
		now := time.Now().In(location)

		// Do not run before 8:00 AM Philippine time.
		if now.Hour() < notificationHour {
			return
		}

		today := now.Format("2006-01-02")

		if lastNotificationRunDate == today {
			return
		}

		log.Printf(
			"starting daily loan notification processing for %s",
			today,
		)

		success := true

		// Process upcoming due dates first.
		if err := s.ProcessDueSoonLoans(); err != nil {
			log.Printf("due soon notification job failed: %v", err)
			success = false
		}

		// Process loans that are already overdue.
		if err := s.ProcessOverdueLoans(); err != nil {
			log.Printf("overdue notification job failed: %v", err)
			success = false
		}

		if success {
			lastNotificationRunDate = today
			log.Printf(
				"daily loan notification processing completed for %s",
				today,
			)
		} else {
			log.Printf(
				"daily loan notification processing incomplete for %s; will retry",
				today,
			)
		}
	}

	for {
		if ctx.Err() != nil {
			return
		}

		// Run the daily job when due. This also handles a service
		// that starts after 8:00 AM Philippine time.
		runDailyLoanNotifications()

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
			// Return to the top and process pending emails.

		case <-dailyTicker.C:
			// Return to the top and check the daily schedule.

		case <-recoveryTicker.C:
			if err := repository.RecoverStaleEmailClaims(s.db, staleAfter); err != nil {
				log.Printf("email outbox recovery failed: %v", err)
			}
		}
	}
}
