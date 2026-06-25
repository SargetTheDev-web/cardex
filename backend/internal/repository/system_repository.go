package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func GetSystemParameter(conn *pgx.Conn, key string) (string, error) {
	var value string

	query := `
	SELECT parameter_value
	FROM system_parameter
	WHERE parameter_key = $1
	`

	err := conn.QueryRow(context.Background(), query, key).Scan(&value)
	return value, err
}
