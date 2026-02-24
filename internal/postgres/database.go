package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port" default:"5432"`
	Database string `mapstructure:"database"`
	Role     string `mapstructure:"role"`
	Password string `mapstructure:"password"`
}
type queryDatname struct {
	Datname string
}

type QueryTable struct {
	Tablename string
}

func (db *DB) Ping() error {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	return conn.Ping(context.Background())
}

func (db *DB) getConnString(dbname string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		db.Host, db.Port, db.Role, db.Password, dbname)
}

func (db *DB) GetAllDatabases() ([]string, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT datname
		FROM pg_catalog.pg_database
		WHERE datname NOT IN ('template0', 'template1')
		ORDER BY 1`)
	if err != nil {
		err = fmt.Errorf("unable to list databases: %w", err)
		return nil, err
	}
	defer rows.Close()

	databases, err := pgx.CollectRows(rows, pgx.RowToStructByName[queryDatname])
	if err != nil {
		return nil, fmt.Errorf("unable to extract databases names from SQL query: %w", err)
	}

	result := make([]string, 0)
	for _, d := range databases {
		result = append(result, d.Datname)
	}
	return result, nil
}

func (db *DB) GetWalLevel() (string, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return "", fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	var walLevel string
	err = conn.QueryRow(context.Background(), "SHOW wal_level").Scan(&walLevel)
	if err != nil {
		return "", fmt.Errorf("unable to get wal level: %w", err)
	}

	return walLevel, nil
}

func (db *DB) DatabaseIsEmpty() (bool, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return false, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	// Check for tables, sequences, indexes, views, materialized views, and foreign tables
	var count int
	err = conn.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM (
			SELECT table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
			UNION ALL
			SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema NOT IN ('pg_catalog', 'information_schema')
			UNION ALL
			SELECT indexname FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
			UNION ALL
			SELECT matviewname FROM pg_matviews WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
			UNION ALL
			SELECT foreign_table_name FROM information_schema.foreign_tables WHERE foreign_table_schema NOT IN ('pg_catalog', 'information_schema')
		) as objects`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("unable to check if database is empty: %w", err)
	}

	return count == 0, nil
}

