// internal/service/top_student_borrowers_service.go

package service

import (
	"time"

	"backend/internal/repository"

	"gorm.io/gorm"
)

type TopStudentBorrowersReport struct {
	Meta     TopStudentBorrowersMeta            `json:"meta"`
	Students []repository.TopStudentBorrowerRow `json:"students"`
}

type TopStudentBorrowersMeta struct {
	IsLive      bool      `json:"is_live"`
	GeneratedAt time.Time `json:"generated_at"`
	DateFrom    string    `json:"date_from"`
	DateTo      string    `json:"date_to"`
}

func GetTopStudentBorrowersReport(
	db *gorm.DB,
	from time.Time,
	to time.Time,
) (TopStudentBorrowersReport, error) {

	report := TopStudentBorrowersReport{
		Meta: TopStudentBorrowersMeta{
			IsLive:      true,
			GeneratedAt: time.Now().UTC(),
			DateFrom:    from.Format("2006-01-02"),
			DateTo:      to.Format("2006-01-02"),
		},
		Students: make([]repository.TopStudentBorrowerRow, 0),
	}

	result, err := repository.GetTopStudentBorrowers(db, from, to)
	if err != nil {
		return report, err
	}

	report.Students = result.Students

	return report, nil
}
