// internal/models/author.go

package model

type Author struct {
	AuthorID   int    `json:"author_id" gorm:"column:author_id;primaryKey"`
	AuthorName string `json:"author_name" gorm:"column:author_name;not null"`
	Biography  string `json:"biography" gorm:"column:biography"`
}

func (Author) TableName() string {
	return "author"
}
