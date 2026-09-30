package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func GetUserRole(db *gorm.DB, userID int) (*model.UserRole, error) {
	var role model.UserRole

	err := db.
		Table("user_role ur").
		Select(`
			ur.role_id,
			ur.role_code,
			ur.role_name,
			ur.is_active,
			ur.description,
			ur.created_by,
			ur.updated_by
		`).
		Joins(`JOIN "user" u ON u.role_id = ur.role_id`).
		Where("u.user_id = ?", userID).
		First(&role).Error

	if err != nil {
		return nil, err
	}

	return &role, nil
}

func GetRolePermissions(
	db *gorm.DB,
	roleID int,
) ([]string, error) {

	var permissions []string

	err := db.
		Table("role_permission rp").
		Select("p.permission_code").
		Joins(
			"JOIN permission p ON p.permission_id = rp.permission_id",
		).
		Where("rp.role_id = ?", roleID).
		Order("p.permission_code ASC").
		Pluck("p.permission_code", &permissions).Error

	if err != nil {
		return nil, err
	}

	return permissions, nil
}
