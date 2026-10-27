// internal/repository/book_repository.go

package repository

import (
	"errors"
	"strings"

	model "backend/internal/models"

	"gorm.io/gorm"
)

// CreateBook inserts a book record.
func CreateBook(db *gorm.DB, book *model.Book) error {
	return db.Create(book).Error
}

// GetAllBooks retrieves books ordered by ID.
func GetAllBooks(db *gorm.DB) ([]model.Book, error) {
	var books []model.Book

	err := db.Order("book_id ASC").Find(&books).Error
	if err != nil {
		return nil, err
	}

	for i := range books {
		authorIDs, err := GetBookAuthorIDs(db, books[i].BookID)
		if err != nil {
			return nil, err
		}
		books[i].AuthorIDs = authorIDs
	}

	return books, nil
}

// GetBookByID retrieves one book and its author IDs.
func GetBookByID(db *gorm.DB, bookID int) (model.Book, error) {
	var book model.Book

	err := db.First(&book, "book_id = ?", bookID).Error
	if err != nil {
		return model.Book{}, err
	}

	book.AuthorIDs, err = GetBookAuthorIDs(db, bookID)
	if err != nil {
		return model.Book{}, err
	}

	return book, nil
}

// GetBookAuthorIDs retrieves the authors linked to a book.
func GetBookAuthorIDs(db *gorm.DB, bookID int) ([]int, error) {
	var authorIDs []int

	err := db.Table("book_author").
		Where("book_id = ?", bookID).
		Order("author_id ASC").
		Pluck("author_id", &authorIDs).Error
	if err != nil {
		return nil, err
	}

	// Return an empty array rather than null when there are no authors.
	if authorIDs == nil {
		authorIDs = []int{}
	}

	return authorIDs, nil
}

// SetBookAuthors replaces all author relationships for a book.
// Call this inside a transaction.
func SetBookAuthors(db *gorm.DB, bookID int, authorIDs []int) error {
	if err := db.Table("book_author").
		Where("book_id = ?", bookID).
		Delete(map[string]interface{}{}).Error; err != nil {
		return err
	}

	seen := make(map[int]struct{}, len(authorIDs))

	for _, authorID := range authorIDs {
		if authorID <= 0 {
			return errors.New("author IDs must be positive")
		}

		if _, exists := seen[authorID]; exists {
			continue
		}
		seen[authorID] = struct{}{}

		if err := db.Table("book_author").Create(map[string]interface{}{
			"book_id":   bookID,
			"author_id": authorID,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

// UpdateBook updates editable book fields.
// Call this inside a transaction.
func UpdateBook(db *gorm.DB, bookID int, updates map[string]interface{}) error {
	result := db.Model(&model.Book{}).
		Where("book_id = ?", bookID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// DeleteBook deletes a book record.
// The book_author rows are expected to cascade through the FK.
func DeleteBook(db *gorm.DB, bookID int) error {
	result := db.Delete(&model.Book{}, bookID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func BookISBNExists(db *gorm.DB, isbn string) (bool, error) {
	var count int64

	normalizedISBN := strings.ReplaceAll(strings.TrimSpace(isbn), "-", "")

	err := db.Model(&model.Book{}).
		Where(
			"REPLACE(TRIM(isbn), '-', '') = ?",
			normalizedISBN,
		).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

type BookCatalogSummary struct {
	TotalTitles     int64 `json:"total_titles"`
	TotalCopies     int64 `json:"total_copies"`
	AvailableCopies int64 `json:"available_copies"`
	BorrowedCopies  int64 `json:"borrowed_copies"`
	ReservedCopies  int64 `json:"reserved_copies"`
	OverdueLoans    int64 `json:"overdue_loans"`
	ReservationsDue int64 `json:"reservations_due"`
}

func GetBookCatalogSummary(db *gorm.DB) (BookCatalogSummary, error) {
	var summary BookCatalogSummary

	err := db.Raw(`
        SELECT
    (SELECT COUNT(*)
     FROM book) AS total_titles,

    (SELECT COUNT(*)
     FROM book_copy) AS total_copies,

    (SELECT COUNT(*)
     FROM book_copy
     WHERE copy_status::text = 'AVAILABLE') AS available_copies,

    (SELECT COUNT(*)
     FROM book_copy
     WHERE copy_status::text = 'BORROWED') AS borrowed_copies,

    (SELECT COUNT(*)
     FROM book_copy
     WHERE copy_status::text = 'RESERVED') AS reserved_copies,

    (SELECT COUNT(*)
     FROM loan
     WHERE status::text = 'OVERDUE'
        OR (
            status::text = 'BORROWED'
            AND due_date < CURRENT_DATE
        )) AS overdue_loans,

    (SELECT COUNT(*)
     FROM reservation
     WHERE status::text IN ('PENDING', 'NOTIFIED')
       AND expiry_date <= CURRENT_DATE) AS reservations_due;
    `).Scan(&summary).Error

	if err != nil {
		return BookCatalogSummary{}, err
	}

	return summary, nil
}
