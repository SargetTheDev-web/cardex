// internal/models/book.go

package model

import "time"

type Book struct {
	BookID          int       `json:"book_id" gorm:"column:book_id;primaryKey"`
	CategoryID      *int      `json:"category_id" gorm:"column:category_id"`
	ISBN            *string   `json:"isbn" gorm:"column:isbn"`
	Title           string    `json:"title" gorm:"column:title;not null"`
	Edition         *string   `json:"edition" gorm:"column:edition"`
	PublisherID     *int      `json:"publisher_id" gorm:"column:publisher_id"`
	PublicationYear *int      `json:"publication_year" gorm:"column:publication_year"`
	Language        string    `json:"language" gorm:"column:language;not null"`
	Description     *string   `json:"description" gorm:"column:description"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Not a database column. Populated by the repository.
	AuthorIDs []int `json:"author_ids" gorm:"-"`
}

func (Book) TableName() string {
	return "book"
}
