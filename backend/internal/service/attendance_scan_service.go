package service

import (
	"errors"
	"fmt"
	"time"
	_ "time/tzdata"

	model "backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidKioskCredential  = errors.New("invalid or expired kiosk credential")
	ErrAttendanceUserNotFound  = errors.New("user not found")
	ErrAttendanceUserInactive  = errors.New("user account is not active")
	ErrDuplicateAttendanceScan = errors.New("duplicate scan; please wait before scanning again")
)

type AttendanceScanResult struct {
	UserID         int       `json:"user_id"`
	Campus         string    `json:"campus"`
	ScanType       string    `json:"scan_type"`
	ScannedAt      time.Time `json:"scanned_at"`
	AttendanceDate string    `json:"attendance_date"`
}

// RecordAttendanceScan validates the kiosk and records one attendance event.
// Account status 3 represents an active account in CARDex.
func RecordAttendanceScan(
	db *gorm.DB,
	credential string,
	userID int,
) (*AttendanceScanResult, error) {
	if userID <= 0 {
		return nil, ErrAttendanceUserNotFound
	}

	kiosk, err := ValidateAttendanceKioskCredential(db, credential)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKioskCredential, err)
	}

	loc, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		return nil, fmt.Errorf("load Manila timezone: %w", err)
	}

	now := time.Now().UTC()
	localNow := now.In(loc)
	localDate := localNow.Format("2006-01-02")

	dayStartLocal := time.Date(
		localNow.Year(),
		localNow.Month(),
		localNow.Day(),
		0, 0, 0, 0,
		loc,
	)
	dayEndLocal := dayStartLocal.AddDate(0, 0, 1)

	var scanType string

	err = db.Transaction(func(tx *gorm.DB) error {
		// Serialize scans for this user to avoid concurrent state changes.
		var user model.User
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("user_id", "status_id").
			Where("user_id = ?", userID).
			First(&user).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAttendanceUserNotFound
		}
		if err != nil {
			return fmt.Errorf("load attendance user: %w", err)
		}
		if user.StatusID != 3 {
			return ErrAttendanceUserInactive
		}

		// Find the latest scan for this user, campus, and Manila day.
		var previous model.AttendanceLog
		err = tx.Where(
			"user_id = ? AND campus = ? AND scanned_at >= ? AND scanned_at < ?",
			userID,
			kiosk.Campus,
			dayStartLocal.UTC(),
			dayEndLocal.UTC(),
		).
			Order("scanned_at DESC, id DESC").
			First(&previous).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("load previous attendance scan: %w", err)
		}

		if err == nil {
			if now.Sub(previous.ScannedAt) < 5*time.Second {
				return ErrDuplicateAttendanceScan
			}

			if previous.ScanType == "CHECK_IN" {
				scanType = "CHECK_OUT"
			} else {
				scanType = "CHECK_IN"
			}
		} else {
			scanType = "CHECK_IN"
		}

		sessionID := kiosk.KioskSessionID
		attendanceLog := model.AttendanceLog{
			UserID:         userID,
			Campus:         kiosk.Campus,
			ScanType:       scanType,
			ScannedAt:      now,
			KioskSessionID: &sessionID,
		}

		if err := tx.Create(&attendanceLog).Error; err != nil {
			return fmt.Errorf("insert attendance log: %w", err)
		}

		if scanType == "CHECK_IN" {
			day := model.AttendanceDay{
				UserID:         userID,
				Campus:         kiosk.Campus,
				AttendanceDate: dayStartLocal,
				FirstCheckInAt: now,
			}

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "campus"},
					{Name: "attendance_date"},
				},
				DoNothing: true,
			}).Create(&day).Error; err != nil {
				return fmt.Errorf("insert daily attendance credit: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &AttendanceScanResult{
		UserID:         userID,
		Campus:         kiosk.Campus,
		ScanType:       scanType,
		ScannedAt:      now,
		AttendanceDate: localDate,
	}, nil
}
