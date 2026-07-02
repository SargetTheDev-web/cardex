package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"backend/internal/repository"
	"backend/pkg/hash"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func ResetPassword(
	db *gorm.DB,
	tokenString string,
	password string,
	confirmPassword string,
	ip string,
	userAgent string,
) error {

	password = strings.TrimSpace(password)
	confirmPassword = strings.TrimSpace(confirmPassword)

	if password != confirmPassword {
		return errors.New("passwords do not match")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return errors.New("invalid or expired reset token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid token claims")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return errors.New("invalid token user")
	}

	tokenPasswordSig, ok := claims["pwd_sig"].(string)
	if !ok {
		return errors.New("invalid token signature")
	}

	user, err := repository.GetUserByID(
		db,
		int(userIDFloat),
	)

	if err != nil {
		return errors.New("user not found")
	}

	currentHashBytes := sha256.Sum256([]byte(user.PasswordHash))
	currentPasswordSig := hex.EncodeToString(currentHashBytes[:])

	if currentPasswordSig != tokenPasswordSig {
		return errors.New("reset token invalidated by password change")
	}

	history, err := repository.GetRecentPasswordHistory(
		db,
		user.UserID,
		5,
	)

	if err != nil {
		return err
	}

	if hash.CheckPassword(user.PasswordHash, password) == nil {
		return errors.New("new password cannot be your current password")
	}

	for _, oldPassword := range history {
		if hash.CheckPassword(oldPassword.PasswordHash, password) == nil {
			return errors.New("password was already used before")
		}
	}

	err = repository.CreatePasswordHistory(
		db,
		user.UserID,
		user.PasswordHash,
	)

	if err != nil {
		return err
	}

	newHash, err := hash.HashPassword(password)
	if err != nil {
		return err
	}

	err = repository.UpdatePassword(
		db,
		user.UserID,
		newHash,
	)

	if err != nil {
		return err
	}

	err = repository.InsertAuditLog(
		db,
		&user.UserID,
		3,
		1,
		"Password reset completed",
		ip,
		userAgent,
	)

	if err != nil {
		return err
	}

	return nil
}
