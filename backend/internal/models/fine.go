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
