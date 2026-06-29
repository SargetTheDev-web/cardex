package repository

import "gorm.io/gorm"

func InsertAuditLog(
	db *gorm.DB,
	userID *int,
	actionID int,
	moduleID int,
	description string,
	ip string,
	userAgent string,
) error {
	return db.Exec(`
		INSERT INTO audit_log
		(user_id, action_id, module_id, description, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	).Error
}
