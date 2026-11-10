// internal/repository/recaptcha_verification.go

package repository

import "gorm.io/gorm"

func InsertRecaptchaLog(
	db *gorm.DB,
	email string,
	token string,
	result bool,
	ip string,
) error {
	return db.Table("recaptcha_log").Create(map[string]interface{}{
		"email_address":       email,
		"recaptcha_token":     token,
		"verification_result": result,
		"ip_address":          ip,
	}).Error
}
