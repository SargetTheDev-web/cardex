// internal/service/attendance_kiosk_service.go

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

const (
	DefaultAttendanceCampus = "CMDI_MAIN"
	MaxKioskSessionDuration = 16 * time.Hour
)

type AttendanceKioskActivation struct {
	Credential string    `json:"credential"`
	SessionID  int64     `json:"session_id"`
	Campus     string    `json:"campus"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// ActivateAttendanceKiosk creates a limited kiosk credential.
// The raw credential is returned once and must not be logged.
func ActivateAttendanceKiosk(
	db *gorm.DB,
	activatedBy int,
	campus string,
	duration time.Duration,
) (*AttendanceKioskActivation, error) {
	if activatedBy <= 0 {
		return nil, errors.New("invalid activating user")
	}

	campus = strings.TrimSpace(campus)
	if campus == "" {
		campus = DefaultAttendanceCampus
	}
	if len(campus) > 50 {
		return nil, errors.New("campus must not exceed 50 characters")
	}

	if duration <= 0 || duration > MaxKioskSessionDuration {
		return nil, errors.New("invalid kiosk session duration")
	}

	// Generate 32 cryptographically secure random bytes.
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, errors.New("failed to generate kiosk credential")
	}

	credential := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Store only the SHA-256 hash, never the raw credential.
	hashBytes := sha256.Sum256([]byte(credential))
	credentialHash := hex.EncodeToString(hashBytes[:])

	now := time.Now().UTC()
	session := &model.AttendanceKioskSession{
		CredentialHash: credentialHash,
		Campus:         campus,
		ActivatedBy:    activatedBy,
		CreatedAt:      now,
		ExpiresAt:      now.Add(duration),
	}

	if err := repository.CreateAttendanceKioskSession(db, session); err != nil {
		return nil, errors.New("failed to activate attendance kiosk")
	}

	return &AttendanceKioskActivation{
		Credential: credential,
		SessionID:  session.KioskSessionID,
		Campus:     session.Campus,
		ExpiresAt:  session.ExpiresAt,
	}, nil
}

// ValidateAttendanceKioskCredential verifies a kiosk credential.
// The returned record contains the stored hash, not the raw credential.
func ValidateAttendanceKioskCredential(
	db *gorm.DB,
	credential string,
) (*model.AttendanceKioskSession, error) {
	if len(credential) < 40 || len(credential) > 100 {
		return nil, gorm.ErrRecordNotFound
	}

	hashBytes := sha256.Sum256([]byte(credential))
	credentialHash := hex.EncodeToString(hashBytes[:])

	session, err := repository.GetActiveAttendanceKioskSession(
		db,
		credentialHash,
	)
	if err != nil {
		return nil, err
	}

	if err := repository.UpdateAttendanceKioskLastUsed(
		db,
		session.KioskSessionID,
	); err != nil {
		return nil, errors.New("failed to update kiosk activity")
	}

	return session, nil
}

func DeactivateAttendanceKiosk(
	db *gorm.DB,
	sessionID int64,
) error {
	if sessionID <= 0 {
		return errors.New("invalid kiosk session ID")
	}

	return repository.RevokeAttendanceKioskSession(db, sessionID)
}
