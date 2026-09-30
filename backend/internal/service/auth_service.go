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

	// --------------------------------------------------
	// 1. Handle temporary account lock
	// --------------------------------------------------

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
				err := repository.ResetLoginAttempts(
					db,
					user.UserID,
				)

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

	// --------------------------------------------------
	// 2. Check account status
	// --------------------------------------------------

	status, err := repository.GetUserStatus(
		db,
		user.StatusID,
	)

	if err != nil {
		return nil, errors.New("failed to retrieve account status")
	}

	if !status.IsAllowedLogin {
		return nil, errors.New("account login not allowed")
	}

	// --------------------------------------------------
	// 3. Check account expiration
	// --------------------------------------------------

	if !user.ExpirationDate.IsZero() &&
		time.Now().After(user.ExpirationDate) {

		return nil, errors.New("account expired")
	}

	// --------------------------------------------------
	// 4. Verify password
	// --------------------------------------------------

	err = hash.CheckPassword(
		user.PasswordHash,
		password,
	)

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

	// --------------------------------------------------
	// 5. Reset login retry counter
	// --------------------------------------------------

	_ = repository.ResetLoginAttempts(
		db,
		user.UserID,
	)

	// --------------------------------------------------
	// 6. Generate JWT
	// --------------------------------------------------

	jwtToken, err := token.GenerateJWT(
		user.UserID,
	)

	if err != nil {
		return nil, err
	}

	// --------------------------------------------------
	// 7. Hash JWT for server-side session storage
	// --------------------------------------------------

	hashedToken := hash.HashToken(jwtToken)

	sessionExpiry := time.Now().Add(
		24 * time.Hour,
	)

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

	// --------------------------------------------------
	// 8. Audit successful login
	// --------------------------------------------------

	_ = repository.InsertAuditLog(
		db,
		&user.UserID,
		5,
		1,
		"Successful login",
		ip,
		userAgent,
	)

	// --------------------------------------------------
	// 9. Get user's role
	// --------------------------------------------------

	role, err := repository.GetUserRole(
		db,
		user.UserID,
	)

	if err != nil {
		return nil, errors.New("failed to retrieve user role")
	}

	// --------------------------------------------------
	// 10. Get role permissions
	// --------------------------------------------------

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
