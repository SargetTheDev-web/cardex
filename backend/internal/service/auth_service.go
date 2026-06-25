package service

import (
	"errors"
	"fmt"
	"time"

	"backend/internal/repository"
	"backend/pkg/token"

	"backend/pkg/hash"

	"github.com/jackc/pgx/v5"
)

func Login(conn *pgx.Conn, email string, password string) (string, error) {
	user, err := repository.GetUserByEmail(conn, email)
	if err != nil {
		return "", errors.New("user not found")
	}

	fmt.Println("DB HASH:", user.PasswordHash)
	fmt.Println("INPUT PASSWORD:", password)
	err = hash.CheckPassword(user.PasswordHash, password)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// ACTIVE = 3 based on your seed
	if user.StatusID != 3 {
		return "", errors.New("account is not active")
	}

	if time.Now().After(user.ExpirationDate) {
		return "", errors.New("account expired")
	}

	jwtToken, err := token.GenerateJWT(user.UserID)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}
