// internal/middleware/auth_middleware.go

package middleware

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
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
				"status": http.StatusUnauthorized,
				"error":  "missing token",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {

			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid authorization format",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "missing token",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 1. Validate JWT
		// --------------------------------------------------

		jwtSecret := os.Getenv("JWT_SECRET")

		if jwtSecret == "" {
			log.Println("[AUTH] JWT_SECRET is not configured")

			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "authentication service is not configured",
			})
			c.Abort()
			return
		}

		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, errors.New("invalid signing method")
				}

				return []byte(jwtSecret), nil
			},
			jwt.WithValidMethods([]string{
				jwt.SigningMethodHS256.Alg(),
			}),
		)

		if err != nil || token == nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid token",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 2. Extract authenticated user ID
		// --------------------------------------------------

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid token claims",
			})
			c.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)

		if !ok ||
			userIDFloat <= 0 ||
			userIDFloat != float64(int(userIDFloat)) {

			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "invalid user identity",
			})
			c.Abort()
			return
		}

		userID := int(userIDFloat)

		// --------------------------------------------------
		// 3. Hash the raw JWT
		// --------------------------------------------------

		tokenHash := hash.HashToken(tokenString)

		// --------------------------------------------------
		// 4. Check server-side session
		// --------------------------------------------------

		sessionExists, err := repository.SessionExists(
			db,
			tokenHash,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to validate session",
			})
			c.Abort()
			return
		}

		if !sessionExists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "session has been invalidated",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 5. Retrieve idle timeout configuration
		// --------------------------------------------------

		idleTimeoutStr, err := repository.GetSystemParameter(
			db,
			"USER_IDLE_TIMEOUT_MINS",
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to retrieve session timeout configuration",
			})
			c.Abort()
			return
		}

		idleTimeoutMinutes, err := strconv.Atoi(idleTimeoutStr)

		if err != nil || idleTimeoutMinutes <= 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "invalid session timeout configuration",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 6. Check session inactivity
		// --------------------------------------------------

		idle, err := repository.IsSessionIdle(
			db,
			tokenHash,
			idleTimeoutMinutes,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to validate session activity",
			})
			c.Abort()
			return
		}

		if idle {
			// Delete the expired session.
			if err := repository.DeleteSessionByToken(
				db,
				tokenHash,
			); err != nil {
				log.Printf(
					"[SESSION EXPIRATION] Failed to delete expired session: %v",
					err,
				)
			}

			// Retrieve the audit action ID.
			actionID, auditErr := repository.GetAuditActionID(
				db,
				"SESSION_EXPIRED",
			)

			if auditErr == nil {
				// Retrieve the authentication module ID.
				moduleID, moduleErr := repository.GetAuditModuleID(
					db,
					"AUTHENTICATION",
				)

				if moduleErr == nil {
					auditErr = repository.InsertAuditLog(
						db,
						&userID,
						actionID,
						moduleID,
						"User session expired due to inactivity",
						c.ClientIP(),
						c.Request.UserAgent(),
					)
				} else {
					auditErr = moduleErr
				}
			}

			if auditErr != nil {
				log.Printf(
					"[SESSION EXPIRATION] Failed to record audit log: %v",
					auditErr,
				)
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"status": http.StatusUnauthorized,
				"error":  "session expired due to inactivity",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 7. Update session activity
		// --------------------------------------------------

		if err := repository.UpdateSessionActivity(
			db,
			tokenHash,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": http.StatusInternalServerError,
				"error":  "failed to update session activity",
			})
			c.Abort()
			return
		}

		// --------------------------------------------------
		// 8. Store authentication context
		// --------------------------------------------------

		c.Set("user_id", userID)
		c.Set("token", tokenString)

		c.Next()
	}
}
