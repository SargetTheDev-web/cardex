package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func InsertAuditLog(
	conn *pgx.Conn,
	userID *int,
	actionID int,
	moduleID int,
	description string,
	ip string,
	userAgent string,
) error {
	query := `
	INSERT INTO audit_log
	(user_id, action_id, module_id, description, ip_address, user_agent)
	VALUES ($1,$2,$3,$4,$5,$6)
	`

	_, err := conn.Exec(
		context.Background(),
		query,
		userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	)

	return err
}
