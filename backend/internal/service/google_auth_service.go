package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"backend/internal/repository"
	"backend/pkg/hash"
	"backend/pkg/token"

	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

type GoogleAuthError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *GoogleAuthError) Error() string {
	return e.Message
}

type GoogleAuthResult struct {
	Login                *LoginResult
	RegistrationRequired bool
	Email                string
	Name                 string
	Picture              string
	ProfileImageURL      *string
}

func googleError(httpStatus int, code, message string) error {
	return &GoogleAuthError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

func GoogleAuthenticate(
	db *gorm.DB,
	rawIDToken string,
	ip string,
	userAgent string,
) (*GoogleAuthResult, error) {
	rawIDToken = strings.TrimSpace(rawIDToken)
	if rawIDToken == "" {
		return nil, googleError(
			http.StatusBadRequest,
			"missing_id_token",
			"Google ID token is required",
		)
	}

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return nil, googleError(
			http.StatusInternalServerError,
			"google_not_configured",
			"Google sign-in is not configured",
		)
	}

	payload, err := idtoken.Validate(
		context.Background(),
		rawIDToken,
		clientID,
	)
	if err != nil {
		return nil, googleError(
			http.StatusUnauthorized,
			"invalid_google_token",
			"Google ID token is invalid or expired",
		)
	}

	subject, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	subject = strings.TrimSpace(subject)
	email = strings.ToLower(strings.TrimSpace(email))

	if subject == "" || email == "" || !emailVerified {
		return nil, googleError(
			http.StatusUnauthorized,
			"unverified_google_identity",
			"Google must provide a verified email and valid subject",
		)
	}

	// Find an account explicitly linked to this Google identity.
	user, err := repository.GetUserBySSO(db, "google", subject)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, googleError(
			http.StatusInternalServerError,
			"account_lookup_failed",
			"Unable to look up the Google account",
		)
	}

	if user == nil {
		// Do not automatically link an existing CARDex account by email.
		existingUser, lookupErr := repository.GetUserByIdentifier(db, email)

		if lookupErr == nil && existingUser != nil {
			return nil, googleError(
				http.StatusConflict,
				"account_exists_unlinked",
				"A CARDex account already uses this email. Account linking is required.",
			)
		}

		if lookupErr != nil &&
			!errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return nil, googleError(
				http.StatusInternalServerError,
				"account_lookup_failed",
				"Unable to look up the CARDex account",
			)
		}

		return &GoogleAuthResult{
			RegistrationRequired: true,
			Email:                email,
			Name:                 name,
			Picture:              picture,
		}, nil
	}

	// Do not reactivate inactive, pending, locked, or suspended accounts.
	if user.StatusID != 3 {
		return nil, googleError(
			http.StatusForbidden,
			"account_not_active",
			"This CARDex account is not active",
		)
	}

	now := time.Now()
	if !user.ExpirationDate.IsZero() &&
		now.After(user.ExpirationDate) {
		return nil, googleError(
			http.StatusForbidden,
			"account_expired",
			"This CARDex account has expired",
		)
	}

	jwtToken, jwtExpiresAt, err := token.GenerateJWT(user.UserID)
	if err != nil {
		return nil, googleError(
			http.StatusInternalServerError,
			"token_generation_failed",
			"Unable to create a CARDex session",
		)
	}

	tokenHash := hash.HashToken(jwtToken)

	// Match the existing CARDex session duration.
	sessionExpiry := jwtExpiresAt

	if err := repository.CreateSession(
		db,
		user.UserID,
		tokenHash,
		ip,
		userAgent,
		sessionExpiry,
	); err != nil {
		return nil, googleError(
			http.StatusInternalServerError,
			"session_creation_failed",
			"Unable to create a CARDex session",
		)
	}

	role, err := repository.GetUserRole(db, user.UserID)
	if err != nil {
		_ = repository.DeleteSessionByToken(db, tokenHash)

		return nil, googleError(
			http.StatusInternalServerError,
			"role_lookup_failed",
			"Unable to load the CARDex account role",
		)
	}

	permissions, err := repository.GetRolePermissions(db, role.RoleID)
	if err != nil {
		_ = repository.DeleteSessionByToken(db, tokenHash)

		return nil, googleError(
			http.StatusInternalServerError,
			"permissions_lookup_failed",
			"Unable to load CARDex permissions",
		)
	}

	// Retrieve the profile, including the image supplied during registration.
	// GetUserProfile is expected to include pending profile-request data
	// through the repository join you implemented.
	_, profile, profileErr := repository.GetUserProfile(
		db,
		user.UserID,
	)

	if profileErr != nil &&
		!errors.Is(profileErr, gorm.ErrRecordNotFound) {
		_ = repository.DeleteSessionByToken(db, tokenHash)

		return nil, googleError(
			http.StatusInternalServerError,
			"profile_lookup_failed",
			"Unable to load the CARDex user profile",
		)
	}

	var profileImageURL *string
	if profile != nil {
		profileImageURL = profile.ProfileImageURL
	}

	// Authentication has succeeded. Do not expose internal audit errors.
	_ = repository.InsertAuditLog(
		db,
		&user.UserID,
		5,
		1,
		"Successful Google login",
		ip,
		userAgent,
	)

	return &GoogleAuthResult{
		Login: &LoginResult{
			Token:          jwtToken,
			TokenExpiresAt: sessionExpiry,
			User:           user,
			Role:           role,
			Permissions:    permissions,
		},
		ProfileImageURL: profileImageURL,
	}, nil
}
