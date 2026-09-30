package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"backend/internal/mail"
	"backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func RequestPasswordReset(
	db *gorm.DB,
	email string,
	ip string,
	userAgent string,
) error {

	email = strings.TrimSpace(email)

	user, err := repository.GetUserByIdentifier(db, email)
	if err != nil {
		return errors.New("user not found")
	}

	hashBytes := sha256.Sum256([]byte(user.PasswordHash))
	passwordSignature := hex.EncodeToString(hashBytes[:])

	claims := jwt.MapClaims{
		"user_id": user.UserID,
		"pwd_sig": passwordSignature,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		return errors.New("JWT_SECRET is not configured")
	}

	resetToken, err := token.SignedString(
		[]byte(jwtSecret),
	)

	if err != nil {
		return err
	}

	/*
		PASSWORD_RESET_URL should point to the Go API.

		Local:
		http://localhost:8080/auth/reset-password

		Production:
		https://cardex-api-ltzc.onrender.com/auth/reset-password
	*/

	resetBaseURL := strings.TrimRight(
		os.Getenv("PASSWORD_RESET_URL"),
		"/",
	)

	if resetBaseURL == "" {
		return errors.New("PASSWORD_RESET_URL is not configured")
	}

	resetLink := fmt.Sprintf(
		"%s?token=%s",
		resetBaseURL,
		resetToken,
	)

	/*
		Audit the request.
	*/
	if err := repository.InsertAuditLog(
		db,
		&user.UserID,
		3,
		1,
		"Password reset request",
		ip,
		userAgent,
	); err != nil {
		return err
	}

	/*
		Send the reset email.
	*/
	if err := mail.SendResetLink(
		email,
		resetLink,
	); err != nil {
		return err
	}

	return nil
}
