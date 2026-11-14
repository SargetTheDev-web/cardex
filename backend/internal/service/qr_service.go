// internal/service/qr_service.go

package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrQRAlreadyExists = errors.New("active QR code already exists")
	ErrQRNotFound      = errors.New("active QR code not found")
	ErrInvalidQR       = errors.New("invalid or revoked QR code")
)

const QRPrefix = "CARDEX:"

func generateQRToken() (string, error) {
	token := make([]byte, 32)

	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

func hashQRToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func createQRCode(
	tx *gorm.DB,
	userID int,
) (*model.QRCodeGenerateResponse, error) {

	token, err := generateQRToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	qr := model.UserQRCode{
		UserID:    userID,
		TokenHash: hashQRToken(token),
		Status:    "ACTIVE",
		CreatedAt: now,
	}

	if err := repository.CreateQRCode(tx, &qr); err != nil {
		return nil, err
	}

	return &model.QRCodeGenerateResponse{
		QRID:      qr.QRID,
		Payload:   QRPrefix + token,
		Status:    qr.Status,
		CreatedAt: qr.CreatedAt,
	}, nil
}

func GenerateQRCode(
	db *gorm.DB,
	userID int,
) (*model.QRCodeGenerateResponse, error) {

	var response *model.QRCodeGenerateResponse

	err := db.Transaction(func(tx *gorm.DB) error {

		_, err := repository.GetActiveQRCode(tx, userID)

		if err == nil {
			return ErrQRAlreadyExists
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var createErr error

		response, createErr = createQRCode(tx, userID)

		return createErr
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func RegenerateQRCode(
	db *gorm.DB,
	userID int,
) (*model.QRCodeGenerateResponse, error) {

	var response *model.QRCodeGenerateResponse

	err := db.Transaction(func(tx *gorm.DB) error {

		oldQR, err := repository.LockActiveQRCode(
			tx,
			userID,
		)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQRNotFound
		}

		if err != nil {
			return err
		}

		if err := repository.RevokeQRCode(
			tx,
			oldQR.QRID,
		); err != nil {
			return err
		}

		response, err = createQRCode(tx, userID)

		return err
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

func GetQRCode(
	db *gorm.DB,
	userID int,
) (*model.QRCodeResponse, error) {

	return repository.GetLatestQRCode(
		db,
		userID,
	)
}

func RevokeQRCodeByUser(
	db *gorm.DB,
	userID int,
) error {

	return db.Transaction(func(tx *gorm.DB) error {

		qr, err := repository.LockActiveQRCode(
			tx,
			userID,
		)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQRNotFound
		}

		if err != nil {
			return err
		}

		return repository.RevokeQRCode(
			tx,
			qr.QRID,
		)
	})
}

// Resolves a QR payload to its owner.
// Used by both QR Login and QR Attendance.
func ResolveQRCode(
	db *gorm.DB,
	payload string,
) (int, error) {

	payload = strings.TrimSpace(payload)

	if !strings.HasPrefix(payload, QRPrefix) {
		return 0, ErrInvalidQR
	}

	token := strings.TrimPrefix(payload, QRPrefix)

	decoded, err := base64.RawURLEncoding.DecodeString(token)

	if err != nil || len(decoded) != 32 {
		return 0, ErrInvalidQR
	}

	qr, err := repository.GetQRCodeByHash(
		db,
		hashQRToken(token),
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrInvalidQR
		}

		return 0, err
	}

	if qr.ExpiresAt != nil &&
		!time.Now().Before(*qr.ExpiresAt) {
		return 0, ErrInvalidQR
	}

	return qr.UserID, nil
}
