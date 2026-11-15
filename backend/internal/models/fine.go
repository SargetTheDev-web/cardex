// internal/models/fine.go

package model

import "time"

type Fine struct {
	FineID    int64      `gorm:"column:fine_id;primaryKey" json:"fine_id"`
	LoanID    int64      `gorm:"column:loan_id;not null" json:"loan_id"`
	Amount    float64    `gorm:"column:amount;not null" json:"amount"`
	Reason    string     `gorm:"column:reason;not null" json:"reason"`
	Status    string     `gorm:"column:status;not null" json:"status"`
	PaidAt    *time.Time `gorm:"column:paid_at" json:"paid_at,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;not null" json:"created_at"`
}

func (Fine) TableName() string {
	return "fine"
}

type FineResponse struct {
	FineID          int64      `json:"fine_id"`
	LoanID          int64      `json:"loan_id"`
	UserID          int        `json:"user_id"`
	Name            string     `json:"name"`
	RoleCode        string     `json:"role_code"`
	InstitutionalID *string    `json:"institutional_id,omitempty"`
	Amount          float64    `json:"amount"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`

	BookTitle  string     `json:"book_title" gorm:"column:book_title"`
	ISBN       string     `json:"isbn" gorm:"column:isbn"`
	LoanDate   time.Time  `json:"loan_date" gorm:"column:loan_date"`
	DueDate    time.Time  `json:"due_date" gorm:"column:due_date"`
	ReturnDate *time.Time `json:"return_date" gorm:"column:return_date"`

	BorrowNotes *string `json:"borrow_notes" gorm:"column:borrow_notes"`
	ReturnNotes *string `json:"return_notes" gorm:"column:return_notes"`
}
