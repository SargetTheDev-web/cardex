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
		WITH visits AS (
			SELECT
				ci.id,
				ci.user_id,

				COALESCE(
					NULLIF(BTRIM(up.course::text), ''),
					'Faculty / Staff'
				) AS course_program,

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

		ORDER BY total_visits DESC, course_program ASC
	`

	err := db.Raw(query, fromDate, toDate).Scan(&rows).Error
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
		SELECT COUNT(DISTINCT user_id)
		FROM attendance_logs

		WHERE
			scan_type = 'CHECK_IN'

			AND scanned_at >= (
				?::date::timestamp AT TIME ZONE 'Asia/Manila'
			)

			AND scanned_at < (
				(?::date + INTERVAL '1 day')
				AT TIME ZONE 'Asia/Manila'
			)
	`

	if err := db.Raw(query, fromDate, toDate).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf(
			"count unique attendance patrons: %w",
			err,
		)
	}

	return count, nil
}
