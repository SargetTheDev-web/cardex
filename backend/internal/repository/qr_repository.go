// internal/repository/qr_repository.go

package repository

import (
	model "backend/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetActiveQRCode(
	db *gorm.DB,
	userID int,
) (*model.UserQRCode, error) {

	var qr model.UserQRCode

	err := db.
		Where("user_id = ? AND status = ?", userID, "ACTIVE").
		First(&qr).Error

	if err != nil {
		return nil, err
	}

	return &qr, nil
}

func GetLatestQRCode(
	db *gorm.DB,
	userID int,
) (*model.QRCodeResponse, error) {

	var response model.QRCodeResponse

	err := db.
		Table("user_qr_code AS qr").
		Select(`
			qr.qr_id,
			qr.user_id,

			COALESCE(
				(
					SELECT CONCAT_WS(
						' ',
						NULLIF(BTRIM(up.first_name), ''),
						NULLIF(BTRIM(up.middle_name), ''),
						NULLIF(BTRIM(up.last_name), ''),
						NULLIF(BTRIM(up.suffix_extension), '')
					)
					FROM user_profile up
					WHERE up.user_id = qr.user_id
					ORDER BY up.profile_id DESC
					LIMIT 1
				),
				''
			) AS full_name,

			qr.status,
			qr.created_at,
			qr.expires_at,
			qr.revoked_at
		`).
		Where("qr.user_id = ?", userID).
		Order("qr.created_at DESC, qr.qr_id DESC").
		Limit(1).
		Scan(&response).Error

	if err != nil {
		return nil, err
	}

	if response.QRID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return &response, nil
}

func GetQRCodeByHash(
	db *gorm.DB,
	tokenHash string,
) (*model.UserQRCode, error) {

	var qr model.UserQRCode

	err := db.
		Where(
			"token_hash = ? AND status = ?",
			tokenHash,
			"ACTIVE",
		).
		First(&qr).Error

	if err != nil {
		return nil, err
	}

	return &qr, nil
}

func LockActiveQRCode(
	tx *gorm.DB,
	userID int,
) (*model.UserQRCode, error) {

	var qr model.UserQRCode

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"user_id = ? AND status = ?",
			userID,
			"ACTIVE",
		).
		First(&qr).Error

	if err != nil {
		return nil, err
	}

	return &qr, nil
}

func CreateQRCode(
	tx *gorm.DB,
	qr *model.UserQRCode,
) error {

	return tx.Create(qr).Error
}

func RevokeQRCode(
	tx *gorm.DB,
	qrID int64,
) error {

	now := time.Now().UTC()

	result := tx.
		Model(&model.UserQRCode{}).
		Where(
			"qr_id = ? AND status = ?",
			qrID,
			"ACTIVE",
		).
		Updates(map[string]interface{}{
			"status":     "REVOKED",
			"revoked_at": now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return errors.New("QR code is no longer active")
	}

	return nil
}
