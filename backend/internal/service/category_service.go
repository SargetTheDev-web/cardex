// internal/service/category_service.go

package service

import (
	"errors"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var ErrInvalidCategory = errors.New("invalid category")

const categoryAuditModule = "CATALOG"

func writeCategoryAudit(
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

	moduleID, err := repository.GetAuditModuleID(tx, categoryAuditModule)
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

func validateCategory(
	name string,
	description string,
) (string, string, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return "", "", fmt.Errorf("%w: category name is required", ErrInvalidCategory)
	}

	if len(name) > 100 {
		return "", "", fmt.Errorf("%w: category name must not exceed 100 characters", ErrInvalidCategory)
	}

	if len(description) > 10000 {
		return "", "", fmt.Errorf("%w: description must not exceed 10000 characters", ErrInvalidCategory)
	}

	return name, description, nil
}

func CreateCategory(
	db *gorm.DB,
	name string,
	description string,
	userID int,
	ip string,
	userAgent string,
) (*model.Category, error) {
	name, description, err := validateCategory(name, description)
	if err != nil {
		return nil, err
	}

	category := &model.Category{
		CategoryName: name,
		Description:  description,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := repository.CreateCategory(tx, category); err != nil {
			return err
		}

		auditDescription := fmt.Sprintf(
			"Created category: %s (ID: %d)",
			category.CategoryName,
			category.CategoryID,
		)

		return writeCategoryAudit(
			tx,
			userID,
			"CREATE",
			auditDescription,
			ip,
			userAgent,
		)
	})
	if err != nil {
		return nil, err
	}

	return category, nil
}

func GetAllCategories(db *gorm.DB) ([]model.Category, error) {
	return repository.GetAllCategories(db)
}

func GetCategoryByID(
	db *gorm.DB,
	categoryID int,
) (model.Category, error) {
	return repository.GetCategoryByID(db, categoryID)
}

func UpdateCategory(
	db *gorm.DB,
	categoryID int,
	name string,
	description string,
	userID int,
	ip string,
	userAgent string,
) (model.Category, error) {
	name, description, err := validateCategory(name, description)
	if err != nil {
		return model.Category{}, err
	}

	var updatedCategory model.Category

	err = db.Transaction(func(tx *gorm.DB) error {
		oldCategory, err := repository.GetCategoryByID(tx, categoryID)
		if err != nil {
			return err
		}

		if err := repository.UpdateCategory(
			tx,
			categoryID,
			name,
			description,
		); err != nil {
			return err
		}

		updatedCategory, err = repository.GetCategoryByID(tx, categoryID)
		if err != nil {
			return err
		}

		auditDescription := fmt.Sprintf(
			"Updated category ID %d: name '%s' to '%s'",
			categoryID,
			oldCategory.CategoryName,
			updatedCategory.CategoryName,
		)

		return writeCategoryAudit(
			tx,
			userID,
			"UPDATE",
			auditDescription,
			ip,
			userAgent,
		)
	})
	if err != nil {
		return model.Category{}, err
	}

	return updatedCategory, nil
}

func DeleteCategory(
	db *gorm.DB,
	categoryID int,
	userID int,
	ip string,
	userAgent string,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		category, err := repository.GetCategoryByID(tx, categoryID)
		if err != nil {
			return err
		}

		if err := repository.DeleteCategory(tx, categoryID); err != nil {
			return err
		}

		auditDescription := fmt.Sprintf(
			"Deleted category: %s (ID: %d)",
			category.CategoryName,
			category.CategoryID,
		)

		return writeCategoryAudit(
			tx,
			userID,
			"DELETE",
			auditDescription,
			ip,
			userAgent,
		)
	})
}
