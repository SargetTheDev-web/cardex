// internal/models/attendance_patron_report.go

package model

type AttendancePatronReportResponse struct {
	Meta    AttendancePatronReportMeta   `json:"meta"`
	Patrons []AttendancePatronReportItem `json:"patrons"`
}

type AttendancePatronReportMeta struct {
	IsLive  bool                          `json:"is_live"`
	Filters AttendancePatronReportFilters `json:"filters"`
}

type AttendancePatronReportFilters struct {
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}

type AttendancePatronReportItem struct {
	PatronName                 string `json:"patron_name"`
	InstitutionalID            string `json:"institutional_id"`
	Role                       string `json:"role"`
	CourseProgram              string `json:"course_program"`
	TotalVisits                int64  `json:"total_visits"`
	TotalStayMinutes           int64  `json:"total_stay_minutes"`
	TotalStayDurationFormatted string `json:"total_stay_duration_formatted"`
	AvgStayMinutes             int64  `json:"avg_stay_minutes"`
	AvgStayPerVisitFormatted   string `json:"avg_stay_per_visit_formatted"`
	LastVisitDate              string `json:"last_visit_date"`
}
