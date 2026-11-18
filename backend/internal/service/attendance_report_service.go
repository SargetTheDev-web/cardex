// internal/service/attendance_report_service.go

package service

import (
	"fmt"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var AttendanceAvailableCourses = []string{
	"BSA",
	"BSIS",
	"BSTM",
	"BSBA",
	"Faculty / Staff",
}

func GetAttendanceReport(
	db *gorm.DB,
	fromDate string,
	toDate string,
	courseProgram string,
) (*model.AttendanceReportResponse, error) {

	loc, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		return nil, fmt.Errorf("load Manila timezone: %w", err)
	}

	// Validate date range.
	startDate, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid from_date: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return nil, fmt.Errorf("invalid to_date: %w", err)
	}

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("to_date cannot be earlier than from_date")
	}

	// Validate course filter.
	validCourse := false

	if courseProgram == "" ||
		courseProgram == "All courses / departments" {
		courseProgram = ""
	} else {
		for _, course := range AttendanceAvailableCourses {
			if courseProgram == course {
				validCourse = true
				break
			}
		}

		if !validCourse {
			return nil, fmt.Errorf("invalid course_program filter")
		}
	}

	rows, err := repository.GetAttendanceReport(
		db,
		fromDate,
		toDate,
		courseProgram,
	)
	if err != nil {
		return nil, err
	}

	records := make(
		[]model.AttendanceReportRecord,
		0,
		len(rows),
	)

	for _, row := range rows {
		durationMinutes := 0
		durationFormatted := "0h 0m"
		timeOut := ""

		// No checkout means zero duration.
		if row.TimeOut != nil {
			parsedTimeIn, err := time.ParseInLocation(
				"2006-01-02T15:04:05",
				row.TimeIn,
				loc,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"parse attendance time in: %w",
					err,
				)
			}

			parsedTimeOut, err := time.ParseInLocation(
				"2006-01-02T15:04:05",
				*row.TimeOut,
				loc,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"parse attendance time out: %w",
					err,
				)
			}

			elapsed := parsedTimeOut.Sub(parsedTimeIn)

			if elapsed < 0 {
				return nil, fmt.Errorf(
					"invalid attendance duration for log %d",
					row.ID,
				)
			}

			durationMinutes = int(elapsed.Minutes())

			hours := durationMinutes / 60
			minutes := durationMinutes % 60

			durationFormatted = fmt.Sprintf(
				"%dh %dm",
				hours,
				minutes,
			)

			timeOut = *row.TimeOut
		}

		records = append(records, model.AttendanceReportRecord{
			ID:                fmt.Sprintf("att-%03d", row.ID),
			Date:              row.Date,
			PatronName:        row.PatronName,
			InstitutionalID:   row.InstitutionalID,
			CourseProgram:     row.CourseProgram,
			Station:           row.Station,
			TimeIn:            row.TimeIn,
			TimeOut:           timeOut,
			DurationMinutes:   durationMinutes,
			DurationFormatted: durationFormatted,
		})
	}

	return &model.AttendanceReportResponse{
		Meta: model.AttendanceReportMeta{
			IsLive: true,
			Filters: model.AttendanceReportFilters{
				FromDate: fromDate,
				ToDate:   toDate,
				CourseProgram: func() string {
					if courseProgram == "" {
						return "All courses / departments"
					}
					return courseProgram
				}(),
			},
			AvailableCourses: AttendanceAvailableCourses,
		},
		Records: records,
	}, nil
}
