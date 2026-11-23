// internal/service/top_borrowed_books_service.go

package service

import (
	"time"

	"backend/internal/repository"

	"gorm.io/gorm"
)

type TopBorrowedBooksReport struct {
	Meta  TopBorrowedBooksMeta            `json:"meta"`
	Books []repository.TopBorrowedBookRow `json:"books"`
}

type TopBorrowedBooksMeta struct {
	IsLive       bool      `json:"is_live"`
	GeneratedAt  time.Time `json:"generated_at"`
	TotalBorrows int64     `json:"total_borrows"`
}

func GetTopBorrowedBooksReport(
	db *gorm.DB,
) (TopBorrowedBooksReport, error) {

	report := TopBorrowedBooksReport{
		Meta: TopBorrowedBooksMeta{
			IsLive:      true,
			GeneratedAt: time.Now().UTC(),
		},
		Books: make([]repository.TopBorrowedBookRow, 0),
	}

	result, err := repository.GetTopBorrowedBooks(db)
	if err != nil {
		return report, err
	}

	report.Meta.TotalBorrows = result.TotalBorrows
	report.Books = result.Books

	return report, nil
}
