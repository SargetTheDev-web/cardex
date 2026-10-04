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
	Token            string
	TokenExpiresAt   time.Time
	User             *model.User
	Role             *model.UserRole
	Permissions      []string
	MaxLoginAttempts int
}

type LoginError struct {
	Message               string
	LoginRetryCount       int
	MaxLoginAttempts      int
	RemainingLoginRetries int
	AccountLocked         bool
}

func (e *LoginError) Error() string {
	return e.Message
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

	// --------------------------------------------------
	// 1. Get user
	// --------------------------------------------------

	user, err := repository.GetUserByIdentifier(db, identifier)
	if err != nil {
		// Keep this generic so we don't reveal whether
		// the account exists.
		return nil, errors.New("invalid credentials")
	}

	// --------------------------------------------------
	// 2. Get maximum login retry count
	// --------------------------------------------------

	maxAttemptsStr, _ := repository.GetSystemParameter(
		db,
		"MAX_LOGIN_RETRY_COUNT",
	)

	maxAttempts, err := strconv.Atoi(maxAttemptsStr)
	if err != nil || maxAttempts <= 0 {
		return nil, errors.New("invalid login retry configuration")
	}

	// --------------------------------------------------
	// 3. Handle temporary account lock
	// --------------------------------------------------

	if user.StatusID == 5 {

		lockDurationStr, _ := repository.GetSystemParameter(
			db,
			"SECURITY_LOCKDOWN_MINS",
		)

		lockDuration, err := strconv.Atoi(lockDurationStr)
		if err != nil || lockDuration <= 0 {
			return nil, errors.New(
				"invalid security lockdown configuration",
			)
		}

		if user.DateTimeLocked != nil {

			unlockTime := user.DateTimeLocked.Add(
				time.Duration(lockDuration) * time.Minute,
			)

			if time.Now().UTC().After(unlockTime.UTC()) {

				if err := repository.ResetLoginAttempts(
					db,
					user.UserID,
				); err != nil {
					return nil, err
				}

				// Synchronize the in-memory user.
				user.StatusID = 3
				user.LoginRetryCount = 0
				user.DateTimeLocked = nil

			} else {
				return nil, errors.New("account still locked")
			}
		}
	}

	// --------------------------------------------------
	// 4. Check account status
	// --------------------------------------------------

	status, err := repository.GetUserStatus(
		db,
		user.StatusID,
	)

	if err != nil {
		return nil, errors.New(
			"failed to retrieve account status",
		)
	}

	if !status.IsAllowedLogin {
		return nil, errors.New(
			"account login not allowed",
		)
	}

	// --------------------------------------------------
	// 5. Check account expiration
	// --------------------------------------------------

	if !user.ExpirationDate.IsZero() &&
		time.Now().UTC().After(user.ExpirationDate.UTC()) {

		return nil, errors.New("account expired")
	}

	// --------------------------------------------------
	// 6. Verify password
	// --------------------------------------------------

	if err := hash.CheckPassword(
		user.PasswordHash,
		password,
	); err != nil {

		// Increment failed login attempts in the database.
		if err := repository.IncrementLoginAttempts(
			db,
			user.UserID,
		); err != nil {
			return nil, err
		}

		// The database count has already been incremented,
		// so calculate the new count from the previous
		// in-memory value.
		currentRetryCount := user.LoginRetryCount + 1

		// Audit failed login.
		_ = repository.InsertAuditLog(
			db,
			&user.UserID,
			6,
			1,
			"Failed login attempt",
			ip,
			userAgent,
		)

		// --------------------------------------------------
		// Maximum retries reached
		// --------------------------------------------------

		if currentRetryCount >= maxAttempts {

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

			return nil, &LoginError{
				Message:               "invalid credentials",
				LoginRetryCount:       currentRetryCount,
				MaxLoginAttempts:      maxAttempts,
				RemainingLoginRetries: 0,
				AccountLocked:         true,
			}
		}

		// --------------------------------------------------
		// Failed login, retries remaining
		// --------------------------------------------------

		return nil, &LoginError{
			Message:               "invalid credentials",
			LoginRetryCount:       currentRetryCount,
			MaxLoginAttempts:      maxAttempts,
			RemainingLoginRetries: maxAttempts - currentRetryCount,
			AccountLocked:         false,
		}
	}

	// --------------------------------------------------
	// 7. Successful password verification
	// --------------------------------------------------

	if err := repository.ResetLoginAttempts(
		db,
		user.UserID,
	); err != nil {
		return nil, err
	}

	// Keep in-memory representation synchronized.
	user.LoginRetryCount = 0
	user.StatusID = 3
	user.DateTimeLocked = nil

	// --------------------------------------------------
	// 8. Generate JWT
	// --------------------------------------------------

	jwtToken, tokenExpiresAt, err := token.GenerateJWT(
		user.UserID,
	)

	if err != nil {
		return nil, err
	}

	// --------------------------------------------------
	// 9. Hash JWT for server-side session storage
	// --------------------------------------------------

	hashedToken := hash.HashToken(jwtToken)

	// Use the exact same expiration timestamp as
	// the JWT.
	sessionExpiry := tokenExpiresAt

	if err := repository.CreateSession(
		db,
		user.UserID,
		hashedToken,
		ip,
		userAgent,
		sessionExpiry,
	); err != nil {
		return nil, err
	}

	// --------------------------------------------------
	// 10. Audit successful login
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
	// 11. Get user's role
	// --------------------------------------------------

	role, err := repository.GetUserRole(
		db,
		user.UserID,
	)

	if err != nil {
		return nil, errors.New(
			"failed to retrieve user role",
		)
	}

	// --------------------------------------------------
	// 12. Get role permissions
	// --------------------------------------------------

	permissions, err := repository.GetRolePermissions(
		db,
		role.RoleID,
	)

	if err != nil {
		return nil, errors.New(
			"failed to retrieve permissions",
		)
	}

	// --------------------------------------------------
	// 13. Return successful login result
	// --------------------------------------------------

	return &LoginResult{
		Token:            jwtToken,
		TokenExpiresAt:   tokenExpiresAt,
		User:             user,
		Role:             role,
		Permissions:      permissions,
		MaxLoginAttempts: maxAttempts,
	}, nil
}
