// internal/models/user_profile.go

package model

import "time"

type UserProfile struct {
	ProfileID int `gorm:"column:profile_id;primaryKey"`

	UserID int `gorm:"column:user_id;not null"`

	InstitutionalID string `gorm:"column:institutional_id;not null"`

	LastName        string  `gorm:"column:last_name;not null"`
	FirstName       string  `gorm:"column:first_name;not null"`
	MiddleName      *string `gorm:"column:middle_name"`
	SuffixExtension *string `gorm:"column:suffix_extension"`

	Course *string `gorm:"column:course"`

	MobileNumber *string    `gorm:"column:mobile_number"`
	BirthDate    *time.Time `gorm:"column:birth_date"`

	CreatedBy *int `gorm:"column:created_by"`
	UpdatedBy *int `gorm:"column:updated_by"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserProfile) TableName() string {
	return "user_profile"
}
