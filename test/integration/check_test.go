package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCheckUserHasReplicationGrants tests the check user-has-replication-grants command functionality
func TestCheckUserHasReplicationGrants(t *testing.T) {
	ctx := context.Background()

	t.Run("user_with_replication_grants_returns_success", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges to the test user
		pgContainer.ExecuteSQL(ctx, t, "ALTER ROLE testuser REPLICATION")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "user-has-replication-grants", "--on", "testdb")

		// Should succeed because user has replication grants
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ User testuser has replication grants")
	})

	t.Run("user_without_replication_grants_returns_failure", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a new user without any special privileges
		pgContainer.ExecuteSQL(ctx, t, "CREATE USER testuser_no_privs WITH PASSWORD 'testpass'")
		pgContainer.ExecuteSQL(ctx, t, "GRANT CONNECT ON DATABASE testdb TO testuser_no_privs")
		pgContainer.ExecuteSQL(ctx, t, "GRANT USAGE ON SCHEMA public TO testuser_no_privs")

		// Create temporary configuration file with the new user
		configContent := fmt.Sprintf(`testdb:
  database: %s
  host: %s
  password: testpass
  port: %d
  role: testuser_no_privs
`, pgContainer.Config.Database, pgContainer.Config.Host, pgContainer.Config.Port)
		CreateTempConfigWithContent(t, configContent)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "user-has-replication-grants", "--on", "testdb")

		// Should fail because user doesn't have replication grants
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ User testuser_no_privs does not have replication grants on testdb, give them with")
	})

	t.Run("user_with_rds_replication_grants_returns_success", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Ensure user does not have replication privileges (default state)
		pgContainer.ExecuteSQL(ctx, t, "ALTER ROLE testuser NOREPLICATION")
		// Revoke CREATE privilege on database to ensure user cannot create publications
		pgContainer.ExecuteSQL(ctx, t, "REVOKE CREATE ON DATABASE testdb FROM testuser")
		pgContainer.ExecuteSQL(ctx, t, "CREATE ROLE rds_replication")
		pgContainer.ExecuteSQL(ctx, t, "GRANT rds_replication TO testuser")
		// Grant CREATE privilege on database to rds_replication role (simulating RDS behavior)
		pgContainer.ExecuteSQL(ctx, t, "GRANT CREATE ON DATABASE testdb TO rds_replication")
		// Grant USAGE on schema to rds_replication role
		pgContainer.ExecuteSQL(ctx, t, "GRANT USAGE ON SCHEMA public TO rds_replication")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "user-has-replication-grants", "--on", "testdb")

		// Should succeed because user has rds replication grants
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ User testuser has replication grants")
	})
}

// TestCheckWalLevelIsLogical tests the check wal-level-is-logical command functionality
func TestCheckWalLevelIsLogical(t *testing.T) {
	ctx := context.Background()

	t.Run("wal_level_replica_returns_failure", func(t *testing.T) {
		// Setup PostgreSQL container with default wal_level (replica)
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "wal-level-is-logical", "--on", "testdb")

		// Should fail because wal_level is 'replica'
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Wal level is not logical: replica")
	})

	t.Run("wal_level_wal_returns_success", func(t *testing.T) {
		// Setup PostgreSQL container with wal_level set to 'wal'
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "wal-level-is-logical", "--on", "testdb")

		// Should succeed because wal_level is 'wal'
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Wal level is logical: logical")
	})
}

