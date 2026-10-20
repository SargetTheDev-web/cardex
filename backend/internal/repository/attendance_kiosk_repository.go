// internal/repository/attendance_kiosk_repository.go

package repository

import (
	"time"

	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreateAttendanceKioskSession(
	db *gorm.DB,
	session *model.AttendanceKioskSession,
) error {
	return db.Create(session).Error
}

func GetActiveAttendanceKioskSession(
	db *gorm.DB,
	credentialHash string,
) (*model.AttendanceKioskSession, error) {
	var session model.AttendanceKioskSession

	err := db.
		Where(
			"credential_hash = ? AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP",
			credentialHash,
		).
		First(&session).Error

	if err != nil {
		return nil, err
	}

	return &session, nil
}

func UpdateAttendanceKioskLastUsed(
	db *gorm.DB,
	kioskSessionID int64,
) error {
	now := time.Now().UTC()

	return db.
		Model(&model.AttendanceKioskSession{}).
		Where(
			"kiosk_session_id = ? AND revoked_at IS NULL AND expires_at > ?",
			kioskSessionID,
			now,
		).
		Update("last_used_at", now).Error
}

func RevokeAttendanceKioskSession(
	db *gorm.DB,
	kioskSessionID int64,
) error {
	now := time.Now().UTC()

	result := db.
		Model(&model.AttendanceKioskSession{}).
		Where(
			"kiosk_session_id = ? AND revoked_at IS NULL",
			kioskSessionID,
		).
		Update("revoked_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
