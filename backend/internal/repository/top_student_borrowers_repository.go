// internal/repository/top_student_borrowers_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type TopStudentBorrowerRow struct {
	Rank                int       `json:"rank"`
	StudentName         string    `json:"student_name"`
	InstitutionalID     *string   `json:"institutional_id"`
	CourseProgram       *string   `json:"course_program"`
	TotalBooksBorrowed  int64     `json:"total_books_borrowed"`
	BooksReturnedOnTime int64     `json:"books_returned_on_time"`
	LastBorrowDate      time.Time `json:"last_borrow_date"`
	OverdueCount        int64     `json:"overdue_count"`
	OverdueStatus       string    `json:"overdue_status"`
}

type TopStudentBorrowersResult struct {
	Students []TopStudentBorrowerRow `json:"students"`
}

func GetTopStudentBorrowers(
	db *gorm.DB,
) (TopStudentBorrowersResult, error) {

	result := TopStudentBorrowersResult{
		Students: make([]TopStudentBorrowerRow, 0),
	}

	err := db.Raw(`
		WITH borrower_stats AS (
			SELECT
				u.user_id,

				COALESCE(
					NULLIF(
						BTRIM(
							CONCAT_WS(
								' ',
								up.first_name,
								up.middle_name,
								up.last_name,
								up.suffix_extension
							)
						),
						''
					),
					u.username
				) AS student_name,

				up.institutional_id,
				up.course::TEXT AS course_program,

				COUNT(l.loan_id)::BIGINT AS total_books_borrowed,

				COUNT(l.loan_id) FILTER (
					WHERE l.return_date IS NOT NULL
					  AND l.return_date <= l.due_date
				)::BIGINT AS books_returned_on_time,

				MAX(l.loan_date) AS last_borrow_date,

				COUNT(l.loan_id) FILTER (
					WHERE l.return_date IS NULL
					  AND l.due_date < CURRENT_DATE
				)::BIGINT AS overdue_count

			FROM "user" u

			INNER JOIN user_profile up
				ON up.user_id = u.user_id

			INNER JOIN loan l
				ON l.user_id = u.user_id

			WHERE u.role_id = 5

			GROUP BY
				u.user_id,
				u.username,
				up.first_name,
				up.middle_name,
				up.last_name,
				up.suffix_extension,
				up.institutional_id,
				up.course

			HAVING COUNT(l.loan_id) > 0
		),

		ranked_students AS (
			SELECT
				ROW_NUMBER() OVER (
					ORDER BY
						bs.total_books_borrowed DESC,
						bs.student_name ASC,
						bs.user_id ASC
				)::INT AS rank,

				bs.student_name,
				bs.institutional_id,
				bs.course_program,
				bs.total_books_borrowed,
				bs.books_returned_on_time,
				bs.last_borrow_date,
				bs.overdue_count

			FROM borrower_stats bs
		)

		SELECT
			rs.rank,
			rs.student_name,
			rs.institutional_id,
			rs.course_program,
			rs.total_books_borrowed,
			rs.books_returned_on_time,
			rs.last_borrow_date,
			rs.overdue_count,

			CASE
				WHEN rs.overdue_count = 0
					THEN '0 — Clean'
				ELSE
					rs.overdue_count::TEXT || ' — Overdue'
			END AS overdue_status

		FROM ranked_students rs

		WHERE rs.rank <= 5

		ORDER BY rs.rank ASC
	`).Scan(&result.Students).Error

	if err != nil {
		return result, err
	}

	return result, nil
}
