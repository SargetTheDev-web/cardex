package repository

import (
	"backend/internal/models"
	"context"

	"github.com/jackc/pgx/v5"
)

func GetUserByIdentifier(conn *pgx.Conn, identifier string) (*models.User, error) {
	var user models.User

	query := `
	SELECT user_id, username, email_address, password_hash,
	status_id, role_id, login_retry_count,
	datetimelocked, expiration_date, last_activity_at
	FROM "user"
	WHERE email_address = $1 OR username = $1
	`

	err := conn.QueryRow(context.Background(), query, identifier).Scan(
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

func IncrementLoginAttempts(conn *pgx.Conn, userID int) error {
	query := `
	UPDATE "user"
	SET login_retry_count = login_retry_count + 1
	WHERE user_id = $1
	`
	_, err := conn.Exec(context.Background(), query, userID)
	return err
}

func ResetLoginAttempts(conn *pgx.Conn, userID int) error {
	query := `
	UPDATE "user"
	SET
		login_retry_count = 0,
		status_id = 3,
		datetimelocked = NULL
	WHERE user_id = $1
	`

	_, err := conn.Exec(context.Background(), query, userID)
	return err
}

func LockAccount(conn *pgx.Conn, userID int) error {
	query := `
	UPDATE "user"
	SET
		status_id = 5,
		datetimelocked = CURRENT_TIMESTAMP
	WHERE user_id = $1
	`

	_, err := conn.Exec(context.Background(), query, userID)
	return err
}
