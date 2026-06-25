package service

import (
	"backend/internal/repository"
	"backend/pkg/hash"
	"backend/pkg/token"
	"errors"

	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func Login(
	conn *pgx.Conn,
	identifier string,
	password string,
	ip string,
	userAgent string,
) (string, error) {

	// Input Sanitization
	identifier = strings.TrimSpace(identifier)
	password = strings.TrimSpace(password)

	/*
		RECAPTCHA LOGIC (DISABLED FOR POSTMAN)

		isHuman := VerifyRecaptcha(recaptchaToken)
		if !isHuman {
			return "", errors.New("captcha verification failed")
		}
	*/

	user, err := repository.GetUserByIdentifier(conn, identifier)
	if err != nil {
		return "", errors.New("user not found")
	}

	maxAttemptsStr, _ := repository.GetSystemParameter(
		conn,
		"MAX_LOGIN_RETRY_COUNT",
	)

	maxAttempts, _ := strconv.Atoi(maxAttemptsStr)

	if user.StatusID == 5 {
		lockDurationStr, err := repository.GetSystemParameter(
			conn,
			"SECURITY_LOCKDOWN_MINS",
		)
		if err != nil {
			return "", err
		}

		lockDuration, err := strconv.Atoi(lockDurationStr)
		if err != nil {
			return "", err
		}

		if user.DateTimeLocked == nil {
			return "", errors.New("account lock timestamp missing")
		}

		canUnlock, err := repository.CanUnlockAccount(
			conn,
			user.DateTimeLocked,
			lockDuration,
		)
		if err != nil {
			return "", err
		}

		if canUnlock {
			err := repository.ResetLoginAttempts(conn, user.UserID)
			if err != nil {
				return "", err
			}

			user.StatusID = 3
			user.LoginRetryCount = 0
		} else {
			return "", errors.New("account still locked")
		}
	}
	// Status Validation
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

	// Password Authentication
	err = hash.CheckPassword(user.PasswordHash, password)

	if err != nil {
		_ = repository.IncrementLoginAttempts(conn, user.UserID)

		_ = repository.InsertAuditLog(
			conn,
			&user.UserID,
			6,
			1,
			"Failed login attempt",
			ip,
			userAgent,
		)

		if user.LoginRetryCount+1 >= maxAttempts {
			_ = repository.LockAccount(conn, user.UserID)

			_ = repository.InsertAuditLog(
				conn,
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

	// Success Reset
	_ = repository.ResetLoginAttempts(conn, user.UserID)

	jwtToken, err := token.GenerateJWT(user.UserID)
	if err != nil {
		return "", err
	}

	_ = repository.InsertAuditLog(
		conn,
		&user.UserID,
		5,
		1,
		"Successful login",
		ip,
		userAgent,
	)

	return jwtToken, nil
}
