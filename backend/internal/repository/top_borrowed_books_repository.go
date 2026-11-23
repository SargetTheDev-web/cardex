// internal/repository/top_borrowed_books_repository.go

package repository

import (
	"gorm.io/gorm"
)

type TopBorrowedBookRow struct {
	Rank            int     `json:"rank"`
	BookTitle       string  `json:"book_title"`
	ISBN            *string `json:"isbn"`
	Authors         string  `json:"authors"`
	Subject         *string `json:"subject"`
	TimesBorrowed   int64   `json:"times_borrowed"`
	UniqueBorrowers int64   `json:"unique_borrowers"`
	SharePercentage float64 `json:"share_percentage"`
}

type TopBorrowedBooksResult struct {
	TotalBorrows int64                `json:"total_borrows"`
	Books        []TopBorrowedBookRow `json:"books"`
}

func GetTopBorrowedBooks(
	db *gorm.DB,
) (TopBorrowedBooksResult, error) {

	result := TopBorrowedBooksResult{
		Books: make([]TopBorrowedBookRow, 0),
	}

	err := db.Raw(`
		WITH total_borrows AS (
			SELECT COUNT(*)::BIGINT AS total
			FROM loan
		),

		book_stats AS (
			SELECT
				b.book_id,
				b.title AS book_title,
				b.isbn,

				c.category_name AS subject,

				COALESCE(
					STRING_AGG(
						DISTINCT a.author_name,
						', ' ORDER BY a.author_name
					),
					''
				) AS authors,

				COUNT(l.loan_id)::BIGINT AS times_borrowed,
				COUNT(DISTINCT l.user_id)::BIGINT AS unique_borrowers

			FROM loan l

			INNER JOIN book_copy bc
				ON bc.copy_id = l.copy_id

			INNER JOIN book b
				ON b.book_id = bc.book_id

			LEFT JOIN category c
				ON c.category_id = b.category_id

			LEFT JOIN book_author ba
				ON ba.book_id = b.book_id

			LEFT JOIN author a
				ON a.author_id = ba.author_id

			GROUP BY
				b.book_id,
				b.title,
				b.isbn,
				c.category_name
		),

		ranked_books AS (
			SELECT
				ROW_NUMBER() OVER (
					ORDER BY
						bs.times_borrowed DESC,
						bs.book_title ASC,
						bs.book_id ASC
				)::INT AS rank,

				bs.book_title,
				bs.isbn,
				bs.authors,
				bs.subject,
				bs.times_borrowed,
				bs.unique_borrowers,

				ROUND(
					(
						bs.times_borrowed::NUMERIC
						* 100
						/ NULLIF(tb.total, 0)
					),
					1
				)::DOUBLE PRECISION AS share_percentage

			FROM book_stats bs
			CROSS JOIN total_borrows tb

			WHERE bs.times_borrowed > 0
		)

		SELECT
			rb.rank,
			rb.book_title,
			rb.isbn,
			rb.authors,
			rb.subject,
			rb.times_borrowed,
			rb.unique_borrowers,
			rb.share_percentage

		FROM ranked_books rb

		WHERE rb.rank <= 10

		ORDER BY rb.rank ASC
	`).Scan(&result.Books).Error

	if err != nil {
		return result, err
	}

	// Total borrow transactions across the library.
	err = db.Raw(`
		SELECT COUNT(*)::BIGINT
		FROM loan
	`).Scan(&result.TotalBorrows).Error

	if err != nil {
		return result, err
	}

	return result, nil
}
