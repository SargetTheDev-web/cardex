package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"backend/internal/repository"
	"backend/pkg/hash"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing token",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization format",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing token",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 1. Validate JWT
		// --------------------------------------------------

		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {

				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("invalid signing method")
				}

				return []byte(os.Getenv("JWT_SECRET")), nil
			},
		)

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 2. Hash the raw JWT
		// --------------------------------------------------

		tokenHash := hash.HashToken(tokenString)

		// --------------------------------------------------
		// 3. Check server-side session
		// --------------------------------------------------

		sessionExists, err := repository.SessionExists(
			db,
			tokenHash,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to validate session",
			})
			c.Abort()
			return
		}

		if !sessionExists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "session has been invalidated",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 4. Extract authenticated user ID
		// --------------------------------------------------

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			c.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user identity",
			})
			c.Abort()
			return
		}

		userID := int(userIDFloat)

		// --------------------------------------------------
		// 5. Store authentication context
		// --------------------------------------------------

		c.Set("user_id", userID)
		c.Set("token", tokenString)

		c.Next()
	}
}
