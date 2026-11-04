// internal/service/fine_service.go

package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrFineNotFound       = errors.New("fine not found")
	ErrInvalidFineRequest = errors.New("invalid fine request")
)

func validFineStatus(status string) bool {
	switch status {
	case "PENDING", "PAID", "WAIVED":
		return true
	default:
		return false
	}
}

func GetFines(
	db *gorm.DB,
	status string,
	loanID int64,
) ([]model.Fine, error) {
	status = strings.ToUpper(strings.TrimSpace(status))

	if status != "" && !validFineStatus(status) {
		return nil, ErrInvalidFineRequest
	}
	if loanID < 0 {
		return nil, ErrInvalidFineRequest
	}

	return repository.GetFines(db, status, loanID)
}

func UpdateFineStatus(
	db *gorm.DB,
	fineID int64,
	status string,
	staffID int,
	ip string,
	userAgent string,
) (*model.Fine, error) {
	status = strings.ToUpper(strings.TrimSpace(status))

	if fineID <= 0 || staffID <= 0 || !validFineStatus(status) {
		return nil, ErrInvalidFineRequest
	}

	var updatedFine model.Fine

	err := db.Transaction(func(tx *gorm.DB) error {
		fine, err := repository.LockFineByID(tx, fineID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFineNotFound
			}
			return err
		}

		var paidAt *time.Time
		if status == "PAID" {
			now := time.Now().UTC()
			paidAt = &now
		}

		if err := repository.UpdateFineStatus(
			tx,
			fineID,
			status,
			paidAt,
		); err != nil {
			return err
		}

		if err := writeCirculationAudit(
			tx,
			staffID,
			"UPDATE",
			fmt.Sprintf(
				"Updated fine ID %d for loan ID %d from %s to %s",
				fine.FineID,
				fine.LoanID,
				fine.Status,
				status,
			),
			ip,
			userAgent,
		); err != nil {
			return err
		}

		fine.Status = status
		fine.PaidAt = paidAt
		updatedFine = fine

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &updatedFine, nil
}
