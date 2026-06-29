package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"backend/internal/repository"
	"backend/pkg/hash"
	"backend/pkg/token"

	"gorm.io/gorm"
)

func Login(
	db *gorm.DB,
	identifier string,
	password string,
	ip string,
	userAgent string,
) (string, error) {

	identifier = strings.TrimSpace(identifier)
	password = strings.TrimSpace(password)

	user, err := repository.GetUserByIdentifier(db, identifier)
	if err != nil {
		return "", errors.New("user not found")
	}

	maxAttemptsStr, _ := repository.GetSystemParameter(
		db,
		"MAX_LOGIN_RETRY_COUNT",
	)

	maxAttempts, _ := strconv.Atoi(maxAttemptsStr)

	if user.StatusID == 5 {
		lockDurationStr, _ := repository.GetSystemParameter(
			db,
			"SECURITY_LOCKDOWN_MINS",
		)

		lockDuration, _ := strconv.Atoi(lockDurationStr)

		if user.DateTimeLocked != nil {
			unlockTime := user.DateTimeLocked.Add(
				time.Duration(lockDuration) * time.Minute,
			)

			if time.Now().UTC().After(unlockTime.UTC()) {
				err := repository.ResetLoginAttempts(db, user.UserID)
				if err != nil {
					return "", err
				}

				user.StatusID = 3
				user.LoginRetryCount = 0
			} else {
				return "", errors.New("account still locked")
			}
		}
	}

	switch user.StatusID {
	case 1:
		return "", errors.New("account inactive")
	case 2:
		return "", errors.New("account pending approval")
	case 5:
		return "", errors.New("account locked")
	case 8:
		return "", errors.New("account suspended")
	}

	if time.Now().After(user.ExpirationDate) {
		return "", errors.New("account expired")
	}

	err = hash.CheckPassword(user.PasswordHash, password)

	if err != nil {
		_ = repository.IncrementLoginAttempts(db, user.UserID)

		_ = repository.InsertAuditLog(
			db,
			&user.UserID,
			6,
			1,
			"Failed login attempt",
			ip,
			userAgent,
		)

		if user.LoginRetryCount+1 >= maxAttempts {
			_ = repository.LockAccount(db, user.UserID)

			_ = repository.InsertAuditLog(
				db,
				&user.UserID,
				7,
				1,
				"Account locked due to max failed attempts",
				ip,
				userAgent,
			)
		}

		return "", errors.New("invalid credentials")
	}

	_ = repository.ResetLoginAttempts(db, user.UserID)

	jwtToken, err := token.GenerateJWT(user.UserID)
	if err != nil {
		return "", err
	}

	_ = repository.InsertAuditLog(
		db,
		&user.UserID,
		5,
		1,
		"Successful login",
		ip,
		userAgent,
	)

	return jwtToken, nil
}
