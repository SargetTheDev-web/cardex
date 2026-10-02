package repository

import (
	"time"

	"gorm.io/gorm"
)

type UserListRow struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	EmailAddress string `json:"email_address"`

	StatusID   int    `json:"status_id"`
	StatusCode string `json:"status_code"`
	StatusName string `json:"status_name"`

	RoleID   int    `json:"role_id"`
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`

	ExpirationDate time.Time `json:"expiration_date"`

	ProfileID       *int       `json:"profile_id"`
	InstitutionalID *string    `json:"institutional_id"`
	LastName        *string    `json:"last_name"`
	FirstName       *string    `json:"first_name"`
	MiddleName      *string    `json:"middle_name"`
	SuffixExtension *string    `json:"suffix_extension"`
	Course          *string    `json:"course"`
	MobileNumber    *string    `json:"mobile_number"`
	BirthDate       *time.Time `json:"birth_date"`
}

func GetUserList(
	db *gorm.DB,
) ([]UserListRow, error) {

	var users []UserListRow

	err := db.
		Table(`"user" AS u`).
		Select(`
			u.user_id,
			u.username,
			u.email_address,

			u.status_id,
			s.status_code,
			s.status_name,

			u.role_id,
			r.role_code,
			r.role_name,

			u.expiration_date,

			up.profile_id,
			up.institutional_id,
			up.last_name,
			up.first_name,
			up.middle_name,
			up.suffix_extension,
			up.course,
			up.mobile_number,
			up.birth_date
		`).
		Joins(`
			LEFT JOIN user_profile up
				ON up.user_id = u.user_id
		`).
		Joins(`
			LEFT JOIN user_status s
				ON s.status_id = u.status_id
		`).
		Joins(`
			LEFT JOIN user_role r
				ON r.role_id = u.role_id
		`).
		Order("u.user_id ASC").
		Scan(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}
