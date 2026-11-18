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
		WITH daily_scans AS (
			SELECT
				d.user_id,
				d.campus,
				d.attendance_date,

				al.id,
				al.scan_type,
				al.scanned_at,

				LEAD(al.scan_type) OVER (
					PARTITION BY
						d.user_id,
						d.campus,
						d.attendance_date
					ORDER BY
						al.scanned_at,
						al.id
				) AS next_scan_type,

				LEAD(al.scanned_at) OVER (
					PARTITION BY
						d.user_id,
						d.campus,
						d.attendance_date
					ORDER BY
						al.scanned_at,
						al.id
				) AS next_scanned_at

			FROM attendance_days d

			INNER JOIN attendance_logs al
				ON al.user_id = d.user_id
				AND al.campus = d.campus
				AND al.scanned_at >= (
					d.attendance_date::timestamp
					AT TIME ZONE 'Asia/Manila'
				)
				AND al.scanned_at < (
					(d.attendance_date + 1)::timestamp
					AT TIME ZONE 'Asia/Manila'
				)

			WHERE
				d.attendance_date BETWEEN ?::date AND ?::date
		),

		daily_totals AS (
			SELECT
				user_id,
				campus,
				attendance_date,

				COALESCE(
					SUM(
						GREATEST(
							0,
							FLOOR(
								EXTRACT(
									EPOCH FROM (
										next_scanned_at - scanned_at
									)
								) / 60
							)
						)::BIGINT
					) FILTER (
						WHERE
							scan_type = 'CHECK_IN'
							AND next_scan_type = 'CHECK_OUT'
					),
					0
				) AS duration_minutes

			FROM daily_scans

			GROUP BY
				user_id,
				campus,
				attendance_date
		),

		patron_totals AS (
			SELECT
				d.user_id,

				COUNT(*) AS total_visits,

				SUM(
					COALESCE(dt.duration_minutes, 0)
				) AS total_stay_minutes,

				ROUND(
					AVG(
						COALESCE(dt.duration_minutes, 0)
					)
				)::BIGINT AS avg_stay_minutes

			FROM attendance_days d

			LEFT JOIN daily_totals dt
				ON dt.user_id = d.user_id
				AND dt.campus = d.campus
				AND dt.attendance_date = d.attendance_date

			WHERE
				d.attendance_date BETWEEN ?::date AND ?::date

			GROUP BY d.user_id
		),

		latest_visits AS (
			SELECT
				user_id,
				MAX(attendance_date) AS last_visit_date

			FROM attendance_days

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
