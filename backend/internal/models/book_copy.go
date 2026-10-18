// internal/models/book_copy.go

package model

import "time"

type BookCopy struct {
	CopyID        int       `json:"copy_id" gorm:"column:copy_id;primaryKey"`
	BookID        int       `json:"book_id" gorm:"column:book_id;not null"`
	Barcode       *string   `json:"barcode" gorm:"column:barcode"`
	ShelfLocation *string   `json:"shelf_location" gorm:"column:shelf_location"`
	CopyStatus    string    `json:"copy_status" gorm:"column:copy_status;not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime:false"`
}

func (BookCopy) TableName() string {
	return "book_copy"
}
