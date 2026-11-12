// internal/repository/audit_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

func InsertAuditLog(
	db *gorm.DB,
	userID *int,
	actionID int,
	moduleID int,
	description string,
	ip string,
	userAgent string,
) error {
	return db.Exec(`
		INSERT INTO audit_log
		(user_id, action_id, module_id, description, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	).Error
}

func GetAuditActionID(
	db *gorm.DB,
	actionCode string,
) (int, error) {

	var result struct {
		ActionID int `gorm:"column:action_id"`
	}

	err := db.
		Table("audit_action").
		Select("action_id").
		Where("action_code = ?", actionCode).
		First(&result).Error

	if err != nil {
		return 0, err
	}

	return result.ActionID, nil
}

func GetAuditModuleID(
	db *gorm.DB,
	moduleCode string,
) (int, error) {
	var result struct {
		ModuleID int `gorm:"column:module_id"`
	}

	err := db.
		Table("audit_module").
		Select("module_id").
		Where("module_code = ?", moduleCode).
		First(&result).Error

	if err != nil {
		return 0, err
	}

	return result.ModuleID, nil
}

type AuditDashboardSummary struct {
	TotalEventsToday int64 `json:"total_events_today"`
	LoginEvents      int64 `json:"login_events"`
	DataChanges      int64 `json:"data_changes"`
	SecurityEvents   int64 `json:"security_events"`
	AdminActions     int64 `json:"admin_actions"`
}

type AuditDashboardLog struct {
	ID              int64     `json:"id"`
	UserID          *int      `json:"user_id"`
	Timestamp       time.Time `json:"timestamp"`
	FullName        *string   `json:"full_name"`
	Role            *string   `json:"role"`
	RoleCode        *string   `json:"role_code"`
	InstitutionalID *string   `json:"institutional_id"`
	EventType       string    `json:"event_type"`
	Description     string    `json:"description"`
	IPAddress       string    `json:"ip_address"`
	Severity        string    `json:"severity"`
}

type auditDashboardLogRow struct {
	ID              int64     `gorm:"column:id"`
	UserID          *int      `gorm:"column:user_id"`
	Timestamp       time.Time `gorm:"column:timestamp"`
	FullName        *string   `gorm:"column:full_name"`
	Role            *string   `gorm:"column:role"`
	RoleCode        *string   `gorm:"column:role_code"`
	InstitutionalID *string   `gorm:"column:institutional_id"`
	EventType       string    `gorm:"column:event_type"`
	Description     string    `gorm:"column:description"`
	IPAddress       string    `gorm:"column:ip_address"`
	Severity        string    `gorm:"column:severity"`
}

func GetAuditDashboardSummary(
	db *gorm.DB,
	start, end time.Time,
) (AuditDashboardSummary, error) {
	var summary AuditDashboardSummary

	err := db.Raw(`
		SELECT
			COUNT(*) AS total_events_today,
			COUNT(*) FILTER (
				WHERE aa.action_code IN ('LOGIN_SUCCESS', 'LOGIN_FAILURE', 'LOGOUT')
			) AS login_events,
			COUNT(*) FILTER (
				WHERE aa.action_code IN ('CREATE', 'UPDATE', 'DELETE')
			) AS data_changes,
			COUNT(*) FILTER (
				WHERE aa.action_code IN ('LOGIN_FAILURE', 'LOCKOUT')
			) AS security_events,
			COUNT(*) FILTER (
				WHERE aa.action_code = 'APPROVE'
				   OR u.role_id = 1
			) AS admin_actions
		FROM audit_log al
		JOIN audit_action aa ON aa.action_id = al.action_id
		LEFT JOIN "user" u ON u.user_id = al.user_id
		WHERE al.created_at >= ?
		  AND al.created_at < ?
	`, start, end).Scan(&summary).Error

	return summary, err
}

func GetRecentAuditDashboardLogs(
	db *gorm.DB,
	limit int,
) ([]AuditDashboardLog, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}

	var rows []auditDashboardLogRow

	err := db.Raw(`
		SELECT
			al.log_id AS id,
			al.user_id AS user_id,
			al.created_at AS timestamp,

			COALESCE(
				NULLIF(
					TRIM(CONCAT_WS(' ',
						NULLIF(TRIM(up.first_name), ''),
						NULLIF(TRIM(up.middle_name), ''),
						NULLIF(TRIM(up.last_name), ''),
						NULLIF(TRIM(up.suffix_extension), '')
					)),
					''
				),
				NULLIF(
					TRIM(CONCAT_WS(' ',
						NULLIF(TRIM(pcr.first_name), ''),
						NULLIF(TRIM(pcr.middle_name), ''),
						NULLIF(TRIM(pcr.last_name), ''),
						NULLIF(TRIM(pcr.suffix_extension), '')
					)),
					''
				),
				NULLIF(TRIM(u.username), '')
			) AS full_name,

			ur.role_name AS role,
			ur.role_code AS role_code,

			COALESCE(
				NULLIF(TRIM(up.institutional_id), ''),
				NULLIF(TRIM(pcr.institutional_id), '')
			) AS institutional_id,

			aa.action_name AS event_type,
			al.description,
			al.ip_address,

			CASE
				WHEN aa.action_code IN ('LOGIN_FAILURE', 'LOCKOUT')
					THEN 'HIGH'
				WHEN aa.action_code IN ('DELETE', 'UPDATE')
					THEN 'MEDIUM'
				ELSE 'LOW'
			END AS severity

		FROM audit_log al

		JOIN audit_action aa
			ON aa.action_id = al.action_id

		LEFT JOIN "user" u
			ON u.user_id = al.user_id

		LEFT JOIN user_profile up
			ON up.user_id = u.user_id

		LEFT JOIN LATERAL (
			SELECT
				p.first_name,
				p.middle_name,
				p.last_name,
				p.suffix_extension,
				p.institutional_id
			FROM profile_change_request p
			WHERE p.user_id = u.user_id
			  AND p.status = 'PENDING'
			ORDER BY
				p.created_at DESC,
				p.change_request_id DESC
			LIMIT 1
		) pcr ON TRUE

		LEFT JOIN user_role ur
			ON ur.role_id = u.role_id

		ORDER BY
			al.created_at DESC,
			al.log_id DESC

		LIMIT ?
	`, limit).Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	logs := make([]AuditDashboardLog, 0, len(rows))

	for _, row := range rows {
		logs = append(logs, AuditDashboardLog{
			ID:              row.ID,
			UserID:          row.UserID,
			Timestamp:       row.Timestamp,
			FullName:        row.FullName,
			Role:            row.Role,
			RoleCode:        row.RoleCode,
			InstitutionalID: row.InstitutionalID,
			EventType:       row.EventType,
			Description:     row.Description,
			IPAddress:       row.IPAddress,
			Severity:        row.Severity,
		})
	}

	return logs, nil
}
