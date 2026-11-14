// internal/models/qr_code.go

package model

import "time"

type UserQRCode struct {
	QRID      int64      `json:"qr_id" gorm:"column:qr_id;primaryKey"`
	UserID    int        `json:"user_id" gorm:"column:user_id;not null"`
	TokenHash string     `json:"-" gorm:"column:token_hash;not null;unique"`
	Status    string     `json:"status" gorm:"column:status;not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" gorm:"column:expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" gorm:"column:revoked_at"`
}

func (UserQRCode) TableName() string {
	return "user_qr_code"
}

// Used by GET /qr/me
type QRCodeResponse struct {
	QRID      int64      `json:"qr_id"`
	UserID    int        `json:"user_id"`
	FullName  string     `json:"full_name"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// Used by POST /qr/generate
// Used by POST /qr/regenerate
type QRCodeGenerateResponse struct {
	QRID      int64     `json:"qr_id"`
	Payload   string    `json:"payload"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
