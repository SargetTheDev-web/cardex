// internal/models/reservation.go

package model

import "time"

type Reservation struct {
	ReservationID       int64      `json:"reservation_id" gorm:"column:reservation_id;primaryKey"`
	BookID              int        `json:"book_id" gorm:"column:book_id;not null"`
	UserID              int        `json:"user_id" gorm:"column:user_id;not null"`
	CopyID              *int       `json:"copy_id" gorm:"column:copy_id"`
	ReservationDate     time.Time  `json:"reservation_date" gorm:"column:reservation_date;autoCreateTime"`
	PreferredPickupDate time.Time  `json:"preferred_pickup_date" gorm:"column:preferred_pickup_date;type:date;not null"`
	Status              string     `json:"status" gorm:"column:status;not null"`
	Reason              *string    `json:"reason" gorm:"column:reason"`
	QueuePosition       *int       `json:"queue_position" gorm:"column:queue_position"`
	ExpiryDate          *time.Time `json:"expiry_date" gorm:"column:expiry_date"`
	CreatedAt           time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime:false"`
}

func (Reservation) TableName() string {
	return "reservation"
}
