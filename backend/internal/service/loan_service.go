// internal/service/loan_service.go

package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrInvalidBorrowRequest = errors.New("invalid borrow request")
	ErrBorrowerNotEligible  = errors.New("borrower is not eligible to borrow")
	ErrCopyUnavailable      = errors.New("book copy is unavailable")
	ErrInvalidLoanPolicy    = errors.New("invalid borrowing policy")
)

type BorrowResult struct {
	Loan    model.Loan `json:"loan"`
	Barcode string     `json:"barcode"`
	Message string     `json:"message"`
}

func getPositivePolicyDays(tx *gorm.DB, key string) (int, error) {
	value, err := repository.GetSystemParameter(tx, key)
	if err != nil {
		return 0, fmt.Errorf("%w: parameter %s: %v", ErrInvalidLoanPolicy, key, err)
	}

	days, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || days <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive integer", ErrInvalidLoanPolicy, key)
	}

	return days, nil
}

func getGuestBorrowingEnabled(tx *gorm.DB) (bool, error) {
	value, err := repository.GetSystemParameter(tx, "GUEST_BORROWING_ENABLED")
	if err != nil {
		return false, fmt.Errorf("%w: guest borrowing setting: %v", ErrInvalidLoanPolicy, err)
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("%w: GUEST_BORROWING_ENABLED must be true or false", ErrInvalidLoanPolicy)
	}

	return enabled, nil
}

func writeCirculationAudit(
	tx *gorm.DB,
	userID int,
	actionCode string,
	description string,
	ip string,
	userAgent string,
) error {
	actionID, err := repository.GetAuditActionID(tx, actionCode)
	if err != nil {
		return fmt.Errorf("get audit action ID: %w", err)
	}

	moduleID, err := repository.GetAuditModuleID(tx, "CIRCULATION")
	if err != nil {
		return fmt.Errorf("get circulation module ID: %w", err)
	}

	return repository.InsertAuditLog(
		tx,
		&userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	)
}

func BorrowBook(
	db *gorm.DB,
	borrowerID int,
	barcode string,
	staffID int,
	ip string,
	userAgent string,
) (*BorrowResult, error) {
	barcode = strings.TrimSpace(barcode)

	if borrowerID <= 0 || staffID <= 0 || barcode == "" {
		return nil, ErrInvalidBorrowRequest
	}

	var result BorrowResult

	err := db.Transaction(func(tx *gorm.DB) error {
		// Validate the borrower account and its current policy.
		account, err := repository.GetBorrowerAccount(tx, borrowerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBorrowerNotEligible
			}
			return err
		}

		if account.StatusCode != "ACTIVE" {
			return ErrBorrowerNotEligible
		}

		if account.ExpirationDate != nil &&
			!account.ExpirationDate.After(time.Now()) {
			return ErrBorrowerNotEligible
		}

		var policyKey string

		switch account.RoleCode {
		case "STUDENT":
			policyKey = "STUDENT_LOAN_DAYS"
		case "FACULTY":
			policyKey = "FACULTY_LOAN_DAYS"
		case "GUEST":
			enabled, err := getGuestBorrowingEnabled(tx)
			if err != nil {
				return err
			}
			if !enabled {
				return ErrBorrowerNotEligible
			}
			return fmt.Errorf("%w: guest loan duration has not been configured", ErrInvalidLoanPolicy)
		default:
			return ErrBorrowerNotEligible
		}

		loanDays, err := getPositivePolicyDays(tx, policyKey)
		if err != nil {
			return err
		}

		// Lock the physical copy before checking its availability.
		copy, err := repository.LockBookCopyByBarcode(tx, barcode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCopyUnavailable
			}
			return err
		}

		if copy.CopyStatus != "AVAILABLE" {
			return ErrCopyUnavailable
		}

		loanDate, err := repository.GetDatabaseCurrentDate(tx)
		if err != nil {
			return err
		}

		loanDate = time.Date(
			loanDate.Year(), loanDate.Month(), loanDate.Day(),
			0, 0, 0, 0, time.UTC,
		)
		dueDate := loanDate.AddDate(0, 0, loanDays)

		loan := model.Loan{
			CopyID:       copy.CopyID,
			UserID:       borrowerID,
			LoanDate:     loanDate,
			DueDate:      dueDate,
			RenewalCount: 0,
			Status:       "BORROWED",
		}

		if err := repository.CreateLoan(tx, &loan); err != nil {
			return err
		}

		if err := repository.MarkCopyBorrowed(tx, copy.CopyID); err != nil {
			return err
		}

		if err := writeCirculationAudit(
			tx,
			staffID,
			"CREATE",
			fmt.Sprintf(
				"Borrowed copy ID %d (barcode %s) for user ID %d; loan ID %d; due date %s",
				copy.CopyID,
				barcode,
				borrowerID,
				loan.LoanID,
				dueDate.Format("2006-01-02"),
			),
			ip,
			userAgent,
		); err != nil {
			return err
		}

		result = BorrowResult{
			Loan:    loan,
			Barcode: barcode,
			Message: "Book borrowed successfully",
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}
