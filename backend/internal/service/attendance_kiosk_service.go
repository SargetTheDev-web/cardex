// internal/service/attendance_kiosk_service.go

package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
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
	ipAddress string,
	userAgent string,
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

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, errors.New("failed to generate kiosk credential")
	}

	credential := base64.RawURLEncoding.EncodeToString(randomBytes)
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

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := repository.CreateAttendanceKioskSession(tx, session); err != nil {
			return err
		}

		actionID, err := repository.GetAuditActionID(tx, "CREATE")
		if err != nil {
			return err
		}
		moduleID, err := repository.GetAuditModuleID(tx, "ATTENDANCE")
		if err != nil {
			return err
		}

		actorID := activatedBy
		description := fmt.Sprintf(
			"Attendance kiosk activated: session_id=%d, campus=%s, expires_at=%s",
			session.KioskSessionID,
			session.Campus,
			session.ExpiresAt.Format(time.RFC3339),
		)

		return repository.InsertAuditLog(
			tx,
			&actorID,
			actionID,
			moduleID,
			description,
			ipAddress,
			userAgent,
		)
	})
	if err != nil {
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
	deactivatedBy int,
	ipAddress string,
	userAgent string,
) error {
	if sessionID <= 0 {
		return errors.New("invalid kiosk session ID")
	}
	if deactivatedBy <= 0 {
		return errors.New("invalid deactivating user")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var session model.AttendanceKioskSession
		if err := tx.
			Where("kiosk_session_id = ?", sessionID).
			First(&session).Error; err != nil {
			return err
		}

		if err := repository.RevokeAttendanceKioskSession(tx, sessionID); err != nil {
			return err
		}

		actionID, err := repository.GetAuditActionID(tx, "UPDATE")
		if err != nil {
			return err
		}
		moduleID, err := repository.GetAuditModuleID(tx, "ATTENDANCE")
		if err != nil {
			return err
		}

		actorID := deactivatedBy
		description := fmt.Sprintf(
			"Attendance kiosk deactivated: session_id=%d, campus=%s",
			session.KioskSessionID,
			session.Campus,
		)

		return repository.InsertAuditLog(
			tx,
			&actorID,
			actionID,
			moduleID,
			description,
			ipAddress,
			userAgent,
		)
	})
}
