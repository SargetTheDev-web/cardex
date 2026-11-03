// internal/models/email_outbox.go

package model

import (
	"encoding/json"
	"time"
)

type EmailOutbox struct {
	EmailOutboxID  int64           `json:"email_outbox_id" gorm:"column:email_outbox_id;primaryKey"`
	RecipientEmail string          `json:"recipient_email" gorm:"column:recipient_email;not null"`
	RecipientName  string          `json:"recipient_name" gorm:"column:recipient_name;not null"`
	Subject        string          `json:"subject" gorm:"column:subject;not null"`
	Heading        string          `json:"heading" gorm:"column:heading;not null"`
	Message        string          `json:"message" gorm:"column:message;not null"`
	Details        json.RawMessage `json:"details" gorm:"column:details;type:jsonb;not null"`

	Status        string     `json:"status" gorm:"column:status;not null"`
	AttemptCount  int        `json:"attempt_count" gorm:"column:attempt_count;not null"`
	NextAttemptAt time.Time  `json:"next_attempt_at" gorm:"column:next_attempt_at;not null"`
	LastError     *string    `json:"last_error,omitempty" gorm:"column:last_error"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at;not null"`
	SentAt        *time.Time `json:"sent_at,omitempty" gorm:"column:sent_at"`
	ClaimedAt     *time.Time `json:"claimed_at,omitempty" gorm:"column:claimed_at"`
}

type EmailOutboxDetail struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (EmailOutbox) TableName() string {
	return "email_outbox"
}
