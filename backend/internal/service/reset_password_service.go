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

	tokenString = strings.TrimSpace(tokenString)
	password = strings.TrimSpace(password)
	confirmPassword = strings.TrimSpace(confirmPassword)

	if tokenString == "" {
		return errors.New("invalid or expired reset token")
	}

	if password == "" || confirmPassword == "" {
		return errors.New("password fields are required")
	}

	if password != confirmPassword {
		return errors.New("passwords do not match")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		return errors.New("JWT_SECRET is not configured")
	}

	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {

			/*
				Explicitly require HS256.

				Do not accept an arbitrary signing algorithm.
			*/
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}

			return []byte(jwtSecret), nil
		},
		jwt.WithValidMethods(
			[]string{jwt.SigningMethodHS256.Alg()},
		),
	)

	if err != nil || !token.Valid {
		return errors.New("invalid or expired reset token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok {
		return errors.New("invalid token claims")
	}

	/*
		user_id
	*/
	userIDFloat, ok := claims["user_id"].(float64)

	if !ok {
		return errors.New("invalid token user")
	}

	userID := int(userIDFloat)

	if userID <= 0 {
		return errors.New("invalid token user")
	}

	/*
		pwd_sig
	*/
	tokenPasswordSig, ok := claims["pwd_sig"].(string)

	if !ok || tokenPasswordSig == "" {
		return errors.New("invalid token signature")
	}

	/*
		Get user.
	*/
	user, err := repository.GetUserByID(
		db,
		userID,
	)

	if err != nil {
		return errors.New("user not found")
	}

	/*
		Recalculate password signature.

		This invalidates old reset tokens whenever
		the user's password changes.
	*/

	currentHashBytes := sha256.Sum256(
		[]byte(user.PasswordHash),
	)

	currentPasswordSig := hex.EncodeToString(
		currentHashBytes[:],
	)

	if currentPasswordSig != tokenPasswordSig {
		return errors.New(
			"reset token invalidated by password change",
		)
	}

	/*
		Get password history.
	*/
	history, err := repository.GetRecentPasswordHistory(
		db,
		user.UserID,
		5,
	)

	if err != nil {
		return err
	}

	/*
		Prevent reusing the current password.
	*/
	if hash.CheckPassword(
		user.PasswordHash,
		password,
	) == nil {
		return errors.New(
			"new password cannot be your current password",
		)
	}

	/*
		Prevent reuse of recent passwords.
	*/
	for _, oldPassword := range history {

		if hash.CheckPassword(
			oldPassword.PasswordHash,
			password,
		) == nil {
			return errors.New(
				"password was already used before",
			)
		}
	}

	/*
		Store current password in history
		before replacing it.
	*/
	err = repository.CreatePasswordHistory(
		db,
		user.UserID,
		user.PasswordHash,
	)

	if err != nil {
		return err
	}

	/*
		Hash the new password.
	*/
	newHash, err := hash.HashPassword(password)

	if err != nil {
		return err
	}

	/*
		Update password.
	*/
	err = repository.UpdatePassword(
		db,
		user.UserID,
		newHash,
	)

	if err != nil {
		return err
	}

	/*
		Audit successful password reset.
	*/
	if err := repository.InsertAuditLog(
		db,
		&user.UserID,
		3,
		1,
		"Password reset completed",
		ip,
		userAgent,
	); err != nil {
		return err
	}

	return nil
}
