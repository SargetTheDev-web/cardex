// internal/repository/attendance_patron_report_repository.go

package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type AttendancePatronReportRow struct {
	PatronName      string `gorm:"column:patron_name"`
	InstitutionalID string `gorm:"column:institutional_id"`
	Role            string `gorm:"column:role"`
	CourseProgram   string `gorm:"column:course_program"`

	TotalVisits      int64 `gorm:"column:total_visits"`
	TotalStayMinutes int64 `gorm:"column:total_stay_minutes"`
	AvgStayMinutes   int64 `gorm:"column:avg_stay_minutes"`

	LastVisitDate string `gorm:"column:last_visit_date"`
}

func GetAttendancePatronReport(
	db *gorm.DB,
	fromDate string,
	toDate string,
) ([]AttendancePatronReportRow, error) {

	var rows []AttendancePatronReportRow

	query := `
		WITH selected_visits AS (
			SELECT
				ci.id,
				ci.user_id,

				CASE
					WHEN next_scan.scan_type = 'CHECK_OUT'
					THEN GREATEST(
						0,
						FLOOR(
							EXTRACT(
								EPOCH FROM (
									next_scan.scanned_at - ci.scanned_at
								)
							) / 60
						)
					)::BIGINT
					ELSE 0
				END AS duration_minutes

			FROM attendance_logs ci

			LEFT JOIN LATERAL (
				SELECT
					al.scan_type,
					al.scanned_at

				FROM attendance_logs al

				WHERE
					al.user_id = ci.user_id
					AND al.campus = ci.campus
					AND (al.scanned_at, al.id) > (ci.scanned_at, ci.id)
					AND al.scanned_at < (
						(
							DATE(
								ci.scanned_at AT TIME ZONE 'Asia/Manila'
							) + INTERVAL '1 day'
						) AT TIME ZONE 'Asia/Manila'
					)

				ORDER BY al.scanned_at ASC, al.id ASC
				LIMIT 1
			) next_scan ON TRUE

			WHERE
				ci.scan_type = 'CHECK_IN'

				AND ci.scanned_at >= (
					?::date::timestamp AT TIME ZONE 'Asia/Manila'
				)

				AND ci.scanned_at < (
					(?::date + INTERVAL '1 day')
					AT TIME ZONE 'Asia/Manila'
				)
		),

		patron_totals AS (
			SELECT
				user_id,
				COUNT(*) AS total_visits,
				SUM(duration_minutes) AS total_stay_minutes,

				ROUND(
					AVG(duration_minutes)
				)::BIGINT AS avg_stay_minutes

			FROM selected_visits

			GROUP BY user_id
		),

		latest_visits AS (
			SELECT
				user_id,
				MAX(
					DATE(
						scanned_at AT TIME ZONE 'Asia/Manila'
					)
				) AS last_visit_date

			FROM attendance_logs

			WHERE scan_type = 'CHECK_IN'

			GROUP BY user_id
		)

		SELECT
			COALESCE(
				NULLIF(
					TRIM(CONCAT_WS(' ',
						NULLIF(BTRIM(up.first_name), ''),
						NULLIF(BTRIM(up.middle_name), ''),
						NULLIF(BTRIM(up.last_name), ''),
						NULLIF(BTRIM(up.suffix_extension::text), '')
					)),
					''
				),
				'Unknown User'
			) AS patron_name,

			COALESCE(up.institutional_id, '') AS institutional_id,

			CASE
				WHEN u.role_id = 5 THEN 'Student'
				WHEN u.role_id = 4 THEN 'Faculty / Staff'
				WHEN u.role_id IN (1, 2, 3) THEN 'Faculty / Staff'
				ELSE 'Guest'
			END AS role,

			CASE
				WHEN u.role_id IN (1, 2, 3, 4)
					THEN 'Faculty / Staff'
				ELSE COALESCE(
					NULLIF(BTRIM(up.course::text), ''),
					'Faculty / Staff'
				)
			END AS course_program,

			pt.total_visits,
			pt.total_stay_minutes,
			pt.avg_stay_minutes,

			COALESCE(
				TO_CHAR(
					lv.last_visit_date,
					'YYYY-MM-DD'
				),
				''
			) AS last_visit_date

		FROM patron_totals pt

		INNER JOIN "user" u
			ON u.user_id = pt.user_id

		LEFT JOIN LATERAL (
			SELECT
				p.first_name,
				p.middle_name,
				p.last_name,
				p.suffix_extension,
				p.institutional_id,
				p.course

			FROM user_profile p

			WHERE p.user_id = pt.user_id

			ORDER BY p.profile_id DESC
			LIMIT 1
		) up ON TRUE

		LEFT JOIN latest_visits lv
			ON lv.user_id = pt.user_id

		ORDER BY
			pt.total_stay_minutes DESC,
			pt.total_visits DESC,
			patron_name ASC
	`

	err := db.Raw(
		query,
		fromDate,
		toDate,
	).Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf(
			"get attendance patron report: %w",
			err,
		)
	}

	if rows == nil {
		rows = []AttendancePatronReportRow{}
	}

	return rows, nil
}
