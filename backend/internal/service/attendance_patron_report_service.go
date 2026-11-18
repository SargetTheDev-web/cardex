// internal/service/attendance_patron_report_service.go

package service

import (
	"fmt"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

func GetAttendancePatronReport(
	db *gorm.DB,
	fromDate string,
	toDate string,
) (*model.AttendancePatronReportResponse, error) {

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

	rows, err := repository.GetAttendancePatronReport(
		db,
		fromDate,
		toDate,
	)
	if err != nil {
		return nil, err
	}

	patrons := make(
		[]model.AttendancePatronReportItem,
		0,
		len(rows),
	)

	for _, row := range rows {
		patrons = append(patrons, model.AttendancePatronReportItem{
			PatronName:                 row.PatronName,
			InstitutionalID:            row.InstitutionalID,
			Role:                       row.Role,
			CourseProgram:              row.CourseProgram,
			TotalVisits:                row.TotalVisits,
			TotalStayMinutes:           row.TotalStayMinutes,
			TotalStayDurationFormatted: formatAttendanceDuration(row.TotalStayMinutes),
			AvgStayMinutes:             row.AvgStayMinutes,
			AvgStayPerVisitFormatted:   formatAttendanceDuration(row.AvgStayMinutes),
			LastVisitDate:              row.LastVisitDate,
		})
	}

	return &model.AttendancePatronReportResponse{
		Meta: model.AttendancePatronReportMeta{
			IsLive: true,
			Filters: model.AttendancePatronReportFilters{
				FromDate: fromDate,
				ToDate:   toDate,
			},
		},
		Patrons: patrons,
	}, nil
}
