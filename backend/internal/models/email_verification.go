// internal/models/email_verification.go

package model

import "time"

type EmailVerification struct {
	VerificationID   int       `gorm:"column:verification_id;primaryKey"`
	EmailAddress     string    `gorm:"column:email_address"`
	VerificationCode string    `gorm:"column:verification_code"`
	ExpiresAt        time.Time `gorm:"column:expires_at"`
	IsVerified       bool      `gorm:"column:is_verified"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (EmailVerification) TableName() string {
	return "email_verification"
}
