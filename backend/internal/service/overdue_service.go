// internal/service/overdue_service.go

package service

import (
	"encoding/json"
	"fmt"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type overdueLoan struct {
	LoanID int64  `gorm:"column:loan_id"`
	UserID int    `gorm:"column:user_id"`
	Email  string `gorm:"column:email_address"`
	Name   string `gorm:"column:username"`
}

func (s *NotificationService) ProcessOverdueLoans() error {
	var loans []overdueLoan

	err := s.db.Raw(`
		SELECT
			l.loan_id,
			l.user_id,
			u.email_address,
			u.username
		FROM loan AS l
		JOIN "user" AS u ON u.user_id = l.user_id
		WHERE l.status = 'BORROWED'
		  AND l.due_date < CURRENT_DATE
		  AND NOT EXISTS (
			  SELECT 1
			  FROM notification AS n
			  WHERE n.user_id = l.user_id
			    AND n.type = 'OVERDUE'
			    AND n.channel = 'IN_APP'
			    AND n.reference_type = 'LOAN'
			    AND n.reference_id = l.loan_id
		  )
	`).Scan(&loans).Error
	if err != nil {
		return fmt.Errorf("find overdue loans: %w", err)
	}

	for _, loan := range loans {
		if loan.Email == "" {
			return fmt.Errorf("user %d has no email address", loan.UserID)
		}

		err := s.db.Transaction(func(tx *gorm.DB) error {
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

			// Insert the in-app notification. The unique index prevents
			// duplicate overdue notices for the same loan.
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
				// This overdue notification already exists; don't queue another email.
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
				RecipientName:  loan.Name,
				Subject:        "CARDex Overdue Book Notice",
				Heading:        "Overdue Book Notice",
				Message:        "One of your borrowed books is overdue. Please return it as soon as possible.",
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
				"create overdue notices for loan %d: %w",
				loan.LoanID,
				err,
			)
		}
	}

	return nil
}
