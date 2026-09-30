// internal/repository/session_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

func CreateSession(
	db *gorm.DB,
	userID int,
	tokenHash string,
	ip string,
	userAgent string,
	expiry time.Time,
) error {
	return db.Table("user_session").Create(map[string]interface{}{
		"user_id":       userID,
		"session_token": tokenHash,
		"ip_address":    ip,
		"user_agent":    userAgent,
		"expires_at":    expiry,
	}).Error
}

func DeleteSessionByToken(
	db *gorm.DB,
	tokenHash string,
) error {
	result := db.
		Table("user_session").
		Where("session_token = ?", tokenHash).
		Delete(nil)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func SessionExists(
	db *gorm.DB,
	tokenHash string,
) (bool, error) {

	var count int64

	err := db.
		Table("user_session").
		Where(
			"session_token = ? AND expires_at > CURRENT_TIMESTAMP",
			tokenHash,
		).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func IsSessionIdle(
	db *gorm.DB,
	tokenHash string,
	idleTimeoutMinutes int,
) (bool, error) {

	var lastActivity time.Time

	err := db.
		Table("user_session").
		Select("last_activity").
		Where("session_token = ?", tokenHash).
		Scan(&lastActivity).Error

	if err != nil {
		return false, err
	}

	idleLimit := lastActivity.Add(
		time.Duration(idleTimeoutMinutes) * time.Minute,
	)

	return time.Now().UTC().After(idleLimit.UTC()), nil
}

// Refresh the session's last activity timestamp.
func UpdateSessionActivity(
	db *gorm.DB,
	tokenHash string,
) error {

	return db.
		Table("user_session").
		Where("session_token = ?", tokenHash).
		Update("last_activity", time.Now().UTC()).
		Error
}
