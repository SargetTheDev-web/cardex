// pkg/token/jwt.go

package token

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const JWTExpiration = 24 * time.Hour

func GenerateJWT(userID int) (string, time.Time, error) {

	expiresAt := time.Now().UTC().Add(JWTExpiration)

	claims := jwt.MapClaims{
		"user_id": userID,
		"iat":     time.Now().UTC().Unix(),
		"exp":     expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		[]byte(os.Getenv("JWT_SECRET")),
	)

	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil

}
