package db

import (
	"context"
	"fmt"
	"os"

	"backend/internal/config"

	"github.com/jackc/pgx/v5"
)

func Connect() (*pgx.Conn, error) {
	config.LoadEnv()

	fmt.Println("DB_USER:", os.Getenv("DB_USER"))
	fmt.Println("DB_NAME:", os.Getenv("DB_NAME"))
	fmt.Println("DB_HOST:", os.Getenv("DB_HOST"))
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
