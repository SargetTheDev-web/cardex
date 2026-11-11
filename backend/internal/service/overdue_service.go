// internal/service/overdue_service.go

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

type overdueLoan struct {
	LoanID     int64  `gorm:"column:loan_id"`
	UserID     int    `gorm:"column:user_id"`
	Email      string `gorm:"column:email_address"`
	FirstName  string `gorm:"column:first_name"`
	MiddleName string `gorm:"column:middle_name"`
	LastName   string `gorm:"column:last_name"`
	Suffix     string `gorm:"column:suffix_extension"`
}

func (s *NotificationService) ProcessOverdueLoans() error {
	var loans []overdueLoan

	err := s.db.Raw(`
    SELECT
        l.loan_id,
        l.user_id,
        u.email_address,
        up.first_name,
        up.middle_name,
        up.last_name,
        up.suffix_extension
    FROM loan AS l
    JOIN "user" AS u
        ON u.user_id = l.user_id
    LEFT JOIN user_profile AS up
        ON up.user_id = u.user_id
    WHERE l.status = 'BORROWED'
      AND l.due_date < CURRENT_DATE
`).Scan(&loans).Error
	if err != nil {
		return fmt.Errorf("find overdue loans: %w", err)
	}

	for _, loan := range loans {

		recipientName := strings.TrimSpace(
			strings.Join(
				[]string{
					loan.FirstName,
					loan.MiddleName,
					loan.LastName,
					loan.Suffix,
				},
				" ",
			),
		)

		if recipientName == "" {
			recipientName = "Library Borrower"
		}
		if loan.Email == "" {
			return fmt.Errorf("user %d has no email address", loan.UserID)
		}

		err := s.db.Transaction(func(tx *gorm.DB) error {
			// Update only if the loan is still BORROWED and overdue.
			result := tx.Exec(`
				UPDATE loan
				SET status = ?
				WHERE loan_id = ?
				  AND status = ?
				  AND due_date < CURRENT_DATE
			`, "OVERDUE", loan.LoanID, "BORROWED")

			if result.Error != nil {
				return fmt.Errorf("update loan status to overdue: %w", result.Error)
			}

			// It may have been returned or updated after the initial query.
			if result.RowsAffected != 1 {
				return nil
			}

			referenceType := "LOAN"
			referenceID := loan.LoanID

			notification := model.Notification{
				UserID:        loan.UserID,
				Type:          "OVERDUE",
				Channel:       repository.NotificationChannelInApp,
				Subject:       "Overdue Book Notice",
				Message:       "One of your borrowed books is overdue. Please return it as soon as possible.",
				IsRead:        false,
				ReferenceType: &referenceType,
				ReferenceID:   &referenceID,
			}

			// The unique index prevents duplicate in-app notices for this loan.
			result = tx.Clauses(clause.OnConflict{
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
							Value:  "OVERDUE",
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
				return result.Error
			}

			if result.RowsAffected == 0 {
				// The notification already exists; don't queue another email.
				return nil
			}

			details, err := json.Marshal([]model.EmailOutboxDetail{
				{
					Label: "Loan ID",
					Value: fmt.Sprintf("%d", loan.LoanID),
				},
			})
			if err != nil {
				return fmt.Errorf("marshal overdue email details: %w", err)
			}

			email := model.EmailOutbox{
				RecipientEmail: loan.Email,
				RecipientName:  recipientName,
				Subject:        "CARDex Overdue Book Notice",
				Heading:        "Overdue Book Notice",
				Message:        "This is a formal reminder from CARDex regarding your borrowed item. Please return or renew your book on or before the due date to ensure uninterrupted library privileges. You may renew your item online through the student portal or at the library circulation desk.",
				Details:        details,
				Status:         "PENDING",
			}

			if err := repository.EnqueueEmail(tx, &email); err != nil {
				return fmt.Errorf("enqueue overdue email: %w", err)
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf(
				"process overdue loan %d: %w",
				loan.LoanID,
				err,
			)
		}
	}

	return nil
}
