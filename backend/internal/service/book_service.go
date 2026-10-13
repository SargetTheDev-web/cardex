// internal/service/book_service.go

package service

import (
	"errors"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var ErrInvalidBook = errors.New("invalid book")

func validateBook(book *model.Book) error {
	book.Title = strings.TrimSpace(book.Title)
	book.Language = strings.TrimSpace(book.Language)

	if book.Title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidBook)
	}

	if book.Language == "" {
		return fmt.Errorf("%w: language is required", ErrInvalidBook)
	}

	if book.ISBN != nil {
		value := strings.TrimSpace(*book.ISBN)
		if value == "" {
			book.ISBN = nil
		} else {
			book.ISBN = &value
		}
	}

	if book.Edition != nil {
		value := strings.TrimSpace(*book.Edition)
		if value == "" {
			book.Edition = nil
		} else {
			book.Edition = &value
		}
	}

	if book.Description != nil {
		value := strings.TrimSpace(*book.Description)
		if value == "" {
			book.Description = nil
		} else {
			book.Description = &value
		}
	}

	return nil
}

func validateBookReferences(tx *gorm.DB, book *model.Book) error {
	if book.CategoryID != nil {
		var exists bool
		err := tx.Raw(
			"SELECT EXISTS (SELECT 1 FROM category WHERE category_id = ?)",
			*book.CategoryID,
		).Scan(&exists).Error

		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: category not found", ErrInvalidBook)
		}
	}

	if book.PublisherID != nil {
		var exists bool
		err := tx.Raw(
			"SELECT EXISTS (SELECT 1 FROM publisher WHERE publisher_id = ?)",
			*book.PublisherID,
		).Scan(&exists).Error

		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: publisher not found", ErrInvalidBook)
		}
	}

	seen := make(map[int]struct{}, len(book.AuthorIDs))

	for _, authorID := range book.AuthorIDs {
		if authorID <= 0 {
			return fmt.Errorf("%w: author IDs must be positive", ErrInvalidBook)
		}

		if _, exists := seen[authorID]; exists {
			continue
		}
		seen[authorID] = struct{}{}

		var exists bool
		err := tx.Raw(
			"SELECT EXISTS (SELECT 1 FROM author WHERE author_id = ?)",
			authorID,
		).Scan(&exists).Error

		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf(
				"%w: author ID %d not found",
				ErrInvalidBook,
				authorID,
			)
		}
	}

	return nil
}

func writeBookAudit(
	tx *gorm.DB,
	userID int,
	actionCode string,
	description string,
	ip string,
	userAgent string,
) error {
	actionID, err := repository.GetAuditActionID(tx, actionCode)
	if err != nil {
		return fmt.Errorf("get audit action ID: %w", err)
	}

	moduleID, err := repository.GetAuditModuleID(tx, "CATALOG")
	if err != nil {
		return fmt.Errorf("get audit module ID: %w", err)
	}

	return repository.InsertAuditLog(
		tx, &userID, actionID, moduleID,
		description, ip, userAgent,
	)
}

func CreateBook(
	db *gorm.DB,
	book *model.Book,
	userID int,
	ip string,
	userAgent string,
) (*model.Book, error) {
	if err := validateBook(book); err != nil {
		return nil, err
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := validateBookReferences(tx, book); err != nil {
			return err
		}

		if err := repository.CreateBook(tx, book); err != nil {
			return err
		}

		if err := repository.SetBookAuthors(
			tx, book.BookID, book.AuthorIDs,
		); err != nil {
			return err
		}

		return writeBookAudit(
			tx,
			userID,
			"CREATE",
			fmt.Sprintf("Created book: %s (ID: %d)", book.Title, book.BookID),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return nil, err
	}

	return book, nil
}

func GetAllBooks(
	db *gorm.DB,
	userID int,
	ip string,
	userAgent string,
) ([]model.Book, error) {
	var books []model.Book

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		books, err = repository.GetAllBooks(tx)
		if err != nil {
			return err
		}

		return writeBookAudit(
			tx, userID, "READ", "Retrieved all books", ip, userAgent,
		)
	})

	if err != nil {
		return nil, err
	}

	return books, nil
}

func GetBookByID(
	db *gorm.DB,
	bookID int,
	userID int,
	ip string,
	userAgent string,
) (model.Book, error) {
	var book model.Book

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		book, err = repository.GetBookByID(tx, bookID)
		if err != nil {
			return err
		}

		return writeBookAudit(
			tx,
			userID,
			"READ",
			fmt.Sprintf("Retrieved book ID: %d", bookID),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return model.Book{}, err
	}

	return book, nil
}

// UpdateBook treats the request as a complete replacement of editable
// fields and author relationships, rather than a partial PATCH.
func UpdateBook(
	db *gorm.DB,
	bookID int,
	updated *model.Book,
	userID int,
	ip string,
	userAgent string,
) (model.Book, error) {
	if err := validateBook(updated); err != nil {
		return model.Book{}, err
	}

	var result model.Book

	err := db.Transaction(func(tx *gorm.DB) error {
		oldBook, err := repository.GetBookByID(tx, bookID)
		if err != nil {
			return err
		}

		if err := validateBookReferences(tx, updated); err != nil {
			return err
		}

		updates := map[string]interface{}{
			"category_id":      updated.CategoryID,
			"isbn":             updated.ISBN,
			"title":            updated.Title,
			"edition":          updated.Edition,
			"publisher_id":     updated.PublisherID,
			"publication_year": updated.PublicationYear,
			"language":         updated.Language,
			"description":      updated.Description,
		}

		if err := repository.UpdateBook(tx, bookID, updates); err != nil {
			return err
		}

		if err := repository.SetBookAuthors(
			tx, bookID, updated.AuthorIDs,
		); err != nil {
			return err
		}

		result, err = repository.GetBookByID(tx, bookID)
		if err != nil {
			return err
		}

		return writeBookAudit(
			tx,
			userID,
			"UPDATE",
			fmt.Sprintf(
				"Updated book ID %d: '%s'",
				bookID,
				oldBook.Title,
			),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return model.Book{}, err
	}

	return result, nil
}

func DeleteBook(
	db *gorm.DB,
	bookID int,
	userID int,
	ip string,
	userAgent string,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		book, err := repository.GetBookByID(tx, bookID)
		if err != nil {
			return err
		}

		if err := repository.DeleteBook(tx, bookID); err != nil {
			return err
		}

		return writeBookAudit(
			tx,
			userID,
			"DELETE",
			fmt.Sprintf("Deleted book: %s (ID: %d)", book.Title, book.BookID),
			ip,
			userAgent,
		)
	})
}

func ValidateBookISBN(db *gorm.DB, isbn string) (bool, error) {
	isbn = strings.TrimSpace(isbn)

	if isbn == "" {
		return false, fmt.Errorf("%w: ISBN is required", ErrInvalidBook)
	}

	exists, err := repository.BookISBNExists(db, isbn)
	if err != nil {
		return false, err
	}

	return exists, nil
}
