package integration

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCopySequences(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source and target
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 100 INCREMENT 2")
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 1000 INCREMENT 10")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 35, true)")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 222, true)")

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "sequences", "--from", "source_db", "--to", "target_db", "--apply")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Copying sequence public.test_seq_1 (last_value: 35)")
	result.AssertStdoutContains(t, "Copying sequence public.test_seq_2 (last_value: 222)")
	result.AssertStdoutContains(t, "Successfully copied sequences")
}

func TestCopySequencesEmptySource(t *testing.T) {
	ctx := context.Background()

	// Setup containers
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Don't create any sequences in source - test empty database

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "sequences", "--from", "source_db", "--to", "target_db", "--apply")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "✅ Successfully copied sequences from source database "+sourceContainer.Config.Database+" in source_db to target database "+targetContainer.Config.Database+" in target_db")
}

func TestCopySequencesExistingTarget(t *testing.T) {
	ctx := context.Background()

	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_brand_new_seq START 200")
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_same_seq START 100")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_brand_new_seq', 250, true)")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_same_seq', 150, true)")

	targetContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_same_seq START 1")
	targetContainer.ExecuteSQL(ctx, t, "SELECT setval('test_same_seq', 50, true)")

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "sequences", "--from", "source_db", "--to", "target_db", "--apply")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Copying sequence public.test_brand_new_seq")
	result.AssertStdoutContains(t, "Copying sequence public.test_same_seq")
	result.AssertStdoutContains(t, "✅ Successfully copied sequences from source database "+sourceContainer.Config.Database+" in source_db to target database "+targetContainer.Config.Database+" in target_db")

	executor = NewPgctlExecutor(t)
	result = executor.Execute(ctx, t, "list", "sequences", "--on", "target_db")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Found 2 sequences in database")
}

func TestCopySequencesDryRun(t *testing.T) {
	ctx := context.Background()

	// Setup containers
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create sequences in source
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE dry_run_seq START 100")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('dry_run_seq', 150, true)")

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	// Don't use --apply flag to test dry run mode
	result := executor.Execute(ctx, t, "copy", "sequences", "--from", "source_db", "--to", "target_db")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "DRY RUN MODE ACTIVATED")
	result.AssertStdoutContains(t, "dry_run_seq (last_value: 150)")
	result.AssertStdoutContains(t, "Sequences would be copied from source database "+sourceContainer.Config.Database+" in source_db to target database "+targetContainer.Config.Database+" in target_db")
}

func TestCopySchemaSuccess(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source and target
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test tables in source database
	sourceContainer.ExecuteSQL(ctx, t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE
		)`)
	sourceContainer.ExecuteSQL(ctx, t, `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			amount DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT NOW()
		)`)

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "schema", "--from", "source_db", "--to", "target_db", "--all-tables", "--apply")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Successfully copied schema from source_db to target_db")

	// Verify tables were created in target
	executor = NewPgctlExecutor(t)
	checkResult := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "target_db")
	checkResult.AssertFailure(t) // Should fail because target is no longer empty
	checkResult.AssertStdoutContains(t, "Database testdb in target_db is not empty")
}

func TestCopySchemaDryRun(t *testing.T) {
	ctx := context.Background()

	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test table in source database
	sourceContainer.ExecuteSQL(ctx, t, `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		)`)

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	// Don't use --apply flag to test dry run mode
	result := executor.Execute(ctx, t, "copy", "schema", "--from", "source_db", "--to", "target_db", "--all-tables")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "DRY RUN MODE ACTIVATED")
	result.AssertStdoutContains(t, "Would have copied schema from source_db to target_db")

	// Verify target database is still empty
	executor = NewPgctlExecutor(t)
	checkResult := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "target_db")
	checkResult.AssertSuccess(t) // Should succeed because target is still empty
	checkResult.AssertStdoutContains(t, "Database testdb in target_db is empty")
}

func TestCopySchemaTargetNotEmpty(t *testing.T) {
	ctx := context.Background()

	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test table in source database
	sourceContainer.ExecuteSQL(ctx, t, `
		CREATE TABLE source_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		)`)

	// Create table in target database to make it non-empty
	targetContainer.ExecuteSQL(ctx, t, `
		CREATE TABLE existing_table (
			id SERIAL PRIMARY KEY,
			value TEXT
		)`)

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "schema", "--from", "source_db", "--to", "target_db", "--all-tables", "--apply")

	result.AssertFailure(t)
	result.AssertStdoutContains(t, "Pre-flight check failed")
	result.AssertStdoutContains(t, "target database is not empty")
	result.AssertExitCode(t, 1)
}

func TestCopySchemaNoTables(t *testing.T) { // Note: we copy a schema even without tables as other objects might exist
	ctx := context.Background()

	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Don't create any tables in source database

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "schema", "--from", "source_db", "--to", "target_db", "--all-tables", "--apply")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Successfully copied schema from source_db to target_db")
}

func TestCopySchemaVersionMismatch(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL 15 container for source
	sourceContainer := SetupPostgreSQLContainerWithVersion(ctx, t, "postgres:15-alpine")
	defer sourceContainer.Cleanup(ctx, t)

	// Setup PostgreSQL 16 container for target (different major version)
	targetContainer := SetupPostgreSQLContainerWithVersion(ctx, t, "postgres:16-alpine")
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test table in source database
	sourceContainer.ExecuteSQL(ctx, t, `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		)`)

	configContent := fmt.Sprintf(`
source_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database, sourceContainer.Config.Host, sourceContainer.Config.Password,
		sourceContainer.Config.Port, sourceContainer.Config.User,
		targetContainer.Config.Database, targetContainer.Config.Host, targetContainer.Config.Password,
		targetContainer.Config.Port, targetContainer.Config.User)
	CreateTempConfigWithContent(t, configContent)

	executor := NewPgctlExecutor(t)
	result := executor.Execute(ctx, t, "copy", "schema", "--from", "source_db", "--to", "target_db", "--all-tables", "--apply")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Database version compatibility check failed")
}
