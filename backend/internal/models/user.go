// internal/models/user.go

package model

import "time"

type User struct {
	UserID          int        `gorm:"column:user_id;primaryKey"`
	Username        string     `gorm:"column:username"`
	EmailAddress    string     `gorm:"column:email_address"`
	PasswordHash    string     `gorm:"column:password_hash" json:"-"`
	SSOProvider     *string    `gorm:"column:sso_provider"`
	SSOProviderID   *string    `gorm:"column:sso_provider_id"`
	StatusID        int        `gorm:"column:status_id"`
	RoleID          int        `gorm:"column:role_id"`
	LoginRetryCount int        `gorm:"column:login_retry_count"`
	DateTimeLocked  *time.Time `gorm:"column:datetimelocked"`
	ExpirationDate  time.Time  `gorm:"column:expiration_date"`
	LastActivityAt  *time.Time `gorm:"column:last_activity_at"`

	PINHash *string `gorm:"column:pin_hash" json:"-"`
}

func (User) TableName() string {
	return "user"
}
