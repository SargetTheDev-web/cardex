// internal/models/attendance_kiosk_session.go

package model

import "time"

type AttendanceKioskSession struct {
	KioskSessionID int64      `gorm:"column:kiosk_session_id;primaryKey"`
	CredentialHash string     `gorm:"column:credential_hash"`
	Campus         string     `gorm:"column:campus"`
	ActivatedBy    int        `gorm:"column:activated_by"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	ExpiresAt      time.Time  `gorm:"column:expires_at"`
	RevokedAt      *time.Time `gorm:"column:revoked_at"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at"`
}

func (AttendanceKioskSession) TableName() string {
	return "attendance_kiosk_sessions"
}
