// /internal/models/loan.go

package model

import "time"

type Loan struct {
	LoanID       int64      `json:"loan_id" gorm:"column:loan_id;primaryKey"`
	CopyID       int        `json:"copy_id" gorm:"column:copy_id;not null"`
	UserID       int        `json:"user_id" gorm:"column:user_id;not null"`
	LoanDate     time.Time  `json:"loan_date" gorm:"column:loan_date;type:date;not null"`
	DueDate      time.Time  `json:"due_date" gorm:"column:due_date;type:date;not null"`
	ReturnDate   *time.Time `json:"return_date,omitempty" gorm:"column:return_date;type:date"`
	RenewalCount int        `json:"renewal_count" gorm:"column:renewal_count;not null"`
	Status       string     `json:"status" gorm:"column:status;not null"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at"`
}

func (Loan) TableName() string {
	return "loan"
}
