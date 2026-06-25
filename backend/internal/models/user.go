package models

import "time"

type User struct {
	UserID          int
	Username        string
	EmailAddress    string
	PasswordHash    string
	StatusID        int
	RoleID          int
	LoginRetryCount int
	DateTimeLocked  *time.Time
	ExpirationDate  time.Time
	LastActivityAt  time.Time
}
