// internal/models/attendance_day.go

package model

import "time"

type AttendanceDay struct {
	UserID         int       `gorm:"column:user_id;primaryKey"`
	Campus         string    `gorm:"column:campus;primaryKey"`
	AttendanceDate time.Time `gorm:"column:attendance_date;primaryKey;type:date"`
	FirstCheckInAt time.Time `gorm:"column:first_check_in_at"`
}

func (AttendanceDay) TableName() string {
	return "attendance_days"
}
