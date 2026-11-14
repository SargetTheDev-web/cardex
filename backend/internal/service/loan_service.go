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
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidBorrowRequest  = errors.New("invalid borrow request")
	ErrBorrowerNotEligible   = errors.New("borrower is not eligible to borrow")
	ErrCopyUnavailable       = errors.New("book copy is unavailable")
	ErrInvalidLoanPolicy     = errors.New("invalid borrowing policy")
	ErrInvalidReturnRequest  = errors.New("invalid return request")
	ErrLoanNotFound          = errors.New("active loan not found")
	ErrInvalidFinePolicy     = errors.New("invalid fine policy")
	ErrBookIDMismatch        = errors.New("book ID does not match barcode")
	ErrDueDateMismatch       = errors.New("due date does not match borrowing policy")
	ErrInvalidLoanRequest    = errors.New("invalid loan request")
	ErrRenewalNotAllowed     = errors.New("loan is not eligible for renewal")
	ErrMaxRenewalsReached    = errors.New("maximum renewal count reached")
	ErrUnpaidFineExists      = errors.New("unpaid fine prevents renewal")
	ErrPendingReservation    = errors.New("another borrower has a pending reservation for this book")
	ErrBorrowingLimitReached = errors.New(
		"maximum active borrowing limit reached",
	)
)

type BorrowResult struct {
	Loan    model.Loan `json:"loan"`
	BookID  int        `json:"book_id"`
	Barcode string     `json:"barcode"`
	Message string     `json:"message"`
}

func getPositivePolicyDays(tx *gorm.DB, key string) (int, error) {
	value, err := repository.GetSystemParameter(tx, key)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: parameter %s: %v",
			ErrInvalidLoanPolicy, key, err,
		)
	}

	days, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || days <= 0 {
		return 0, fmt.Errorf(
			"%w: %s must be a positive integer",
			ErrInvalidLoanPolicy, key,
		)
	}

	return days, nil
}

