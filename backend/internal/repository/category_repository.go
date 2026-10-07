// internal/repository/category_repository.go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreateCategory(db *gorm.DB, category *model.Category) error {
	return db.Create(category).Error
}

func GetAllCategories(db *gorm.DB) ([]model.Category, error) {
	var categories []model.Category

	err := db.
		Order("category_id ASC").
		Find(&categories).Error

	return categories, err
}

func GetCategoryByID(db *gorm.DB, categoryID int) (model.Category, error) {
	var category model.Category

	err := db.
		Where("category_id = ?", categoryID).
		First(&category).Error

	return category, err
}

func UpdateCategory(
	db *gorm.DB,
	categoryID int,
	name string,
	description string,
) error {
	return db.
		Model(&model.Category{}).
		Where("category_id = ?", categoryID).
		Updates(map[string]interface{}{
			"category_name": name,
			"description":   description,
		}).Error
}

func DeleteCategory(db *gorm.DB, categoryID int) error {
	return db.
		Where("category_id = ?", categoryID).
		Delete(&model.Category{}).Error
}
