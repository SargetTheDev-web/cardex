// internal/models/attendance_course_report.go

package model

type AttendanceCourseReportResponse struct {
	Meta    AttendanceCourseReportMeta    `json:"meta"`
	Summary AttendanceCourseReportSummary `json:"summary"`
	Courses []AttendanceCourseReportItem  `json:"courses"`
}

type AttendanceCourseReportMeta struct {
	IsLive  bool                          `json:"is_live"`
	Filters AttendanceCourseReportFilters `json:"filters"`
}

type AttendanceCourseReportFilters struct {
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}

type AttendanceCourseReportSummary struct {
	TopVisitingCourse string `json:"top_visiting_course"`
	TotalVisits       int64  `json:"total_visits"`
	UniquePatrons     int64  `json:"unique_patrons"`
}

type AttendanceCourseReportItem struct {
	CourseProgram             string  `json:"course_program"`
	TotalVisits               int64   `json:"total_visits"`
	UniqueVisitors            int64   `json:"unique_visitors"`
	TotalMinutes              int64   `json:"total_minutes"`
	TotalHoursFormatted       string  `json:"total_hours_formatted"`
	AvgVisitMinutes           int64   `json:"avg_visit_minutes"`
	AvgVisitDurationFormatted string  `json:"avg_visit_duration_formatted"`
	SharePercentage           float64 `json:"share_percentage"`
}
