// internal/service/auth_service.go

package service

import (
	model "backend/internal/models"
	"errors"
	"strconv"
	"strings"
	"time"

	"backend/internal/repository"
	"backend/pkg/hash"
	"backend/pkg/token"

	"gorm.io/gorm"
)

type LoginResult struct {
	Token       string
	User        *model.User
	Role        *model.UserRole
	Permissions []string
}

func Login(
	db *gorm.DB,
	identifier string,
	password string,
	ip string,
	userAgent string,
) (*LoginResult, error) {

	identifier = strings.TrimSpace(identifier)
	password = strings.TrimSpace(password)

	user, err := repository.GetUserByIdentifier(db, identifier)
	if err != nil {
		return nil, errors.New("invalid credentials")
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
					return nil, err
				}

				user.StatusID = 3
				user.LoginRetryCount = 0
			} else {
				return nil, errors.New("account still locked")
			}
		}
	}

	switch user.StatusID {
	case 1:
		return nil, errors.New("account inactive")
	case 2:
		return nil, errors.New("account pending approval")
	case 5:
		return nil, errors.New("account locked")
	case 8:
		return nil, errors.New("account suspended")
	}

	if time.Now().After(user.ExpirationDate) {
		return nil, errors.New("account expired")
	}

	err = hash.CheckPassword(user.PasswordHash, password)

	if err != nil {
		if err := repository.IncrementLoginAttempts(
			db,
			user.UserID,
		); err != nil {
			return nil, err
		}

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
			_ = repository.LockAccount(
				db,
				user.UserID,
			)

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

		return nil, errors.New("invalid credentials")
	}

	_ = repository.ResetLoginAttempts(
		db,
		user.UserID,
	)

	// Generate JWT
	jwtToken, err := token.GenerateJWT(user.UserID)
	if err != nil {
		return nil, err
	}

	// Hash JWT for server-side session storage
	hashedToken := hash.HashToken(jwtToken)

	sessionExpiry := time.Now().Add(24 * time.Hour)

	err = repository.CreateSession(
		db,
		user.UserID,
		hashedToken,
		ip,
		userAgent,
		sessionExpiry,
	)

	if err != nil {
		return nil, err
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

	// Get user's role
	role, err := repository.GetUserRole(
		db,
		user.UserID,
	)

	if err != nil {
		return nil, errors.New("failed to retrieve user role")
	}

	// Get role permissions
	permissions, err := repository.GetRolePermissions(
		db,
		role.RoleID,
	)

	if err != nil {
		return nil, errors.New("failed to retrieve permissions")
	}

	return &LoginResult{
		Token:       jwtToken,
		User:        user,
		Role:        role,
		Permissions: permissions,
	}, nil
}
