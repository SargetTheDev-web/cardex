// internal/models/role_permission.go

package model

import "time"

type RolePermission struct {
	RoleID       int       `gorm:"column:role_id;primaryKey"`
	PermissionID int       `gorm:"column:permission_id;primaryKey"`
	CreatedBy    *int      `gorm:"column:created_by"`
	UpdatedBy    *int      `gorm:"column:updated_by"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (RolePermission) TableName() string {
	return "role_permission"
}
