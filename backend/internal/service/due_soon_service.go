// internal/service/due_soon_service.go

package service

import (
	"encoding/json"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type dueSoonLoan struct {
	LoanID     int64  `gorm:"column:loan_id"`
	UserID     int    `gorm:"column:user_id"`
	Email      string `gorm:"column:email_address"`
	FirstName  string `gorm:"column:first_name"`
	MiddleName string `gorm:"column:middle_name"`
	LastName   string `gorm:"column:last_name"`
	Suffix     string `gorm:"column:suffix_extension"`
	DueDate    string `gorm:"column:due_date"`
}

func (s *NotificationService) ProcessDueSoonLoans() error {
	var loans []dueSoonLoan

	err := s.db.Raw(`
		SELECT
			l.loan_id,
			l.user_id,
			u.email_address,
			up.first_name,
			up.middle_name,
			up.last_name,
			up.suffix_extension,
			TO_CHAR(l.due_date::date, 'YYYY-MM-DD') AS due_date
		FROM loan AS l
		JOIN "user" AS u
			ON u.user_id = l.user_id
		LEFT JOIN user_profile AS up
			ON up.user_id = u.user_id
		WHERE l.status = 'BORROWED'
		  AND l.due_date::date = CURRENT_DATE
	`).Scan(&loans).Error

	if err != nil {
		return fmt.Errorf("find due soon loans: %w", err)
	}

	for _, loan := range loans {
		recipientName := strings.TrimSpace(strings.Join([]string{
			loan.FirstName,
			loan.MiddleName,
			loan.LastName,
			loan.Suffix,
		}, " "))

		if recipientName == "" {
			recipientName = "Library Borrower"
		}

		if strings.TrimSpace(loan.Email) == "" {
			return fmt.Errorf(
				"user %d has no email address for due soon loan %d",
				loan.UserID,
				loan.LoanID,
			)
		}

		err := s.db.Transaction(func(tx *gorm.DB) error {
			// Recheck the loan inside the transaction. It may have been
			// returned, renewed, or otherwise updated since the initial query.
			var stillEligible int64

			err := tx.Raw(`
				SELECT COUNT(*)
				FROM loan
				WHERE loan_id = ?
				  AND user_id = ?
				  AND status = 'BORROWED'
				  AND due_date::date = CURRENT_DATE
			`, loan.LoanID, loan.UserID).Scan(&stillEligible).Error

			if err != nil {
				return fmt.Errorf("recheck due soon loan: %w", err)
			}

			if stillEligible == 0 {
				return nil
			}

			referenceType := "LOAN"
			referenceID := loan.LoanID

			notification := model.Notification{
				UserID:        loan.UserID,
				Type:          "DUE_SOON",
				Channel:       repository.NotificationChannelInApp,
				Subject:       "Upcoming Book Due Date",
				Message:       fmt.Sprintf("One of your borrowed books is due on %s. Please return or renew it on or before the due date.", loan.DueDate),
				IsRead:        false,
				ReferenceType: &referenceType,
				ReferenceID:   &referenceID,
			}

			// The unique index prevents duplicate in-app notices and
			// therefore prevents duplicate email enqueueing for this loan.
			result := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "type"},
					{Name: "channel"},
					{Name: "reference_type"},
					{Name: "reference_id"},
				},
				TargetWhere: clause.Where{
					Exprs: []clause.Expression{
						clause.Eq{
							Column: clause.Column{Name: "type"},
							Value:  "DUE_SOON",
						},
						clause.Eq{
							Column: clause.Column{Name: "channel"},
							Value:  "IN_APP",
						},
						clause.Eq{
							Column: clause.Column{Name: "reference_type"},
							Value:  "LOAN",
						},
					},
				},
				DoNothing: true,
			}).Create(&notification)

			if result.Error != nil {
				return fmt.Errorf("create due soon notification: %w", result.Error)
			}

			// Existing notification means this loan was already processed.
			if result.RowsAffected == 0 {
				return nil
			}

			details, err := json.Marshal([]model.EmailOutboxDetail{
				{
					Label: "Loan ID",
					Value: fmt.Sprintf("%d", loan.LoanID),
				},
				{
					Label: "Due Date",
					Value: loan.DueDate,
				},
			})

			if err != nil {
				return fmt.Errorf("marshal due soon email details: %w", err)
			}

			email := model.EmailOutbox{
				RecipientEmail: loan.Email,
				RecipientName:  recipientName,
				Subject:        "CARDex Upcoming Book Due Date",
				Heading:        "Your Book Is Due Soon",
				Message:        "This is a courtesy reminder from CARDex that one of your borrowed library items is approaching its due date. Please return or renew your item on or before the indicated date to avoid overdue penalties and ensure uninterrupted library privileges.",
				Details:        details,
				Status:         "PENDING",
			}

			if err := repository.EnqueueEmail(tx, &email); err != nil {
				return fmt.Errorf("enqueue due soon email: %w", err)
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("process due soon loan %d: %w", loan.LoanID, err)
		}
	}

	return nil
}
