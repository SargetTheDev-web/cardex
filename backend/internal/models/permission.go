// internal/models/permission.go

package model

import "time"

type Permission struct {
	PermissionID   int       `gorm:"column:permission_id" json:"permission_id"`
	PermissionCode string    `gorm:"column:permission_code" json:"permission_code"`
	PermissionName string    `gorm:"column:permission_name" json:"permission_name"`
	Description    string    `gorm:"column:description" json:"description"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Permission) TableName() string {
	return "permission"
}
