// internal/models/attendance_log.go

package model

import "time"

type AttendanceLog struct {
	ID             int64     `gorm:"column:id;primaryKey"`
	UserID         int       `gorm:"column:user_id"`
	Campus         string    `gorm:"column:campus"`
	ScanType       string    `gorm:"column:scan_type"`
	ScannedAt      time.Time `gorm:"column:scanned_at"`
	KioskSessionID *int64    `gorm:"column:kiosk_session_id"`
}

func (AttendanceLog) TableName() string {
	return "attendance_logs"
}
