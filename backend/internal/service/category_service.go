// internal/service/category_service.go

package service

import (
	"errors"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

func CreateCategory(
	db *gorm.DB,
	name string,
	description string,
) (*model.Category, error) {

	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return nil, errors.New("category name is required")
	}

	if len(name) > 100 {
		return nil, errors.New("category name must not exceed 100 characters")
	}

	if len(description) > 0 {
		if len(description) > 10000 {
			return nil, errors.New("description is too long")
		}
	}

	category := &model.Category{
		CategoryName: name,
		Description:  description,
	}

	if err := repository.CreateCategory(db, category); err != nil {
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
) (model.Category, error) {

	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return model.Category{}, errors.New("category name is required")
	}

	if len(name) > 100 {
		return model.Category{}, errors.New("category name must not exceed 100 characters")
	}

	if err := repository.UpdateCategory(
		db,
		categoryID,
		name,
		description,
	); err != nil {
		return model.Category{}, err
	}

	return repository.GetCategoryByID(db, categoryID)
}

func DeleteCategory(
	db *gorm.DB,
	categoryID int,
) error {

	_, err := repository.GetCategoryByID(db, categoryID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}

		return err
	}

	return repository.DeleteCategory(db, categoryID)
}
