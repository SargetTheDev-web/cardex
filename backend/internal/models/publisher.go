// internal/models/publisher.go

package model

type Publisher struct {
	PublisherID   int    `json:"publisher_id" gorm:"column:publisher_id;primaryKey"`
	PublisherName string `json:"publisher_name" gorm:"column:publisher_name;not null"`
	Address       string `json:"address" gorm:"column:address"`
	ContactNumber string `json:"contact_number" gorm:"column:contact_number"`
}

func (Publisher) TableName() string {
	return "publisher"
}
