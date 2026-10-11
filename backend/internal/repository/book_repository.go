// internal/repository/book_repository.go

package repository

import (
	"errors"

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
