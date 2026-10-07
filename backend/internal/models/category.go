// internal/models/category.go

package model

type Category struct {
	CategoryID   int    `json:"category_id" gorm:"column:category_id;primaryKey"`
	CategoryName string `json:"category_name" gorm:"column:category_name;unique;not null"`
	Description  string `json:"description" gorm:"column:description"`
}

func (Category) TableName() string {
	return "category"
}
