// internal/repository/attendance_course_report_repository.go

package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type AttendanceCourseReportRow struct {
	CourseProgram   string  `gorm:"column:course_program"`
	TotalVisits     int64   `gorm:"column:total_visits"`
	UniqueVisitors  int64   `gorm:"column:unique_visitors"`
	TotalMinutes    int64   `gorm:"column:total_minutes"`
	AvgVisitMinutes int64   `gorm:"column:avg_visit_minutes"`
	SharePercentage float64 `gorm:"column:share_percentage"`
}

func GetAttendanceCourseBreakdown(
	db *gorm.DB,
	fromDate string,
	toDate string,
) ([]AttendanceCourseReportRow, error) {

	var rows []AttendanceCourseReportRow

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

		visits AS (
			SELECT
				d.user_id,

				COALESCE(
					NULLIF(BTRIM(up.course::text), ''),
					'Faculty / Staff'
				) AS course_program,

				COALESCE(dt.duration_minutes, 0) AS duration_minutes

			FROM attendance_days d

			LEFT JOIN daily_totals dt
				ON dt.user_id = d.user_id
				AND dt.campus = d.campus
				AND dt.attendance_date = d.attendance_date

			LEFT JOIN LATERAL (
				SELECT
					p.course

				FROM user_profile p

				WHERE p.user_id = d.user_id

				ORDER BY p.profile_id DESC
				LIMIT 1
			) up ON TRUE

			WHERE
				d.attendance_date BETWEEN ?::date AND ?::date
		),

		course_totals AS (
			SELECT
				course_program,
				COUNT(*) AS total_visits,
				COUNT(DISTINCT user_id) AS unique_visitors,
				SUM(duration_minutes) AS total_minutes,

				ROUND(
					AVG(duration_minutes)
				)::BIGINT AS avg_visit_minutes

			FROM visits

			GROUP BY course_program
		)

		SELECT
			course_program,
			total_visits,
			unique_visitors,
			total_minutes,
			avg_visit_minutes,

			ROUND(
				(
					total_visits::NUMERIC /
					NULLIF(SUM(total_visits) OVER (), 0)
				) * 100,
				1
			) AS share_percentage

		FROM course_totals

		ORDER BY
			total_visits DESC,
			course_program ASC
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
			"get attendance course breakdown: %w",
			err,
		)
	}

	if rows == nil {
		rows = []AttendanceCourseReportRow{}
	}

	return rows, nil
}

func GetUniqueAttendancePatrons(
	db *gorm.DB,
	fromDate string,
	toDate string,
) (int64, error) {

	var count int64

	query := `
		SELECT
			COUNT(DISTINCT user_id)

		FROM attendance_days

		WHERE
			attendance_date BETWEEN ?::date AND ?::date
	`

	if err := db.Raw(
		query,
		fromDate,
		toDate,
	).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf(
			"count unique attendance patrons: %w",
			err,
		)
	}

	return count, nil
}
