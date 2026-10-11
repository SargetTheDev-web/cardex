// internal/repository/publisher_repository.go

package repository

import (
	model "backend/internal/models"

	"gorm.io/gorm"
)

func CreatePublisher(db *gorm.DB, publisher *model.Publisher) error {
	return db.Create(publisher).Error
}

func GetAllPublishers(db *gorm.DB) ([]model.Publisher, error) {
	var publishers []model.Publisher

	err := db.Order("publisher_id ASC").Find(&publishers).Error
	return publishers, err
}

func GetPublisherByID(db *gorm.DB, publisherID int) (model.Publisher, error) {
	var publisher model.Publisher

	err := db.First(&publisher, "publisher_id = ?", publisherID).Error
	return publisher, err
}

func UpdatePublisher(
	db *gorm.DB,
	publisherID int,
	name string,
	address string,
	contactNumber string,
) error {
	result := db.Model(&model.Publisher{}).
		Where("publisher_id = ?", publisherID).
		Updates(map[string]interface{}{
			"publisher_name": name,
			"address":        address,
			"contact_number": contactNumber,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func DeletePublisher(db *gorm.DB, publisherID int) error {
	result := db.Delete(&model.Publisher{}, publisherID)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
