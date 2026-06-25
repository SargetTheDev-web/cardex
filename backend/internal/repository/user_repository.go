package repository

import (
	"context"

	"backend/internal/models"

	"github.com/jackc/pgx/v5"
)

func GetUserByEmail(conn *pgx.Conn, email string) (*models.User, error) {
	var user models.User

	query := `
		SELECT
			user_id,
			username,
			email_address,
			password_hash,
			status_id,
			role_id,
			login_retry_count,
			datetimelocked,
			expiration_date,
			last_activity_at
		FROM "user"
		WHERE email_address = $1
	`

	err := conn.QueryRow(context.Background(), query, email).Scan(
		&user.UserID,
		&user.Username,
		&user.EmailAddress,
		&user.PasswordHash,
		&user.StatusID,
		&user.RoleID,
		&user.LoginRetryCount,
		&user.DateTimeLocked,
		&user.ExpirationDate,
		&user.LastActivityAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
