// internal/repository/portal_repository.go

package repository

import (
	"time"

	"gorm.io/gorm"
)

type PortalStatistics struct {
	BooksBorrowed int64 `json:"books_borrowed"`
	Reservations  int64 `json:"reservations"`
	Overdue       int64 `json:"overdue"`
}

type PortalAuthor struct {
	AuthorID   int     `json:"author_id"`
	AuthorName string  `json:"author_name"`
	Biography  *string `json:"biography,omitempty"`
}

type PortalCategory struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Description  *string `json:"description,omitempty"`
}

type PortalPublisher struct {
	PublisherID   int     `json:"publisher_id"`
	PublisherName string  `json:"publisher_name"`
	Address       *string `json:"address,omitempty"`
	ContactNumber *string `json:"contact_number,omitempty"`
}

type PortalBook struct {
	BookID          int              `json:"book_id"`
	ISBN            *string          `json:"isbn,omitempty"`
	Title           string           `json:"title"`
	Edition         *string          `json:"edition,omitempty"`
	PublicationYear *int             `json:"publication_year,omitempty"`
	Language        string           `json:"language"`
	Description     *string          `json:"description,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Category        *PortalCategory  `json:"category,omitempty"`
	Publisher       *PortalPublisher `json:"publisher,omitempty"`
	Authors         []PortalAuthor   `json:"authors"`
}

type PortalTransaction struct {
	TransactionID string      `json:"transaction_id" gorm:"column:transaction_id"`
	BookID        int         `json:"book_id" gorm:"column:book_id"`
	Type          string      `json:"type" gorm:"column:type"`
	Date          time.Time   `json:"date" gorm:"column:transaction_date"`
	Status        string      `json:"status" gorm:"column:status"`
	Book          *PortalBook `json:"book,omitempty" gorm:"-"`
}

// GetPortalStatistics retrieves dashboard statistics for one user.
func GetPortalStatistics(
	db *gorm.DB,
	userID int,
) (PortalStatistics, error) {
	var statistics PortalStatistics

	const query = `
		SELECT
			(
				SELECT COUNT(*)
				FROM loan
				WHERE user_id = ?
				  AND return_date IS NULL
				  AND status::text IN ('BORROWED', 'OVERDUE')
			) AS books_borrowed,

			(
				SELECT COUNT(*)
				FROM reservation
				WHERE user_id = ?
				  AND status::text IN ('PENDING', 'NOTIFIED')
				  AND (
						expiry_date IS NULL
						OR expiry_date > CURRENT_TIMESTAMP
				  )
			) AS reservations,

			(
				SELECT COUNT(*)
				FROM loan
				WHERE user_id = ?
				  AND return_date IS NULL
				  AND (
						status::text = 'OVERDUE'
						OR (
							status::text = 'BORROWED'
							AND due_date < CURRENT_DATE
						)
				  )
			) AS overdue
	`

	err := db.Raw(
		query,
		userID,
		userID,
		userID,
	).Scan(&statistics).Error

	if err != nil {
		return PortalStatistics{}, err
	}

	return statistics, nil
}

// GetPortalRecentTransactions retrieves the latest borrowing,
// return, and reservation activity for one user.
func GetPortalRecentTransactions(
	db *gorm.DB,
	userID int,
	limit int,
) ([]PortalTransaction, error) {
	transactions := make([]PortalTransaction, 0)

	if limit <= 0 {
		limit = 10
	}

	const query = `
		SELECT *
		FROM (
			-- Borrowing activity
			SELECT
				CONCAT('LOAN-', l.loan_id) AS transaction_id,
				bc.book_id,
				'BORROW' AS type,
				l.loan_date::timestamp AS transaction_date,
				CASE
					WHEN l.return_date IS NOT NULL THEN 'RETURNED'
					WHEN l.status::text = 'OVERDUE'
						OR (
							l.status::text = 'BORROWED'
							AND l.due_date < CURRENT_DATE
						) THEN 'OVERDUE'
					ELSE l.status::text
				END AS status
			FROM loan l
			INNER JOIN book_copy bc
				ON bc.copy_id = l.copy_id
			WHERE l.user_id = ?

			UNION ALL

			-- Return activity
			SELECT
				CONCAT('RETURN-', l.loan_id) AS transaction_id,
				bc.book_id,
				'RETURN' AS type,
				l.return_date::timestamp AS transaction_date,
				'RETURNED' AS status
			FROM loan l
			INNER JOIN book_copy bc
				ON bc.copy_id = l.copy_id
			WHERE l.user_id = ?
			  AND l.return_date IS NOT NULL

			UNION ALL

			-- Reservation creation activity
			SELECT
				CONCAT('RESERVATION-', r.reservation_id) AS transaction_id,
				r.book_id,
				'RESERVATION' AS type,
				r.reservation_date AS transaction_date,
				r.status::text AS status
			FROM reservation r
			WHERE r.user_id = ?
		) AS activity
		ORDER BY transaction_date DESC, transaction_id DESC
		LIMIT ?
	`

	err := db.Raw(
		query,
		userID,
		userID,
		userID,
		limit,
	).Scan(&transactions).Error

	if err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return transactions, nil
	}

	// Collect distinct book IDs.
	bookIDs := make([]int, 0, len(transactions))
	bookIDSet := make(map[int]struct{})

	for _, transaction := range transactions {
		if _, exists := bookIDSet[transaction.BookID]; !exists {
			bookIDSet[transaction.BookID] = struct{}{}
			bookIDs = append(bookIDs, transaction.BookID)
		}
	}

	books, err := GetPortalBooksByIDs(db, bookIDs)
	if err != nil {
		return nil, err
	}

	// Attach book details to each transaction.
	for i := range transactions {
		if book, exists := books[transactions[i].BookID]; exists {
			transactions[i].Book = &book
		}
	}

	return transactions, nil
}

// GetPortalBooksByIDs retrieves full book information
// for a set of book IDs.
func GetPortalBooksByIDs(
	db *gorm.DB,
	bookIDs []int,
) (map[int]PortalBook, error) {
	books := make(map[int]PortalBook)

	if len(bookIDs) == 0 {
		return books, nil
	}

	// Retrieve the book, category, and publisher information.
	type bookRow struct {
		BookID          int       `gorm:"column:book_id"`
		ISBN            *string   `gorm:"column:isbn"`
		Title           string    `gorm:"column:title"`
		Edition         *string   `gorm:"column:edition"`
		PublicationYear *int      `gorm:"column:publication_year"`
		Language        string    `gorm:"column:language"`
		Description     *string   `gorm:"column:description"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UpdatedAt       time.Time `gorm:"column:updated_at"`

		CategoryID          *int    `gorm:"column:category_id"`
		CategoryName        *string `gorm:"column:category_name"`
		CategoryDescription *string `gorm:"column:category_description"`

		PublisherID      *int    `gorm:"column:publisher_id"`
		PublisherName    *string `gorm:"column:publisher_name"`
		PublisherAddress *string `gorm:"column:publisher_address"`
		PublisherContact *string `gorm:"column:publisher_contact"`
	}

	var rows []bookRow

	const bookQuery = `
		SELECT
			b.book_id,
			b.isbn,
			b.title,
			b.edition,
			b.publication_year,
			b.language,
			b.description,
			b.created_at,
			b.updated_at,

			c.category_id,
			c.category_name,
			c.description AS category_description,

			p.publisher_id,
			p.publisher_name,
			p.address AS publisher_address,
			p.contact_number AS publisher_contact

		FROM book b

		LEFT JOIN category c
			ON c.category_id = b.category_id

		LEFT JOIN publisher p
			ON p.publisher_id = b.publisher_id

		WHERE b.book_id IN ?
	`

	err := db.Raw(bookQuery, bookIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		book := PortalBook{
			BookID:          row.BookID,
			ISBN:            row.ISBN,
			Title:           row.Title,
			Edition:         row.Edition,
			PublicationYear: row.PublicationYear,
			Language:        row.Language,
			Description:     row.Description,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
			Authors:         make([]PortalAuthor, 0),
		}

		if row.CategoryID != nil && row.CategoryName != nil {
			book.Category = &PortalCategory{
				CategoryID:   *row.CategoryID,
				CategoryName: *row.CategoryName,
				Description:  row.CategoryDescription,
			}
		}

		if row.PublisherID != nil && row.PublisherName != nil {
			book.Publisher = &PortalPublisher{
				PublisherID:   *row.PublisherID,
				PublisherName: *row.PublisherName,
				Address:       row.PublisherAddress,
				ContactNumber: row.PublisherContact,
			}
		}

		books[row.BookID] = book
	}

	// Retrieve authors for all requested books.
	type authorRow struct {
		BookID     int     `gorm:"column:book_id"`
		AuthorID   int     `gorm:"column:author_id"`
		AuthorName string  `gorm:"column:author_name"`
		Biography  *string `gorm:"column:biography"`
	}

	var authorRows []authorRow

	const authorQuery = `
		SELECT
			ba.book_id,
			a.author_id,
			a.author_name,
			a.biography
		FROM book_author ba
		INNER JOIN author a
			ON a.author_id = ba.author_id
		WHERE ba.book_id IN ?
		ORDER BY a.author_name ASC
	`

	err = db.Raw(authorQuery, bookIDs).Scan(&authorRows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range authorRows {
		book, exists := books[row.BookID]
		if !exists {
			continue
		}

		book.Authors = append(book.Authors, PortalAuthor{
			AuthorID:   row.AuthorID,
			AuthorName: row.AuthorName,
			Biography:  row.Biography,
		})

		books[row.BookID] = book
	}

	return books, nil
}
