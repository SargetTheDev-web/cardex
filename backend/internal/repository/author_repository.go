// internal/repository/author_repository.go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreateAuthor(db *gorm.DB, author *model.Author) error {
	return db.Create(author).Error
}

func GetAllAuthors(db *gorm.DB) ([]model.Author, error) {
	var authors []model.Author

	err := db.Order("author_id ASC").Find(&authors).Error
	return authors, err
}

func GetAuthorByID(db *gorm.DB, authorID int) (model.Author, error) {
	var author model.Author

	err := db.First(&author, "author_id = ?", authorID).Error
	return author, err
}

func UpdateAuthor(
	db *gorm.DB,
	authorID int,
	name string,
	biography string,
) error {
	result := db.Model(&model.Author{}).
		Where("author_id = ?", authorID).
		Updates(map[string]interface{}{
			"author_name": name,
			"biography":   biography,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func DeleteAuthor(db *gorm.DB, authorID int) error {
	result := db.Delete(&model.Author{}, authorID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
