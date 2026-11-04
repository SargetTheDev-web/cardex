package service

import (
	"errors"
	"fmt"
	"time"
	_ "time/tzdata"

	model "backend/internal/models"
	"backend/internal/repository"

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

			// Alert active admins, librarians, and staff if the attendee
			// has one or more overdue loans.
			var overdueCount int64
			if err := tx.
				Table("loan").
				Where(
					"user_id = ? AND status = ? AND due_date < CURRENT_DATE AND returned_at IS NULL",
					userID,
					"BORROWED",
				).
				Count(&overdueCount).Error; err != nil {
				return fmt.Errorf("check attendee overdue loans: %w", err)
			}

			if overdueCount > 0 {
				var staffIDs []int
				if err := tx.
					Table(`"user"`).
					Where("role_id IN ? AND status_id = ?", []int{1, 2, 3}, 3).
					Pluck("user_id", &staffIDs).Error; err != nil {
					return fmt.Errorf("load active staff recipients: %w", err)
				}

				message := fmt.Sprintf(
					"User ID %d checked in and has %d overdue loan(s).",
					userID,
					overdueCount,
				)

				for _, staffID := range staffIDs {
					notification := model.Notification{
						UserID:  staffID,
						Type:    "ATTENDANCE",
						Channel: repository.NotificationChannelInApp,
						Subject: "Attendee Has Overdue Loans",
						Message: message,
						IsRead:  false,
					}

					if err := repository.CreateNotification(tx, &notification); err != nil {
						return fmt.Errorf("notify staff about overdue attendee: %w", err)
					}
				}
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
