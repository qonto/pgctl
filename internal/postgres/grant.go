package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (db *DB) GrantGrant(grant, role string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	query := fmt.Sprintf("GRANT %s ON DATABASE %s TO %s",
		grant, // cli has input validation for grant, no need to sanitize
		pgx.Identifier{db.Database}.Sanitize(),
		pgx.Identifier{role}.Sanitize())

	_, err = conn.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("unable to grant: %w", err)
	}
	return nil
}

func (db *DB) RevokeGrant(grant, role string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	query := fmt.Sprintf("REVOKE %s ON DATABASE %s FROM %s",
		grant, // cli has input validation for grant, no need to sanitize
		pgx.Identifier{db.Database}.Sanitize(),
		pgx.Identifier{role}.Sanitize())

	_, err = conn.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("unable to revoke grant: %w", err)
	}
	return nil
}