func getGuestBorrowingEnabled(tx *gorm.DB) (bool, error) {
	value, err := repository.GetSystemParameter(tx, "GUEST_BORROWING_ENABLED")
	if err != nil {
		return false, fmt.Errorf(
			"%w: guest borrowing setting: %v",
			ErrInvalidLoanPolicy, err,
		)
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf(
			"%w: GUEST_BORROWING_ENABLED must be true or false",
			ErrInvalidLoanPolicy,
		)
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
	bookID int,
	requestedDueDate string,
	borrowNotes string,
	staffID int,
	ip string,
	userAgent string,
) (*BorrowResult, error) {
	barcode = strings.TrimSpace(barcode)
	requestedDueDate = strings.TrimSpace(requestedDueDate)
	borrowNotes = strings.TrimSpace(borrowNotes)

	if borrowerID <= 0 ||
		staffID <= 0 ||
		bookID <= 0 ||
		barcode == "" ||
		requestedDueDate == "" {
		return nil, ErrInvalidBorrowRequest
	}

	// Validate the frontend's date format: YYYY-MM-DD.
	parsedDueDate, err := time.Parse("2006-01-02", requestedDueDate)
	if err != nil {
		return nil, ErrInvalidBorrowRequest
	}

	var result BorrowResult

	err = db.Transaction(func(tx *gorm.DB) error {
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

		// Preserve role-based borrowing restrictions.
		switch account.RoleCode {
		case "STUDENT", "FACULTY":
			// Allowed. Due date is supplied by the frontend.
		case "GUEST":
			enabled, err := getGuestBorrowingEnabled(tx)
			if err != nil {
				return err
			}
			if !enabled {
				return ErrBorrowerNotEligible
			}
			return fmt.Errorf(
				"%w: guest borrowing is enabled but a guest loan policy is not implemented",
				ErrInvalidLoanPolicy,
			)
		default:
			return ErrBorrowerNotEligible
		}

		if err := enforceBorrowingLimit(
			tx,
			borrowerID,
			account.RoleCode,
		); err != nil {
			return err
		}

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

		if copy.BookID != bookID {
			return ErrBookIDMismatch
		}

		loanDate, err := repository.GetDatabaseCurrentDate(tx)
		if err != nil {
			return err
		}

		loanDate = time.Date(
			loanDate.Year(),
			loanDate.Month(),
			loanDate.Day(),
			0, 0, 0, 0,
			time.UTC,
		)

		dueDate := time.Date(
			parsedDueDate.Year(),
			parsedDueDate.Month(),
			parsedDueDate.Day(),
			0, 0, 0, 0,
			time.UTC,
		)

		if dueDate.Before(loanDate) {
			return ErrInvalidBorrowRequest
		}

		loan := model.Loan{
			CopyID:       copy.CopyID,
			UserID:       borrowerID,
			LoanDate:     loanDate,
			DueDate:      dueDate,
			RenewalCount: 0,
			Status:       "BORROWED",
		}

		if borrowNotes != "" {
			loan.BorrowNotes = &borrowNotes
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
			BookID:  copy.BookID,
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
	Barcode    string    `gorm:"column:barcode"`
	DueDate    time.Time `gorm:"column:due_date"`
	LoanStatus string    `gorm:"column:loan_status"`
}

func ReturnBook(
	db *gorm.DB,
	loanID int64,
	condition string,
	returnNotes string,
	fineAmount float64,
	staffID int,
	ip string,
	userAgent string,
) (*ReturnResult, error) {
	condition = strings.ToUpper(strings.TrimSpace(condition))
	returnNotes = strings.TrimSpace(returnNotes)

	if loanID <= 0 ||
		staffID <= 0 ||
		(condition != "GOOD" && condition != "DAMAGED") ||
		fineAmount < 0 {
		return nil, ErrInvalidReturnRequest
	}

	var result ReturnResult

	err := db.Transaction(func(tx *gorm.DB) error {
		var loan returnLoanRecord

		err := tx.Raw(`
			SELECT
				l.loan_id,
				l.copy_id,
				l.user_id,
				l.due_date,
				l.status AS loan_status,
				COALESCE(c.barcode, '') AS barcode,
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
			JOIN book_copy c ON c.copy_id = l.copy_id
			JOIN "user" u ON u.user_id = l.user_id
			LEFT JOIN user_profile up ON up.user_id = u.user_id
			WHERE l.loan_id = ?
			  AND l.status IN ('BORROWED', 'OVERDUE')
			  AND l.return_date IS NULL
			FOR UPDATE OF l, c
		`, loanID).Scan(&loan).Error

		if err != nil {
			return err
		}

		if loan.LoanID == 0 {
			return ErrLoanNotFound
		}

		var overdueDays int
		if err := tx.Raw(`
			SELECT GREATEST(CURRENT_DATE - ?::date, 0)
		`, loan.DueDate.Format("2006-01-02")).Scan(&overdueDays).Error; err != nil {
			return err
		}

		copyStatus := "AVAILABLE"
		if condition == "DAMAGED" {
			copyStatus = "DAMAGED"
		}

		if returnNotes == "" {
			if err := tx.Exec(`
				UPDATE loan
				SET status = 'RETURNED',
				    return_date = CURRENT_DATE,
				    return_notes = NULL
				WHERE loan_id = ?
				  AND return_date IS NULL
			`, loan.LoanID).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Exec(`
				UPDATE loan
				SET status = 'RETURNED',
				    return_date = CURRENT_DATE,
				    return_notes = ?
				WHERE loan_id = ?
				  AND return_date IS NULL
			`, returnNotes, loan.LoanID).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec(`
			UPDATE book_copy
			SET copy_status = ?
			WHERE copy_id = ?
		`, copyStatus, loan.CopyID).Error; err != nil {
			return err
		}

		// Use the amount and reason supplied to ReturnBook.
		// A zero amount means no fine record is created.
		if fineAmount > 0 {
			fineReason := returnNotes
			if fineReason == "" {
				fineReason = "Fine assessed during book return"
			}

			if err := tx.Exec(`
        INSERT INTO fine (loan_id, amount, reason, status)
        VALUES (?, ?::numeric, ?, 'PENDING')
    `, loan.LoanID, fineAmount, fineReason).Error; err != nil {
				return fmt.Errorf("create fine: %w", err)
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
				loan.Barcode,
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
			Barcode:     loan.Barcode,
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

type RenewLoanResult struct {
	LoanID       int64     `json:"loan_id"`
	PreviousDue  time.Time `json:"previous_due_date"`
	NewDueDate   time.Time `json:"new_due_date"`
	RenewalCount int       `json:"renewal_count"`
	Message      string    `json:"message"`
}

func RenewLoan(
	db *gorm.DB,
	loanID int64,
	staffID int,
	ipAddress string,
	userAgent string,
) (*RenewLoanResult, error) {
	if staffID <= 0 || loanID <= 0 {
		return nil, ErrInvalidLoanRequest
	}

	return renewLoan(
		db,
		loanID,
		staffID,
		nil,
		ipAddress,
		userAgent,
	)
}

func GetMyLoans(db *gorm.DB, userID int) ([]repository.MyLoan, error) {
	if userID <= 0 {
		return nil, ErrInvalidLoanRequest
	}

	return repository.GetMyLoans(db, userID)
}

func RenewMyLoan(
	db *gorm.DB,
	loanID int64,
	userID int,
	ipAddress string,
	userAgent string,
) (*RenewLoanResult, error) {
	if userID <= 0 || loanID <= 0 {
		return nil, ErrInvalidLoanRequest
	}

	return renewLoan(
		db,
		loanID,
		userID,
		&userID,
		ipAddress,
		userAgent,
	)
}

func renewLoan(
	db *gorm.DB,
	loanID int64,
	actorID int,
	ownerID *int,
	ipAddress string,
	userAgent string,
) (*RenewLoanResult, error) {
	if db == nil || loanID <= 0 || actorID <= 0 {
		return nil, ErrInvalidLoanRequest
	}

	var result RenewLoanResult

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Load renewal policy.
		maxRenewals, err := getPositivePolicyDays(
			tx,
			"LOAN_MAX_RENEWALS",
		)
		if err != nil {
			return err
		}

		renewalDays, err := getPositivePolicyDays(
			tx,
			"LOAN_RENEWAL_DAYS",
		)
		if err != nil {
			return err
		}

		// 2. Lock the loan record.
		// Borrowers can only access their own loans.
		// Staff can access any borrower's loan.
		query := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("loan_id = ?", loanID)

		if ownerID != nil {
			query = query.Where("user_id = ?", *ownerID)
		}

		var loan model.Loan

		err = query.First(&loan).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLoanNotFound
		}

		if err != nil {
			return err
		}

		// 3. Validate loan status.
		if loan.Status != "BORROWED" || loan.ReturnDate != nil {
			return ErrRenewalNotAllowed
		}

		// 4. Enforce maximum renewal count.
		if loan.RenewalCount >= maxRenewals {
			return ErrMaxRenewalsReached
		}

		// 5. Get the database's current date.
		currentDate, err := repository.GetDatabaseCurrentDate(tx)
		if err != nil {
			return err
		}

		// Normalize dates to prevent timezone-related comparisons.
		currentDate = time.Date(
			currentDate.Year(),
			currentDate.Month(),
			currentDate.Day(),
			0, 0, 0, 0,
			time.UTC,
		)

		dueDate := time.Date(
			loan.DueDate.Year(),
			loan.DueDate.Month(),
			loan.DueDate.Day(),
			0, 0, 0, 0,
			time.UTC,
		)

		// A loan due today is eligible.
		// An overdue loan cannot be renewed.
		if dueDate.Before(currentDate) {
			return ErrRenewalNotAllowed
		}

		// 6. Check for unpaid fines.
		var unpaidFineCount int64

		err = tx.
			Table("fine").
			Where(
				"loan_id = ? AND status = ?",
				loanID,
				"PENDING",
			).
			Count(&unpaidFineCount).Error

		if err != nil {
			return err
		}

		if unpaidFineCount > 0 {
			return ErrUnpaidFineExists
		}

		// 7. Resolve the book associated with the loan.
		var bookID int

		err = tx.
			Table("book_copy").
			Select("book_id").
			Where("copy_id = ?", loan.CopyID).
			Scan(&bookID).Error

		if err != nil {
			return err
		}

		if bookID <= 0 {
			return ErrLoanNotFound
		}

		// 8. Check for another borrower's pending reservation.
		// Exclude the actual borrower, not the staff member
		// performing the renewal.
		var pendingReservationCount int64

		err = tx.
			Table("reservation").
			Where(
				"book_id = ? AND user_id <> ? AND status = ?",
				bookID,
				loan.UserID,
				"PENDING",
			).
			Count(&pendingReservationCount).Error

		if err != nil {
			return err
		}

		if pendingReservationCount > 0 {
			return ErrPendingReservation
		}

		// 9. Calculate the new due date.
		previousDueDate := dueDate

		newDueDate := previousDueDate.AddDate(
			0,
			0,
			renewalDays,
		)

		newRenewalCount := loan.RenewalCount + 1

		// 10. Update the loan.
		// The row lock protects this operation from concurrent
		// renewal requests for the same loan.
		update := tx.
			Model(&model.Loan{}).
			Where(`
				loan_id = ?
				AND status = ?
				AND return_date IS NULL
				AND renewal_count = ?
			`,
				loanID,
				"BORROWED",
				loan.RenewalCount,
			).
			Updates(map[string]interface{}{
				"due_date":      newDueDate,
				"renewal_count": newRenewalCount,
			})

		if update.Error != nil {
			return update.Error
		}

		if update.RowsAffected != 1 {
			return ErrRenewalNotAllowed
		}

		// 11. Determine the actor for audit logging.
		actorType := "STAFF"

		if ownerID != nil {
			actorType = "BORROWER"
		}

		// 12. Record the renewal.
		if err := writeCirculationAudit(
			tx,
			actorID,
			"UPDATE",
			fmt.Sprintf(
				"%s user ID %d renewed loan ID %d for borrower user ID %d. Previous due date: %s; new due date: %s; renewal count: %d",
				actorType,
				actorID,
				loanID,
				loan.UserID,
				previousDueDate.Format("2006-01-02"),
				newDueDate.Format("2006-01-02"),
				newRenewalCount,
			),
			ipAddress,
			userAgent,
		); err != nil {
			return err
		}

		// 13. Prepare response.
		result = RenewLoanResult{
			LoanID:       loanID,
			PreviousDue:  previousDueDate,
			NewDueDate:   newDueDate,
			RenewalCount: newRenewalCount,
			Message:      "Loan successfully renewed.",
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func enforceBorrowingLimit(
	tx *gorm.DB,
	borrowerID int,
	roleCode string,
) error {
	if borrowerID <= 0 {
		return ErrInvalidBorrowRequest
	}

	var parameterKey string

	switch roleCode {
	case "STUDENT":
		parameterKey = "MAX_BOOKS_PER_STUDENT"

	case "FACULTY":
		parameterKey = "MAX_BOOKS_PER_FACULTY"

	default:
		return ErrBorrowerNotEligible
	}

	// Lock the borrower's account row.
	// This serializes concurrent borrowing operations
	// for the same borrower.
	var lockedUserID int

	if err := tx.Raw(`
		SELECT user_id
		FROM "user"
		WHERE user_id = ?
		FOR UPDATE
	`, borrowerID).Scan(&lockedUserID).Error; err != nil {
		return err
	}

	if lockedUserID == 0 {
		return ErrBorrowerNotEligible
	}

	// Retrieve the role-specific borrowing limit.
	maxBooks, err := getPositivePolicyDays(tx, parameterKey)
	if err != nil {
		return err
	}

	// Count all currently active loans.
	var activeLoanCount int64

	if err := tx.
		Table("loan").
		Where("user_id = ?", borrowerID).
		Where("status IN ?", []string{
			"BORROWED",
			"OVERDUE",
		}).
		Count(&activeLoanCount).Error; err != nil {
		return err
	}

	if activeLoanCount >= int64(maxBooks) {
		return ErrBorrowingLimitReached
	}

	return nil
}
