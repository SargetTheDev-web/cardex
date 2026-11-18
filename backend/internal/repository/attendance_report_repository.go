// internal/repository/attendance_report_repository.go

package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type AttendanceReportRow struct {
	ID              int64   `gorm:"column:id"`
	Date            string  `gorm:"column:attendance_date"`
	PatronName      string  `gorm:"column:patron_name"`
	InstitutionalID string  `gorm:"column:institutional_id"`
	CourseProgram   string  `gorm:"column:course_program"`
	Station         string  `gorm:"column:station"`
	TimeIn          string  `gorm:"column:time_in"`
	TimeOut         *string `gorm:"column:time_out"`

	// Internal calculation field.
	// Do not expose this directly in the API response.
	DurationMinutes int64 `gorm:"column:duration_minutes"`
}

func GetAttendanceReport(
	db *gorm.DB,
	fromDate string,
	toDate string,
	courseProgram string,
) ([]AttendanceReportRow, error) {

	var records []AttendanceReportRow

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

				MIN(scanned_at) FILTER (
					WHERE scan_type = 'CHECK_IN'
				) AS first_check_in_at,

				MAX(next_scanned_at) FILTER (
					WHERE
						scan_type = 'CHECK_IN'
						AND next_scan_type = 'CHECK_OUT'
				) AS last_check_out_at,

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
		)

		SELECT
			d.first_check_in_id AS id,

			TO_CHAR(
				d.attendance_date,
				'YYYY-MM-DD'
			) AS attendance_date,

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

			COALESCE(
				NULLIF(BTRIM(up.course::text), ''),
				'Faculty / Staff'
			) AS course_program,

			d.campus AS station,

			TO_CHAR(
				d.first_check_in_at AT TIME ZONE 'Asia/Manila',
				'YYYY-MM-DD"T"HH24:MI:SS'
			) AS time_in,

			CASE
				WHEN d.last_check_out_at IS NOT NULL
				THEN TO_CHAR(
					d.last_check_out_at AT TIME ZONE 'Asia/Manila',
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
				ELSE NULL
			END AS time_out,

			d.duration_minutes

		FROM (
			SELECT
				dt.*,

				(
					SELECT MIN(ds.id)
					FROM daily_scans ds
					WHERE
						ds.user_id = dt.user_id
						AND ds.campus = dt.campus
						AND ds.attendance_date = dt.attendance_date
						AND ds.scan_type = 'CHECK_IN'
				) AS first_check_in_id

			FROM daily_totals dt
		) d

		LEFT JOIN LATERAL (
			SELECT
				p.first_name,
				p.middle_name,
				p.last_name,
				p.suffix_extension,
				p.institutional_id,
				p.course

			FROM user_profile p

			WHERE p.user_id = d.user_id

			ORDER BY p.profile_id DESC
			LIMIT 1
		) up ON TRUE

		WHERE
			? = ''
			OR COALESCE(
				NULLIF(BTRIM(up.course::text), ''),
				'Faculty / Staff'
			) = ?

		ORDER BY
			d.attendance_date DESC,
			d.first_check_in_at DESC,
			d.first_check_in_id DESC
	`

	err := db.Raw(
		query,
		fromDate,
		toDate,
		courseProgram,
		courseProgram,
	).Scan(&records).Error

	if err != nil {
		return nil, fmt.Errorf(
			"get attendance report: %w",
			err,
		)
	}

	if records == nil {
		records = []AttendanceReportRow{}
	}

	return records, nil
}
