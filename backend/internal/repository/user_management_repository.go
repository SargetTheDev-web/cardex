// internal/repository/user_management_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type UserListRow struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	EmailAddress string `json:"email_address"`

	StatusID          int    `json:"status_id"`
	StatusCode        string `json:"status_code"`
	StatusName        string `json:"status_name"`
	StatusDescription string `json:"status_description"`
	IsAllowedLogin    bool   `json:"is_allowed_login"`

	RoleID          int    `json:"role_id"`
	RoleCode        string `json:"role_code"`
	RoleName        string `json:"role_name"`
	RoleDescription string `json:"role_description"`

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
	ProfileImageURL *string    `json:"profile_image_url"`

	ChangeRequestStatus *string `json:"change_request_status"`
	HasPendingChanges   bool    `json:"has_pending_changes"`
}

func GetUserList(
	db *gorm.DB,
) ([]UserListRow, error) {

	var users []UserListRow

	err := db.
		Raw(`
			WITH latest_pending AS (
				SELECT DISTINCT ON (user_id)
					*
				FROM profile_change_request
				WHERE status = 'PENDING'
				ORDER BY user_id, created_at DESC
			)
			SELECT
				u.user_id,

				COALESCE(pcr.username, u.username) AS username,
				COALESCE(pcr.email_address, u.email_address) AS email_address,

				u.expiration_date,

				up.profile_id,

				COALESCE(pcr.institutional_id, up.institutional_id) AS institutional_id,
				COALESCE(pcr.last_name, up.last_name) AS last_name,
				COALESCE(pcr.first_name, up.first_name) AS first_name,
				COALESCE(pcr.middle_name, up.middle_name) AS middle_name,
				COALESCE(pcr.suffix_extension, up.suffix_extension) AS suffix_extension,
				COALESCE(pcr.mobile_number, up.mobile_number) AS mobile_number,
				COALESCE(pcr.birth_date, up.birth_date) AS birth_date,
				COALESCE(pcr.course, up.course) AS course,
				COALESCE(pcr.profile_image_url, up.profile_image_url) AS profile_image_url,

				r.role_id,
				r.role_code,
				r.role_name,
				r.description AS role_description,

				s.status_id,
				s.status_code,
				s.status_name,
				s.is_allowed_login,
				s.description AS status_description,

				pcr.status AS change_request_status,
				(pcr.change_request_id IS NOT NULL) AS has_pending_changes

			FROM "user" u

			LEFT JOIN user_profile up
				ON up.user_id = u.user_id

			LEFT JOIN latest_pending pcr
				ON pcr.user_id = u.user_id

			LEFT JOIN user_role r
				ON r.role_id = COALESCE(pcr.requested_role_id, u.role_id)

			LEFT JOIN user_status s
				ON s.status_id = u.status_id

			ORDER BY u.user_id ASC
		`).
		Scan(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}
