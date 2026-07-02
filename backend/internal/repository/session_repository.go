// internal/repository/session_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

func CreateSession(
	db *gorm.DB,
	userID int,
	token string,
	ip string,
	userAgent string,
	expiry time.Time,
) error {
	return db.Table("user_session").Create(map[string]interface{}{
		"user_id":       userID,
		"session_token": token,
		"ip_address":    ip,
		"user_agent":    userAgent,
		"expires_at":    expiry,
	}).Error
}
