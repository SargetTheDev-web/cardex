package models

import "time"

type User struct {
	UserID          int        `json:"user_id"`
	Username        string     `json:"username"`
	EmailAddress    string     `json:"email_address"`
	PasswordHash    string     `json:"password_hash"`
	StatusID        int        `json:"status_id"`
	RoleID          int        `json:"role_id"`
	LoginRetryCount int        `json:"login_retry_count"`
	DateTimeLocked  *time.Time `json:"datetimelocked"`
	ExpirationDate  time.Time  `json:"expiration_date"`
	LastActivityAt  time.Time  `json:"last_activity_at"`
}
