// internal/service/attendance_course_report_service.go

package service

import (
	"fmt"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

func GetAttendanceCourseReport(
	db *gorm.DB,
	fromDate string,
	toDate string,
) (*model.AttendanceCourseReportResponse, error) {

	startDate, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid from_date: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return nil, fmt.Errorf("invalid to_date: %w", err)
	}

	if endDate.Before(startDate) {
		return nil, fmt.Errorf(
			"to_date cannot be earlier than from_date",
		)
	}

	rows, err := repository.GetAttendanceCourseBreakdown(
		db,
		fromDate,
		toDate,
	)
	if err != nil {
		return nil, err
	}

	uniquePatrons, err := repository.GetUniqueAttendancePatrons(
		db,
		fromDate,
		toDate,
	)
	if err != nil {
		return nil, err
	}

	courses := make(
		[]model.AttendanceCourseReportItem,
		0,
		len(rows),
	)

	var totalVisits int64
	topVisitingCourse := ""

	for i, row := range rows {
		totalVisits += row.TotalVisits

		if i == 0 {
			topVisitingCourse = row.CourseProgram
		}

		courses = append(courses, model.AttendanceCourseReportItem{
			CourseProgram:             row.CourseProgram,
			TotalVisits:               row.TotalVisits,
			UniqueVisitors:            row.UniqueVisitors,
			TotalMinutes:              row.TotalMinutes,
			TotalHoursFormatted:       formatAttendanceDuration(row.TotalMinutes),
			AvgVisitMinutes:           row.AvgVisitMinutes,
			AvgVisitDurationFormatted: formatAttendanceDuration(row.AvgVisitMinutes),
			SharePercentage:           row.SharePercentage,
		})
	}

	return &model.AttendanceCourseReportResponse{
		Meta: model.AttendanceCourseReportMeta{
			IsLive: true,
			Filters: model.AttendanceCourseReportFilters{
				FromDate: fromDate,
				ToDate:   toDate,
			},
		},
		Summary: model.AttendanceCourseReportSummary{
			TopVisitingCourse: topVisitingCourse,
			TotalVisits:       totalVisits,
			UniquePatrons:     uniquePatrons,
		},
		Courses: courses,
	}, nil
}

func formatAttendanceDuration(totalMinutes int64) string {
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	return fmt.Sprintf("%dh %dm", hours, minutes)
}
