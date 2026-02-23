package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type QueryPublication struct {
	PubName string
}

func (db *DB) GetPublications(database string) ([]QueryPublication, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), "SELECT pubname FROM pg_publication")
	if err != nil {
		return nil, fmt.Errorf("unable to get publications: %w", err)
	}
	defer rows.Close()

	publications, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryPublication])
	if err != nil {
		return nil, fmt.Errorf("unable to extract publications names from SQL query: %w", err)
	}

	return publications, nil
}

func (db *DB) CreatePublication(publication string, tables []string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	quotedTables := make([]string, len(tables))
	for i, table := range tables {
		quotedTables[i] = fmt.Sprintf(`"%s"`, table)
	}
	tablesList := strings.Join(quotedTables, ",\n")
	query := fmt.Sprintf("CREATE PUBLICATION %s FOR TABLE\n%s", publication, tablesList)
	fmt.Println("query: ", query)
	_, err = conn.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("unable to create publication: %w", err)
	}

	return nil
}

func (db *DB) DropPublication(publication string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	_, err = conn.Exec(context.Background(), fmt.Sprintf("DROP PUBLICATION %s", publication))
	if err != nil {
		return fmt.Errorf("unable to drop publication: %w", err)
	}

	return nil
}