// TestCheckDatabaseIsEmpty tests the check database-is-empty command functionality
func TestCheckDatabaseIsEmpty(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	t.Run("database_with_only_extensions_returns_success", func(t *testing.T) {
		// Create test extensions (these should not count as making database non-empty)
		pgContainer.CreateTestExtensions(ctx, t)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should succeed because extensions don't count as database content
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Database testdb in testdb is empty")
	})

	t.Run("empty_database_returns_success", func(t *testing.T) {
		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should succeed because database is empty
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Database testdb in testdb is empty")
	})

	t.Run("database_with_table_returns_failure", func(t *testing.T) {
		// Create a table to make the database non-empty
		pgContainer.CreateTestTable(ctx, t)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should fail because database has a table
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Database testdb in testdb is not empty")
	})

	t.Run("database_with_table_and_data_returns_failure", func(t *testing.T) {
		// Create a table and insert data
		pgContainer.CreateTestTable(ctx, t)
		pgContainer.InsertTestData(ctx, t)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should fail because database has a table with data
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Database testdb in testdb is not empty")
	})

	t.Run("database_with_sequence_returns_failure", func(t *testing.T) {
		// Create a sequence
		pgContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_sequence START 1;")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should fail because database has a sequence
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Database testdb in testdb is not empty")
	})

	t.Run("database_with_view_returns_failure", func(t *testing.T) {
		// Create a table first (views need underlying tables)
		pgContainer.CreateTestTable(ctx, t)

		// Create a view
		pgContainer.ExecuteSQL(ctx, t, "CREATE VIEW test_view AS SELECT id, name FROM test_users;")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should fail because database has a view (and table)
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Database testdb in testdb is not empty")
	})

	t.Run("database_with_materialized_view_returns_failure", func(t *testing.T) {
		// Create a table first (materialized views need underlying tables)
		pgContainer.CreateTestTable(ctx, t)

		// Create a materialized view
		pgContainer.ExecuteSQL(ctx, t, "CREATE MATERIALIZED VIEW test_matview AS SELECT id, name FROM test_users;")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should fail because database has a materialized view (and table)
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Database testdb in testdb is not empty")
	})

	t.Run("database_with_index_returns_failure", func(t *testing.T) {
		// Create a table first (indexes need underlying tables)
		pgContainer.CreateTestTable(ctx, t)

		// Create an additional index
		pgContainer.ExecuteSQL(ctx, t, "CREATE INDEX test_users_name_idx ON test_users(name);")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "database-is-empty", "--on", "testdb")

		// Should fail because database has an index (and table)
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Database testdb in testdb is not empty")
	})
}

// TestCheckTablesHaveProperReplicaIdentity tests the tables-have-proper-replica-identity command functionality
func TestCheckTablesHaveProperReplicaIdentity(t *testing.T) {
	ctx := context.Background()

	t.Run("check_tables_with_proper_replica_identity_success", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create tables with proper replica identity (primary keys)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(255) UNIQUE NOT NULL
			);
			CREATE TABLE orders (
				order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id INTEGER REFERENCES users(id),
				total_amount DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "tables-have-proper-replica-identity", "--on", "testdb", "--tables", "users,orders")

		// Should succeed because all tables have primary keys (proper replica identity)
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ All tables have proper replica identity")
	})

	t.Run("check_tables_with_default_replica_identity_but_no_primary_key", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a table without primary key (default replica identity)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE test_logs (
				id INTEGER,
				message TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "tables-have-proper-replica-identity", "--on", "testdb", "--tables", "test_logs")

		// Should fail because table has default replica identity but no primary key
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Tables do not have proper replica identity:")
		result.AssertStdoutContains(t, "test_logs")
	})

	t.Run("check_tables_with_replica_identity_nothing", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a table with replica identity set to nothing
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE test_logs (
				id SERIAL PRIMARY KEY,
				message TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			ALTER TABLE test_logs REPLICA IDENTITY NOTHING;
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "tables-have-proper-replica-identity", "--on", "testdb", "--tables", "test_logs")

		// Should fail because table has replica identity set to nothing
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Tables do not have proper replica identity:")
		result.AssertStdoutContains(t, "test_logs")
	})

	t.Run("check_tables_with_mixed_replica_identity", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create mixed tables - some with proper, some without proper replica identity
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(255) UNIQUE NOT NULL
			);
			CREATE TABLE test_logs (
				id INTEGER,
				message TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE products (
				product_id INTEGER PRIMARY KEY,
				name VARCHAR(200) NOT NULL,
				price DECIMAL(10,2) NOT NULL
			);
			ALTER TABLE products REPLICA IDENTITY NOTHING;
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "tables-have-proper-replica-identity", "--on", "testdb", "--tables", "users,test_logs,products")

		// Should fail because test_logs and products have improper replica identity
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Tables do not have proper replica identity:")
		result.AssertStdoutContains(t, "test_logs")
		result.AssertStdoutContains(t, "products")
		// Should not mention users (which has proper replica identity)
		require.NotContains(t, result.Stdout, "users")
	})

	t.Run("check_tables_with_all_tables_flag", func(t *testing.T) {
		// Setup PostgreSQL container
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create tables with proper replica identity
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(255) UNIQUE NOT NULL
			);
			CREATE TABLE orders (
				order_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id INTEGER REFERENCES users(id),
				total_amount DECIMAL(10,2) NOT NULL
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "tables-have-proper-replica-identity", "--on", "testdb", "--all-tables")

		// Should succeed because all tables have proper replica identity
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ All tables have proper replica identity")
	})
}

