// internal/models/attendance_report.go

package model

type AttendanceReportResponse struct {
	Meta    AttendanceReportMeta     `json:"meta"`
	Records []AttendanceReportRecord `json:"records"`
}

type AttendanceReportMeta struct {
	IsLive           bool                    `json:"is_live"`
	Filters          AttendanceReportFilters `json:"filters"`
	AvailableCourses []string                `json:"available_courses"`
}

type AttendanceReportFilters struct {
	FromDate      string `json:"from_date"`
	ToDate        string `json:"to_date"`
	CourseProgram string `json:"course_program"`
}

type AttendanceReportRecord struct {
	ID                string `json:"id"`
	Date              string `json:"date"`
	PatronName        string `json:"patron_name"`
	InstitutionalID   string `json:"institutional_id"`
	CourseProgram     string `json:"course_program"`
	Station           string `json:"station"`
	TimeIn            string `json:"time_in"`
	TimeOut           string `json:"time_out"`
	DurationMinutes   int    `json:"duration_minutes"`
	DurationFormatted string `json:"duration_formatted"`
}
