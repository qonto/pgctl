package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Extensions struct {
	Name             string
	InstalledVersion string
}

type ExtensionsUpdatable struct {
	Name                string
	InstalledVersion    string
	DefaultVersion      string
	NeedOnlyMinorUpdate bool
}

func (db *DB) GetExtensions(selectedDatabase string) ([]Extensions, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(selectedDatabase))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT
		  name,
		  installed_version as installedversion
		FROM pg_available_extensions
		WHERE installed_version != ''`)
	if err != nil {
		return nil, fmt.Errorf("unable to list extensions: %w", err)
	}
	defer rows.Close()

	extensions, err := pgx.CollectRows(rows, pgx.RowToStructByName[Extensions])
	if err != nil {
		return nil, fmt.Errorf("unable to extract extensions to update from SQL query: %w", err)
	}

	return extensions, nil
}

func (db *DB) GetExtensionsUpdatable(selectedDatabase string) ([]ExtensionsUpdatable, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(selectedDatabase))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT
		  name,
		  installed_version as installedversion,
		  default_version as defaultversion,
		  CASE
		    WHEN position('.' in installed_version) = 0 AND position('.' in default_version) = 0 THEN
		      CASE WHEN installed_version = default_version THEN true ELSE false END
		    WHEN SPLIT_PART(installed_version, '.', 1) = SPLIT_PART(default_version, '.', 1) THEN
			  true ELSE false
		  END as needonlyminorupdate
		FROM pg_available_extensions
		WHERE installed_version != ''
		AND default_version != installed_version`)
	if err != nil {
		return nil, fmt.Errorf("unable to list extensions available: %w", err)
	}
	defer rows.Close()

	extensionsUpdatable, err := pgx.CollectRows(rows, pgx.RowToStructByName[ExtensionsUpdatable])
	if err != nil {
		return nil, fmt.Errorf("unable to extract extensions to update from SQL query: %w", err)
	}

	return extensionsUpdatable, nil
}

func (db *DB) UpdateExtensions(selectedDatabase string, extensions []string) error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(selectedDatabase))
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	for _, extension := range extensions {
		updateSQL := fmt.Sprintf("ALTER EXTENSION %s UPDATE", extension)
		_, err := conn.Exec(context.Background(), updateSQL)
		if err != nil {
			return fmt.Errorf("unable to run update of %s in %s : %w", extension, selectedDatabase, err)
		}
	}
	return nil
}
