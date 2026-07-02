package model

import "time"

type PasswordHistory struct {
	HistoryID    int       `gorm:"column:history_id;primaryKey"`
	UserID       int       `gorm:"column:user_id"`
	PasswordHash string    `gorm:"column:password_hash"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (PasswordHistory) TableName() string {
	return "password_history"
}
