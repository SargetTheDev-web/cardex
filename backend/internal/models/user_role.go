package model

type UserRole struct {
	RoleID      int    `gorm:"column:role_id;primaryKey"`
	RoleCode    string `gorm:"column:role_code"`
	RoleName    string `gorm:"column:role_name"`
	IsActive    bool   `gorm:"column:is_active"`
	Description string `gorm:"column:description"`
	CreatedBy   *int   `gorm:"column:created_by"`
	UpdatedBy   *int   `gorm:"column:updated_by"`
}

func (UserRole) TableName() string {
	return "user_role"
}
