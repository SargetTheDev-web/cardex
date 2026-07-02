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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	resetToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return err
	}

	resetLink := fmt.Sprintf(
		"http://localhost:3000/reset-password?token=%s",
		resetToken,
	)

	err = repository.InsertAuditLog(
		db,
		&user.UserID,
		3,
		1,
		"Password reset request",
		ip,
		userAgent,
	)
	err = mail.SendResetLink(
		email,
		resetLink,
	)

	if err != nil {
		return err
	}

	return nil
}
