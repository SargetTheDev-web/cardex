// internal/repository/auth_repository.go

package repository

import "gorm.io/gorm"

func IncrementLoginAttempts(
	db *gorm.DB,
	userID int,
) error {
	return db.
		Table("user").
		Where("user_id = ?", userID).
		Update(
			"login_retry_count",
			gorm.Expr("login_retry_count + 1"),
		).Error
}

func ResetLoginAttempts(
	db *gorm.DB,
	userID int,
) error {
	return db.
		Table("user").
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"login_retry_count": 0,
			"status_id":         3,
			"datetimelocked":    nil,
		}).Error
}

func LockAccount(
	db *gorm.DB,
	userID int,
) error {
	return db.
		Table("user").
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"status_id":      5,
			"datetimelocked": gorm.Expr("NOW()"),
		}).Error
}
