// internal/repository/audit_repository.go

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

func GetAuditActionID(
	db *gorm.DB,
	actionCode string,
) (int, error) {

	var actionID int

	err := db.
		Table("audit_action").
		Select("action_id").
		Where("action_code = ?", actionCode).
		First(&actionID).Error

	return actionID, err
}
