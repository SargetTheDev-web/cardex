// internal/service/loan_service.go

package service

import (
	"errors"
	"fmt"
	"math/big"
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
	ErrInvalidReturnRequest = errors.New("invalid return request")
	ErrLoanNotFound         = errors.New("active loan not found")
	ErrInvalidFinePolicy    = errors.New("invalid fine policy")
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

type ReturnResult struct {
	LoanID      int64     `json:"loan_id"`
	CopyID      int       `json:"copy_id"`
	UserID      int       `json:"user_id"`
	Borrower    string    `json:"borrower"`
	Barcode     string    `json:"barcode"`
	ReturnDate  time.Time `json:"return_date"`
	OverdueDays int       `json:"overdue_days"`
	FineAmount  float64   `json:"fine_amount"`
	CopyStatus  string    `json:"copy_status"`
	Message     string    `json:"message"`
}

type returnLoanRecord struct {
	LoanID     int64     `gorm:"column:loan_id"`
	CopyID     int       `gorm:"column:copy_id"`
	UserID     int       `gorm:"column:user_id"`
	Borrower   string    `gorm:"column:borrower"`
	DueDate    time.Time `gorm:"column:due_date"`
	LoanStatus string    `gorm:"column:loan_status"`
}

func ReturnBook(
	db *gorm.DB,
	barcode string,
	condition string,
	staffID int,
	ip string,
	userAgent string,
) (*ReturnResult, error) {
	barcode = strings.TrimSpace(barcode)
	condition = strings.ToUpper(strings.TrimSpace(condition))

	if barcode == "" || staffID <= 0 ||
		(condition != "GOOD" && condition != "DAMAGED") {
		return nil, ErrInvalidReturnRequest
	}

	var result ReturnResult

	err := db.Transaction(func(tx *gorm.DB) error {
		// Lock both the active loan and its physical copy.
		var loan returnLoanRecord

		err := tx.Raw(`
    SELECT
        l.loan_id,
        l.copy_id,
        l.user_id,
        l.due_date,
        l.status AS loan_status,

        COALESCE(
            NULLIF(
                BTRIM(
                    CONCAT_WS(
                        ' ',
                        NULLIF(BTRIM(up.first_name), ''),
                        NULLIF(BTRIM(up.middle_name), ''),
                        NULLIF(BTRIM(up.last_name), ''),
                        NULLIF(BTRIM(up.suffix_extension), '')
                    )
                ),
                ''
            ),
            NULLIF(u.username, ''),
            NULLIF(u.email_address, ''),
            CONCAT('User #', l.user_id)
        ) AS borrower

    FROM loan l
    JOIN book_copy c
        ON c.copy_id = l.copy_id
    JOIN "user" u
        ON u.user_id = l.user_id
    LEFT JOIN user_profile up
        ON up.user_id = u.user_id

    WHERE c.barcode = ?
      AND l.status IN ('BORROWED', 'OVERDUE')
      AND l.return_date IS NULL

    FOR UPDATE OF l, c
`, barcode).Scan(&loan).Error
		if err != nil {
			return err
		}

		// Scan does not reliably return ErrRecordNotFound for a zero-row
		// result, so explicitly check the primary key.
		if loan.LoanID == 0 {
			return ErrLoanNotFound
		}

		overdueDays := 0
		if err := tx.Raw(`
			SELECT GREATEST(CURRENT_DATE - ?::date, 0)
		`, loan.DueDate.Format("2006-01-02")).Scan(&overdueDays).Error; err != nil {
			return err
		}

		copyStatus := "AVAILABLE"
		if condition == "DAMAGED" {
			copyStatus = "DAMAGED"
		}

		// Use the database date, not the Go server's local clock.
		if err := tx.Exec(`
			UPDATE loan
			SET status = 'RETURNED',
			    return_date = CURRENT_DATE
			WHERE loan_id = ?
			  AND return_date IS NULL
		`, loan.LoanID).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			UPDATE book_copy
			SET copy_status = ?
			WHERE copy_id = ?
		`, copyStatus, loan.CopyID).Error; err != nil {
			return err
		}

		var fineAmount float64

		if overdueDays > 0 {
			rate, err := repository.GetSystemParameter(
				tx,
				"FINE_RATE_PER_DAY_PHP",
			)
			if err != nil {
				return fmt.Errorf(
					"%w: FINE_RATE_PER_DAY_PHP is unavailable",
					ErrInvalidFinePolicy,
				)
			}

			rate = strings.TrimSpace(rate)

			// Validate the parameter before passing it to PostgreSQL as numeric.
			rateValue, parseErr := strconv.ParseFloat(rate, 64)
			if parseErr != nil || rateValue < 0 {
				return ErrInvalidFinePolicy
			}

			if _, ok := new(big.Rat).SetString(rate); !ok {
				return ErrInvalidFinePolicy
			}

			if rateValue > 0 {
				if err := tx.Raw(`
					SELECT (?::numeric * ?::numeric)::double precision
				`, overdueDays, rate).Scan(&fineAmount).Error; err != nil {
					return err
				}

				reason := fmt.Sprintf(
					"Overdue return: %d day(s) at PHP %s per day",
					overdueDays,
					rate,
				)

				// fine_id is generated by the PostgreSQL identity column.
				if err := tx.Exec(`
					INSERT INTO fine (loan_id, amount, reason, status)
					VALUES (
						?,
						(?::numeric * ?::numeric),
						?,
						'PENDING'
					)
				`, loan.LoanID, overdueDays, rate, reason).Error; err != nil {
					return err
				}
			}
		}

		var returnDate time.Time
		if err := tx.Raw(`SELECT CURRENT_DATE`).Scan(&returnDate).Error; err != nil {
			return err
		}

		if err := writeCirculationAudit(
			tx,
			staffID,
			"UPDATE",
			fmt.Sprintf(
				"Returned copy ID %d (barcode %s) for user ID %d; loan ID %d; condition %s; overdue days %d; fine PHP %.2f",
				loan.CopyID,
				barcode,
				loan.UserID,
				loan.LoanID,
				condition,
				overdueDays,
				fineAmount,
			),
			ip,
			userAgent,
		); err != nil {
			return err
		}

		result = ReturnResult{
			LoanID:      loan.LoanID,
			CopyID:      loan.CopyID,
			UserID:      loan.UserID,
			Borrower:    loan.Borrower,
			Barcode:     barcode,
			ReturnDate:  returnDate,
			OverdueDays: overdueDays,
			FineAmount:  fineAmount,
			CopyStatus:  copyStatus,
			Message:     "Book returned successfully",
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func GetRecentTransactions(
	db *gorm.DB,
) ([]repository.RecentTransaction, error) {
	return repository.GetRecentTransactions(db)

}
