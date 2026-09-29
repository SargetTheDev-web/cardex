package model

type UserRole struct {
	RoleID   int    `gorm:"column:role_id"`
	RoleCode string `gorm:"column:role_code"`
	RoleName string `gorm:"column:role_name"`
}

func (UserRole) TableName() string {
	return "user_role"
}