func TestCheckHaveSimilarSequences(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source and target
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 200")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 35, true)")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 222, true)")

	targetContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
	targetContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 200")
	targetContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 35, true)")
	targetContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 222, true)")

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
	result := executor.Execute(ctx, t, "check", "have-similar-sequences", "--from", "source_db", "--to", "target_db")

	result.AssertSuccess(t)
	result.AssertStdoutContains(t, "Sequences are the same")
	result.AssertStdoutContains(t, "Sequence public.test_seq_1: source source_db 35 and target target_db 35")
	result.AssertStdoutContains(t, "Sequence public.test_seq_2: source source_db 222 and target target_db 222")
}

func TestCheckSequencesDifferentValues(t *testing.T) {
	ctx := context.Background()

	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 200")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 50, true)")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 300, true)")

	targetContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
	targetContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 200")
	targetContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 25, true)")
	targetContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 250, true)")

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
	result := executor.Execute(ctx, t, "check", "have-similar-sequences", "--from", "source_db", "--to", "target_db")

	result.AssertFailure(t)
	result.AssertStdoutContains(t, "Sequences are different")
	result.AssertStdoutContains(t, "Sequence public.test_seq_1 has different last values: source source_db 50 and target target_db 25")
	result.AssertStdoutContains(t, "Sequence public.test_seq_2 has different last values: source source_db 300 and target target_db 250")
}

func TestCheckSequencesMissingSequence(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source and target
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create sequences only in source
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
	sourceContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 200")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 50, true)")
	sourceContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 300, true)")

	// Create only one sequence in target
	targetContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
	targetContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 50, true)")

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
	result := executor.Execute(ctx, t, "check", "have-similar-sequences", "--from", "source_db", "--to", "target_db")

	result.AssertFailure(t)
	result.AssertStdoutContains(t, "Sequences are different")
	result.AssertStdoutContains(t, "Sequence public.test_seq_1: source source_db 50 and target target_db 50")
	result.AssertStdoutContains(t, "Sequence public.test_seq_2 not found in target")
}

func TestCheckSequencesEmptyDatabases(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source and target
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Don't create any sequences - both databases should be empty

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

	t.Run("empty_databases_return_success", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "have-similar-sequences", "--from", "source_db", "--to", "target_db")

		// Should succeed because both databases are empty
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ No sequences found in source database testdb")
	})
}

