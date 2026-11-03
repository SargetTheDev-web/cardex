// internal/models/notification.go

package model

import "time"

type Notification struct {
	NotificationID int64     `json:"notification_id" gorm:"column:notification_id;primaryKey"`
	UserID         int       `json:"user_id" gorm:"column:user_id;not null"`
	Type           string    `json:"type" gorm:"column:type;not null"`
	Channel        string    `json:"channel" gorm:"column:channel;not null"`
	Subject        string    `json:"subject" gorm:"column:subject;not null"`
	Message        string    `json:"message" gorm:"column:message;not null"`
	IsRead         bool      `json:"is_read" gorm:"column:is_read;not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;not null"`

	ReferenceType *string `json:"reference_type,omitempty" gorm:"column:reference_type"`
	ReferenceID   *int64  `json:"reference_id,omitempty" gorm:"column:reference_id"`
}

func (Notification) TableName() string {
	return "notification"
}
