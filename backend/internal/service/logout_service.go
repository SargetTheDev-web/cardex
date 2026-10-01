// internal/service/logout_service.go

package service

import (
	"errors"
	"os"
	"strings"

	"backend/internal/repository"
	"backend/pkg/hash"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func Logout(
	db *gorm.DB,
	tokenString string,
	ip string,
	userAgent string,
) error {

	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return errors.New("missing token")
	}

	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {

			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}

			return []byte(os.Getenv("JWT_SECRET")), nil
		},
	)

	if err != nil || !token.Valid {
		return errors.New("invalid token")
	}

	// Extract user ID from JWT
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid token claims")
	}

	userIDValue, ok := claims["user_id"].(float64)
	if !ok {
		return errors.New("invalid user ID in token")
	}

	userID := int(userIDValue)

	// Hash the JWT before looking it up in the database
	tokenHash := hash.HashToken(tokenString)

	// Invalidate the server-side session
	err = repository.DeleteSessionByToken(
		db,
		tokenHash,
	)

	if err != nil {
		return err
	}

	// Get the LOGOUT audit action dynamically
	actionID, err := repository.GetAuditActionID(
		db,
		"LOGOUT",
	)

	if err != nil {
		return err
	}

	// Record logout in audit trail
	err = repository.InsertAuditLog(
		db,
		&userID,
		actionID,
		1, // AUTHENTICATION module
		"User logged out",
		ip,
		userAgent,
	)

	if err != nil {
		return err
	}

	return nil
}