func (db *DB) GetTables() ([]QueryTable, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname != 'pg_catalog'
		AND schemaname != 'information_schema'`)
	if err != nil {
		return nil, fmt.Errorf("unable to list tables: %w", err)
	}
	defer rows.Close()

	tables, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryTable])
	if err != nil {
		return nil, fmt.Errorf("unable to extract tables names from SQL query: %w", err)
	}

	return tables, nil
}

func (db *DB) GetTablesWithSchema() ([]QueryTable, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), `
		SELECT schemaname::text || '.' || tablename::text as tablename
		FROM pg_tables
		WHERE schemaname != 'pg_catalog'
		AND schemaname != 'information_schema'`)
	if err != nil {
		return nil, fmt.Errorf("unable to list tables: %w", err)
	}
	defer rows.Close()

	tables, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryTable])
	if err != nil {
		return nil, fmt.Errorf("unable to extract tables names from SQL query: %w", err)
	}

	return tables, nil
}

func ConvertToStringArray(tables []QueryTable) []string {
	tableNames := make([]string, len(tables))
	for i, table := range tables {
		tableNames[i] = table.Tablename
	}
	return tableNames
}

func (db *DB) GetTablesWithoutProperReplicaIdentity(tables []string) ([]string, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	var query string
	if len(tables) > 0 {
		escapedTables := make([]string, len(tables))
		for i, table := range tables {
			escapedTables[i] = fmt.Sprintf("'%s'", table)
		}
		query = fmt.Sprintf(`
			SELECT t.relname
			FROM pg_class t
			WHERE t.relnamespace = 'public'::regnamespace
				AND t.relkind = 'r'
				AND t.relname IN (%s)
				AND (
					-- Tables with replica identity set to nothing
					t.relreplident = 'n'
					OR
					-- Tables with replica identity set to default but no primary key exists
					(t.relreplident = 'd' AND NOT EXISTS (
						SELECT 1
						FROM pg_constraint c
						WHERE c.conrelid = t.oid
							AND c.contype = 'p'
							AND c.connamespace = 'public'::regnamespace
					))
				)`, strings.Join(escapedTables, ","))
	} else {
		query = `
			SELECT t.relname
			FROM pg_class t
			WHERE t.relnamespace = 'public'::regnamespace
				AND t.relkind = 'r'
				AND (
					-- Tables with replica identity set to nothing
					t.relreplident = 'n'
					OR
					-- Tables with replica identity set to default but no primary key exists
					(t.relreplident = 'd' AND NOT EXISTS (
						SELECT 1
						FROM pg_constraint c
						WHERE c.conrelid = t.oid
							AND c.contype = 'p'
							AND c.connamespace = 'public'::regnamespace
					))
				)`
	}

	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("unable to query tables without proper replica identity: %w", err)
	}
	defer rows.Close()

	tablesWithoutReplicaIdentity, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("unable to collect table names: %w", err)
	}

	return tablesWithoutReplicaIdentity, nil
}

func (db *DB) HasReplicationGrants() (bool, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return false, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	// Test replication grants by creating and deleting a publication within a transaction
	testPublicationName := "pgctl_test_replication_grants"

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return false, fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background()) //nolint: errcheck

	// Try to create a test publication
	_, err = tx.Exec(context.Background(), fmt.Sprintf("CREATE PUBLICATION %s", testPublicationName))
	if err != nil {
		// If creation fails, user doesn't have replication grants
		return false, nil
	}

	// Try to drop the test publication
	_, err = tx.Exec(context.Background(), fmt.Sprintf("DROP PUBLICATION %s", testPublicationName))
	if err != nil {
		return false, nil
	}

	// Commit the transaction if both succeeded
	if err := tx.Commit(context.Background()); err != nil {
		return false, fmt.Errorf("unable to commit transaction: %w", err)
	}

	return true, nil
}

func (db *DB) HasSubscriptionGrants() (bool, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return false, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	var count int

	err = conn.QueryRow(context.Background(), `
	SELECT COUNT(*) FROM pg_roles WHERE pg_has_role( $1, oid, 'member') AND rolname = 'pg_create_subscription'
	`, db.Role).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("unable to check if user has subscription grants: %w", err)
	}

	return count > 0, nil
}

// DumpSchema uses pg_dump to create schema-only dump for specified tables
func (db *DB) DumpSchema() ([]byte, error) {
	// First try with local pg_dump
	output, err := db.dumpSchemaLocal()
	if err != nil {
		// If local pg_dump fails due to version mismatch, try Docker
		if strings.Contains(err.Error(), "server version mismatch") || strings.Contains(err.Error(), "version") {
			return db.dumpSchemaDocker()
		}
		return nil, err
	}
	return output, nil
}

// dumpSchemaLocal attempts to use local pg_dump
func (db *DB) dumpSchemaLocal() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	args := []string{
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--no-sync",
		"-h", db.Host,
		"-p", strconv.Itoa(db.Port),
		"-U", db.Role,
		db.Database,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...) //nolint: gosec
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password))

	output, err := cmd.Output()
	if err != nil {
		// Capture stderr if available
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf("pg_dump failed: %w, stderr: %s", err, string(exitError.Stderr))
		}
		return nil, fmt.Errorf("pg_dump failed: %w", err)
	}

	return output, nil
}

// dumpSchemaDocker uses Docker to run a compatible pg_dump version
func (db *DB) dumpSchemaDocker() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	// Get server version to determine which Docker image to use
	serverVersion, err := db.GetServerVersion()
	if err != nil {
		return nil, fmt.Errorf("unable to get server version for Docker pg_dump: %w", err)
	}

	// Use PostgreSQL Docker image matching the server version
	image := fmt.Sprintf("postgres:%d-alpine", serverVersion.Major)

	args := []string{
		"run", "--rm", "--network", "host",
		"-e", fmt.Sprintf("PGPASSWORD=%s", db.Password),
		image,
		"pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--no-sync",
		"-h", db.Host,
		"-p", strconv.Itoa(db.Port),
		"-U", db.Role,
		db.Database,
	}

	cmd := exec.CommandContext(ctx, "docker", args...) //nolint: gosec

	output, err := cmd.Output()
	if err != nil {
		// Capture stderr if available
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return nil, fmt.Errorf("docker pg_dump failed: %w, stderr: %s", err, string(exitError.Stderr))
		}
		return nil, fmt.Errorf("docker pg_dump failed: %w", err)
	}

	return output, nil
}

// RestoreSchema uses psql to restore schema from a dump file
func (db *DB) RestoreSchema(schemaSQL []byte) error {
	// First try with local psql
	err := db.restoreSchemaLocal(schemaSQL)
	if err != nil {
		// If local psql fails due to version mismatch, try Docker
		if strings.Contains(err.Error(), "server version mismatch") || strings.Contains(err.Error(), "version") {
			return db.restoreSchemaDocker(schemaSQL)
		}
		return err
	}
	return nil
}

// restoreSchemaLocal attempts to use local psql
func (db *DB) restoreSchemaLocal(schemaSQL []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	// Create a temporary file to store the schema SQL
	tmpFile, err := os.CreateTemp("", "schema_*.sql")
	if err != nil {
		return fmt.Errorf("unable to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint: errcheck // Clean up the temp file

	// Write the schema SQL to the temporary file
	if _, err := tmpFile.Write(schemaSQL); err != nil {
		err2 := tmpFile.Close()
		if err2 != nil {
			return fmt.Errorf("unable to close temporary file: %w", err2)
		}
		return fmt.Errorf("unable to write to temporary file: %w", err)
	}
	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("unable to close temporary file: %w", err)
	}

	args := []string{
		"-h", db.Host,
		"-p", strconv.Itoa(db.Port),
		"-U", db.Role,
		"-d", db.Database,
		"-f", tmpFile.Name(),
	}

	cmd := exec.CommandContext(ctx, "psql", args...) //nolint: gosec
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql failed: %w, output: %s", err, string(output))
	}

	return nil
}

// restoreSchemaDocker uses Docker to run a compatible psql version
func (db *DB) restoreSchemaDocker(schemaSQL []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	// Get server version to determine which Docker image to use
	serverVersion, err := db.GetServerVersion()
	if err != nil {
		return fmt.Errorf("unable to get server version for Docker psql: %w", err)
	}

	// Use PostgreSQL Docker image matching the server version
	image := fmt.Sprintf("postgres:%d-alpine", serverVersion.Major)

	// Create a temporary file to store the schema SQL
	tmpFile, err := os.CreateTemp("", "schema_*.sql")
	if err != nil {
		return fmt.Errorf("unable to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint: errcheck // Clean up the temp file

	// Write the schema SQL to the temporary file
	if _, err := tmpFile.Write(schemaSQL); err != nil {
		err2 := tmpFile.Close()
		if err2 != nil {
			return fmt.Errorf("unable to close temporary file: %w", err2)
		}
		return fmt.Errorf("unable to write to temporary file: %w", err)
	}
	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("unable to close temporary file: %w", err)
	}

	args := []string{
		"run", "--rm", "--network", "host",
		"-e", fmt.Sprintf("PGPASSWORD=%s", db.Password),
		"-v", fmt.Sprintf("%s:/tmp/schema.sql", tmpFile.Name()),
		image,
		"psql",
		"-h", db.Host,
		"-p", strconv.Itoa(db.Port),
		"-U", db.Role,
		"-d", db.Database,
		"-f", "/tmp/schema.sql",
	}

	cmd := exec.CommandContext(ctx, "docker", args...) //nolint: gosec

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker psql failed: %w, output: %s", err, string(output))
	}

	return nil
}

// CheckPgDumpCompatibility checks if pg_dump is available and compatible
func (db *DB) CheckPgDumpCompatibility() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if pg_dump is available
	_, err := exec.LookPath("pg_dump")
	if err != nil {
		return fmt.Errorf("pg_dump not found in PATH: %w", err)
	}
	// Get pg_dump version
	cmd := exec.CommandContext(ctx, "pg_dump", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("unable to get pg_dump version: %w", err)
	}

	pgDumpVersion, err := ParseVersion(string(output))
	if err != nil {
		return fmt.Errorf("unable to parse pg_dump version: %w", err)
	}

	// Get target database version
	targetVersion, err := db.GetServerVersion()
	if err != nil {
		return fmt.Errorf("unable to get target database version: %w", err)
	}

	// Check if major versions match
	if pgDumpVersion.Major != targetVersion.Major {
		return fmt.Errorf("pg_dump --version returned (%d.%d) which major version is not equal to major version of the target PostgreSQL database (%d.%d). This can cause compatibility issues when copying the schema. Make sure to use the same major pg_dump version as your target.",
			pgDumpVersion.Major, pgDumpVersion.Minor, targetVersion.Major, targetVersion.Minor)
	}

	return nil
}

// Version represents PostgreSQL version
type Version struct {
	Major int
	Minor int
}

// GetServerVersion gets PostgreSQL server version
func (db *DB) GetServerVersion() (*Version, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	var versionString string
	err = conn.QueryRow(context.Background(), "SELECT version()").Scan(&versionString)
	if err != nil {
		return nil, fmt.Errorf("unable to get server version: %w", err)
	}

	return ParseVersion(versionString)
}

// ParseVersion parses PostgreSQL version string
func ParseVersion(versionString string) (*Version, error) {
	// Match patterns like "PostgreSQL 15.3" or "pg_dump (PostgreSQL) 15.3"
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionString)

	if len(matches) < 3 {
		return nil, fmt.Errorf("unable to parse version from: %s", versionString)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("unable to parse major version: %w", err)
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("unable to parse minor version: %w", err)
	}

	return &Version{Major: major, Minor: minor}, nil
}

func (db *DB) ListRoles() ([]string, error) {
	conn, err := pgx.Connect(context.Background(), db.getConnString(db.Database))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(context.Background()) //nolint: errcheck

	rows, err := conn.Query(context.Background(), "SELECT rolname as rolename FROM pg_roles")
	if err != nil {
		return nil, fmt.Errorf("unable to get roles: %w", err)
	}
	defer rows.Close()

	roles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("unable to extract roles from SQL query: %w", err)
	}

	return roles, nil
}
