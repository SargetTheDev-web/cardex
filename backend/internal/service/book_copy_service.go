// internal/service/book_copy_service.go

package service

import (
	"errors"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var ErrInvalidBookCopy = errors.New("invalid book copy")

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func validateCopyStatus(tx *gorm.DB, status string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("%w: copy status is required", ErrInvalidBookCopy)
	}

	var exists bool
	err := tx.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_enum e
			JOIN pg_type t ON t.oid = e.enumtypid
			JOIN pg_namespace n ON n.oid = t.typnamespace
			WHERE t.typname = 'copy_status'
			  AND n.nspname = current_schema()
			  AND e.enumlabel = ?
		)
	`, status).Scan(&exists).Error
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf(
			"%w: invalid copy status %q",
			ErrInvalidBookCopy,
			status,
		)
	}

	return nil
}

func validateBookCopyReferences(
	tx *gorm.DB,
	copy *model.BookCopy,
	excludeCopyID int,
) error {
	var bookExists bool
	if err := tx.Raw(
		"SELECT EXISTS (SELECT 1 FROM book WHERE book_id = ?)",
		copy.BookID,
	).Scan(&bookExists).Error; err != nil {
		return err
	}

	if !bookExists {
		return fmt.Errorf(
			"%w: referenced book does not exist",
			ErrInvalidBookCopy,
		)
	}

	if copy.Barcode != nil {
		var duplicate bool

		query := tx.Table("book_copy").
			Select("EXISTS (?)",
				tx.Table("book_copy").
					Select("1").
					Where("barcode = ?", *copy.Barcode),
			)

		// A copy may retain its own barcode during an update.
		if excludeCopyID > 0 {
			query = tx.Table("book_copy").
				Select("EXISTS (?)",
					tx.Table("book_copy").
						Select("1").
						Where(
							"barcode = ? AND copy_id <> ?",
							*copy.Barcode,
							excludeCopyID,
						),
				)
		}

		if err := query.Scan(&duplicate).Error; err != nil {
			return err
		}

		if duplicate {
			return fmt.Errorf(
				"%w: barcode is already assigned to another copy",
				ErrInvalidBookCopy,
			)
		}
	}

	return nil
}

func writeBookCopyAudit(
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
		tx,
		&userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	)
}

func CreateBookCopy(
	db *gorm.DB,
	copy *model.BookCopy,
	userID int,
	ip string,
	userAgent string,
) (*model.BookCopy, error) {
	if copy.BookID <= 0 {
		return nil, fmt.Errorf("%w: book ID is required", ErrInvalidBookCopy)
	}

	copy.Barcode = normalizeOptionalString(copy.Barcode)
	copy.ShelfLocation = normalizeOptionalString(copy.ShelfLocation)

	if strings.TrimSpace(copy.CopyStatus) == "" {
		copy.CopyStatus = "AVAILABLE"
	} else {
		copy.CopyStatus = strings.TrimSpace(copy.CopyStatus)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := validateCopyStatus(tx, copy.CopyStatus); err != nil {
			return err
		}

		if err := validateBookCopyReferences(tx, copy, 0); err != nil {
			return err
		}

		if err := repository.CreateBookCopy(tx, copy); err != nil {
			return err
		}

		return writeBookCopyAudit(
			tx,
			userID,
			"CREATE",
			fmt.Sprintf(
				"Added physical book copy ID %d for book ID %d",
				copy.CopyID,
				copy.BookID,
			),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return nil, err
	}

	return copy, nil
}

func GetAllBookCopies(
	db *gorm.DB,
	userID int,
	ip string,
	userAgent string,
) ([]model.BookCopy, error) {
	var copies []model.BookCopy

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		copies, err = repository.GetAllBookCopies(tx)
		if err != nil {
			return err
		}

		return writeBookCopyAudit(
			tx,
			userID,
			"READ",
			"Retrieved book copy inventory",
			ip,
			userAgent,
		)
	})

	if err != nil {
		return nil, err
	}

	return copies, nil
}

func GetBookCopyByID(
	db *gorm.DB,
	copyID int,
	userID int,
	ip string,
	userAgent string,
) (model.BookCopy, error) {
	var copy model.BookCopy

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		copy, err = repository.GetBookCopyByID(tx, copyID)
		if err != nil {
			return err
		}

		return writeBookCopyAudit(
			tx,
			userID,
			"READ",
			fmt.Sprintf("Retrieved book copy ID %d", copyID),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return model.BookCopy{}, err
	}

	return copy, nil
}

func UpdateBookCopy(
	db *gorm.DB,
	copyID int,
	updated *model.BookCopy,
	userID int,
	ip string,
	userAgent string,
) (model.BookCopy, error) {
	if updated.BookID <= 0 {
		return model.BookCopy{}, fmt.Errorf(
			"%w: book ID is required",
			ErrInvalidBookCopy,
		)
	}

	updated.Barcode = normalizeOptionalString(updated.Barcode)
	updated.ShelfLocation = normalizeOptionalString(updated.ShelfLocation)

	var result model.BookCopy

	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := repository.GetBookCopyByID(tx, copyID)
		if err != nil {
			return err
		}

		if err := validateBookCopyReferences(tx, updated, copyID); err != nil {
			return err
		}

		// Preserve the existing status; status changes use a dedicated operation.
		updates := map[string]interface{}{
			"book_id":        updated.BookID,
			"barcode":        updated.Barcode,
			"shelf_location": updated.ShelfLocation,
		}

		if err := repository.UpdateBookCopy(tx, copyID, updates); err != nil {
			return err
		}

		result, err = repository.GetBookCopyByID(tx, copyID)
		if err != nil {
			return err
		}

		return writeBookCopyAudit(
			tx,
			userID,
			"UPDATE",
			fmt.Sprintf(
				"Updated book copy ID %d (previous book ID %d)",
				copyID,
				current.BookID,
			),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return model.BookCopy{}, err
	}

	return result, nil
}

func UpdateBookCopyStatus(
	db *gorm.DB,
	copyID int,
	newStatus string,
	userID int,
	ip string,
	userAgent string,
) (model.BookCopy, error) {
	newStatus = strings.TrimSpace(newStatus)
	if newStatus == "" {
		return model.BookCopy{}, fmt.Errorf(
			"%w: copy status is required",
			ErrInvalidBookCopy,
		)
	}

	var result model.BookCopy

	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := repository.GetBookCopyByID(tx, copyID)
		if err != nil {
			return err
		}

		if err := validateCopyStatus(tx, newStatus); err != nil {
			return err
		}

		if current.CopyStatus == newStatus {
			result = current
			return nil
		}

		if err := repository.UpdateBookCopyStatus(
			tx,
			copyID,
			newStatus,
		); err != nil {
			return err
		}

		result, err = repository.GetBookCopyByID(tx, copyID)
		if err != nil {
			return err
		}

		return writeBookCopyAudit(
			tx,
			userID,
			"UPDATE",
			fmt.Sprintf(
				"Changed book copy ID %d status from %s to %s",
				copyID,
				current.CopyStatus,
				newStatus,
			),
			ip,
			userAgent,
		)
	})

	if err != nil {
		return model.BookCopy{}, err
	}

	return result, nil
}
