// internal/models/user_status.go

package model

type UserStatus struct {
	StatusID       int    `gorm:"column:status_id;primaryKey"`
	StatusCode     string `gorm:"column:status_code"`
	StatusName     string `gorm:"column:status_name"`
	IsAllowedLogin bool   `gorm:"column:is_allowed_login"`
	Description    string `gorm:"column:description"`
	CreatedBy      *int   `gorm:"column:created_by"`
	UpdatedBy      *int   `gorm:"column:updated_by"`
}

func (UserStatus) TableName() string {
	return "user_status"
}
