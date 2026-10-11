// internal/service/author_service.go

package service

import (
	"errors"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var ErrInvalidAuthor = errors.New("invalid author")

func writeAuthorAudit(
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

func validateAuthor(name, biography string) (string, string, error) {
	name = strings.TrimSpace(name)
	biography = strings.TrimSpace(biography)

	if name == "" {
		return "", "", fmt.Errorf("%w: author name is required", ErrInvalidAuthor)
	}

	if len(name) > 150 {
		return "", "", fmt.Errorf("%w: author name must not exceed 150 characters", ErrInvalidAuthor)
	}

	return name, biography, nil
}

func CreateAuthor(
	db *gorm.DB,
	name string,
	biography string,
	userID int,
	ip string,
	userAgent string,
) (*model.Author, error) {
	name, biography, err := validateAuthor(name, biography)
	if err != nil {
		return nil, err
	}

	author := &model.Author{
		AuthorName: name,
		Biography:  biography,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := repository.CreateAuthor(tx, author); err != nil {
			return err
		}

		return writeAuthorAudit(
			tx,
			userID,
			"CREATE",
			fmt.Sprintf("Created author: %s (ID: %d)", author.AuthorName, author.AuthorID),
			ip,
			userAgent,
		)
	})
	if err != nil {
		return nil, err
	}

	return author, nil
}

func GetAllAuthors(
	db *gorm.DB,
	userID int,
	ip string,
	userAgent string,
) ([]model.Author, error) {
	var authors []model.Author

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		authors, err = repository.GetAllAuthors(tx)
		if err != nil {
			return err
		}

		return writeAuthorAudit(
			tx,
			userID,
			"READ",
			"Retrieved all authors",
			ip,
			userAgent,
		)
	})
	if err != nil {
		return nil, err
	}

	return authors, nil
}

func GetAuthorByID(
	db *gorm.DB,
	authorID int,
	userID int,
	ip string,
	userAgent string,
) (model.Author, error) {
	var author model.Author

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		author, err = repository.GetAuthorByID(tx, authorID)
		if err != nil {
			return err
		}

		return writeAuthorAudit(
			tx,
			userID,
			"READ",
			fmt.Sprintf("Retrieved author ID: %d", authorID),
			ip,
			userAgent,
		)
	})
	if err != nil {
		return model.Author{}, err
	}

	return author, nil
}

func UpdateAuthor(
	db *gorm.DB,
	authorID int,
	name string,
	biography string,
	userID int,
	ip string,
	userAgent string,
) (model.Author, error) {
	name, biography, err := validateAuthor(name, biography)
	if err != nil {
		return model.Author{}, err
	}

	var author model.Author

	err = db.Transaction(func(tx *gorm.DB) error {
		oldAuthor, err := repository.GetAuthorByID(tx, authorID)
		if err != nil {
			return err
		}

		if err := repository.UpdateAuthor(tx, authorID, name, biography); err != nil {
			return err
		}

		author, err = repository.GetAuthorByID(tx, authorID)
		if err != nil {
			return err
		}

		description := fmt.Sprintf(
			"Updated author ID %d: name '%s' to '%s'",
			authorID,
			oldAuthor.AuthorName,
			author.AuthorName,
		)

		return writeAuthorAudit(tx, userID, "UPDATE", description, ip, userAgent)
	})
	if err != nil {
		return model.Author{}, err
	}

	return author, nil
}

func DeleteAuthor(
	db *gorm.DB,
	authorID int,
	userID int,
	ip string,
	userAgent string,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		author, err := repository.GetAuthorByID(tx, authorID)
		if err != nil {
			return err
		}

		if err := repository.DeleteAuthor(tx, authorID); err != nil {
			return err
		}

		description := fmt.Sprintf(
			"Deleted author: %s (ID: %d)",
			author.AuthorName,
			author.AuthorID,
		)

		return writeAuthorAudit(tx, userID, "DELETE", description, ip, userAgent)
	})
}