// TestCheckSubscriptionLag tests the check subscription-lag command functionality
func TestCheckSubscriptionLag(t *testing.T) {
	ctx := context.Background()

	// Setup publisher and subscriber containers
	publisher, subscriber := SetupPublisherSubscriberContainers(ctx, t)
	defer publisher.Cleanup(ctx, t)
	defer subscriber.Cleanup(ctx, t)

	// Create temporary configuration file for both containers
	CreateTempPgctlConfigForPublisherSubscriber(t, publisher, subscriber)

	// Wait for containers to be ready
	publisher.WaitForReadiness(ctx, t, 30*time.Second)
	subscriber.WaitForReadiness(ctx, t, 30*time.Second)

	executor := NewPgctlExecutor(t)
	// Create test table and publication in publisher
	publisher.ExecuteSQL(ctx, t, `
		CREATE TABLE test_users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	err := publisher.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, "CREATE PUBLICATION test_pub FOR TABLE test_users")
	require.NoError(t, err, "Failed to publish table")

	t.Run("subscription_does_not_exist", func(t *testing.T) {
		result := executor.Execute(ctx, t, "check", "subscription-lag", "--on", "subscriber", "--name", "nonexistent_subscription")

		// Should fail because the subscription doesn't exist
		result.AssertFailure(t)
		result.AssertStderrContains(t, "❌ Unable to get subscription lag:")
	})
	t.Run("subscription_does_exist", func(t *testing.T) {
		// Grant replication privileges to both users
		publisher.ExecuteSQL(ctx, t, "ALTER ROLE publisher_user REPLICATION")
		subscriber.ExecuteSQL(ctx, t, "ALTER ROLE subscriber_user REPLICATION")

		subscriptionSQL := fmt.Sprintf(`
			CREATE SUBSCRIPTION sub_subscriber_db
			CONNECTION 'postgres://%s:%s@publisher:5432/publisher_db'
			PUBLICATION pub_publisher_db
			WITH (connect = true, create_slot = true)
		`, publisher.Config.User, publisher.Config.Password)
		err = subscriber.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, subscriptionSQL)
		require.NoError(t, err, "Failed to create subscription")

		result := executor.Execute(ctx, t, "check", "subscription-lag", "--on", "publisher", "--name", "sub_subscriber_db")
		// Should work
		result.AssertSuccess(t)
	})
}

func TestCheckRolesBetweenSourceAndTarget(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source (publisher) and target (subscriber)
	sourceContainer := SetupPostgreSQLContainer(ctx, t)
	defer sourceContainer.Cleanup(ctx, t)

	targetContainer := SetupPostgreSQLContainer(ctx, t)
	defer targetContainer.Cleanup(ctx, t)

	sourceContainer.WaitForReadiness(ctx, t, 30*time.Second)
	targetContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Clean up users in case they already exist (for repeatability of test)
	sourceContainer.ExecuteSQL(ctx, t, "DROP ROLE IF EXISTS source_role")
	targetContainer.ExecuteSQL(ctx, t, "DROP ROLE IF EXISTS source_role")

	// Add a user that should match on both source and target
	sourceContainer.ExecuteSQL(ctx, t, "CREATE ROLE source_role WITH PASSWORD 'testpass'")
	targetContainer.ExecuteSQL(ctx, t, "CREATE ROLE source_role WITH PASSWORD 'testpass'")

	// Compose config file content
	configContent := fmt.Sprintf(`
source_db_instance:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
target_db_instance:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		sourceContainer.Config.Database,
		sourceContainer.Config.Host,
		sourceContainer.Config.Password,
		sourceContainer.Config.Port,
		sourceContainer.Config.User,
		targetContainer.Config.Database,
		targetContainer.Config.Host,
		targetContainer.Config.Password,
		targetContainer.Config.Port,
		targetContainer.Config.User,
	)
	CreateTempConfigWithContent(t, configContent)

	t.Run("roles_exist_on_target_instance", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "roles-between-source-and-target", "--from", "source_db_instance", "--to", "target_db_instance")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "All roles of the source instance source_db_instance exist on the target instance target_db_instance")
	})

	t.Run("roles_missing_on_target_instance", func(t *testing.T) {
		sourceContainer.ExecuteSQL(ctx, t, "DROP ROLE IF EXISTS missing_source_role")
		sourceContainer.ExecuteSQL(ctx, t, "CREATE ROLE missing_source_role WITH PASSWORD 'testpass'")

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "check", "roles-between-source-and-target", "--from", "source_db_instance", "--to", "target_db_instance")

		result.AssertStdoutContains(t, "The following roles exist on the source instance source_db_instance but missing on the target instance target_db_instance")
		result.AssertStdoutContains(t, "missing_source_role")
	})
}
