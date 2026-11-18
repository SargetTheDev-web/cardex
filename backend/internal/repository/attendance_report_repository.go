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
}

func GetAttendanceReport(
	db *gorm.DB,
	fromDate string,
	toDate string,
	courseProgram string,
) ([]AttendanceReportRow, error) {

	var records []AttendanceReportRow

	query := `
		SELECT
			ci.id,

			TO_CHAR(
				ci.scanned_at AT TIME ZONE 'Asia/Manila',
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

			ci.campus AS station,

			TO_CHAR(
				ci.scanned_at AT TIME ZONE 'Asia/Manila',
				'YYYY-MM-DD"T"HH24:MI:SS'
			) AS time_in,

			CASE
				WHEN next_scan.scan_type = 'CHECK_OUT'
				THEN TO_CHAR(
					next_scan.scanned_at AT TIME ZONE 'Asia/Manila',
					'YYYY-MM-DD"T"HH24:MI:SS'
				)
				ELSE NULL
			END AS time_out

		FROM attendance_logs ci

		LEFT JOIN LATERAL (
			SELECT
				al.scan_type,
				al.scanned_at

			FROM attendance_logs al

			WHERE
				al.user_id = ci.user_id
				AND al.campus = ci.campus
				AND al.scanned_at > ci.scanned_at
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

		LEFT JOIN LATERAL (
			SELECT
				p.first_name,
				p.middle_name,
				p.last_name,
				p.suffix_extension,
				p.institutional_id,
				p.course

			FROM user_profile p

			WHERE p.user_id = ci.user_id

			ORDER BY p.profile_id DESC
			LIMIT 1
		) up ON TRUE

		WHERE
			ci.scan_type = 'CHECK_IN'

			AND ci.scanned_at >= (
				?::date::timestamp AT TIME ZONE 'Asia/Manila'
			)

			AND ci.scanned_at < (
				(?::date + INTERVAL '1 day')
				AT TIME ZONE 'Asia/Manila'
			)

			AND (
				? = ''
				OR COALESCE(
					NULLIF(BTRIM(up.course::text), ''),
					'Faculty / Staff'
				) = ?
			)

		ORDER BY ci.scanned_at DESC, ci.id DESC
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
