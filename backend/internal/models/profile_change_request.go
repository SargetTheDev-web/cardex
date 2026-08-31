package model

import "time"

type ProfileChangeRequest struct {
	ChangeRequestID int64 `gorm:"column:change_request_id;primaryKey"`

	UserID int `gorm:"column:user_id;not null"`

	Username     *string `gorm:"column:username"`
	EmailAddress *string `gorm:"column:email_address"`

	InstitutionalID *string    `gorm:"column:institutional_id"`
	LastName        *string    `gorm:"column:last_name"`
	FirstName       *string    `gorm:"column:first_name"`
	MiddleName      *string    `gorm:"column:middle_name"`
	SuffixExtension *string    `gorm:"column:suffix_extension"`
	MobileNumber    *string    `gorm:"column:mobile_number"`
	BirthDate       *time.Time `gorm:"column:birth_date"`

	Status string `gorm:"column:status"`

	ReviewedBy      *int       `gorm:"column:reviewed_by"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at"`
	RejectionReason *string    `gorm:"column:rejection_reason"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ProfileChangeRequest) TableName() string {
	return "profile_change_request"
}
