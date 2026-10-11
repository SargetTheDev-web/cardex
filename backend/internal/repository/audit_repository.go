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

	var result struct {
		ActionID int `gorm:"column:action_id"`
	}

	err := db.
		Table("audit_action").
		Select("action_id").
		Where("action_code = ?", actionCode).
		First(&result).Error

	if err != nil {
		return 0, err
	}

	return result.ActionID, nil
}

func GetAuditModuleID(
	db *gorm.DB,
	moduleCode string,
) (int, error) {
	var result struct {
		ModuleID int `gorm:"column:module_id"`
	}

	err := db.
		Table("audit_module").
		Select("module_id").
		Where("module_code = ?", moduleCode).
		First(&result).Error

	if err != nil {
		return 0, err
	}

	return result.ModuleID, nil
}
