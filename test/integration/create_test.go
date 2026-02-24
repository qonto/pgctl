package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateSubscription(t *testing.T) {
	ctx := context.Background()

	t.Run("create_subscription_golden_path", func(t *testing.T) {
		// Setup publisher and subscriber containers
		publisher, subscriber := SetupPublisherSubscriberContainers(ctx, t)
		defer publisher.Cleanup(ctx, t)
		defer subscriber.Cleanup(ctx, t)

		// Wait for containers to be ready
		publisher.WaitForReadiness(ctx, t, 30*time.Second)
		subscriber.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges to both users
		publisher.ExecuteSQL(ctx, t, "ALTER ROLE publisher_user REPLICATION")
		subscriber.ExecuteSQL(ctx, t, "ALTER ROLE subscriber_user REPLICATION")

		// Create test table and publication in publisher
		err := publisher.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)
		require.NoError(t, err, "Failed to create test table in publisher")
		err = publisher.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, "CREATE PUBLICATION pub_publisher_db FOR TABLE test_users")
		require.NoError(t, err, "Failed to publish table")

		// create test table in subscriber (it needs to exist to be properly subscribed)
		err = subscriber.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)
		require.NoError(t, err, "Failed to create test table in subscriber")

		// Create temporary configuration file for both containers
		CreateTempPgctlConfigForPublisherSubscriber(t, publisher, subscriber)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "subscription", "--on", "subscriber", "--from", "publisher", "--publication", "pub_publisher_db", "--apply")

		// Should succeed and create the subscription
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Subscription sub_subscriber_db created on subscriber from publisher on publication pub_publisher_db")

		// Verify the subscription was actually created
		pool := subscriber.CreatePgxPool(ctx, t)
		defer pool.Close()

		var subName string
		err = pool.QueryRow(ctx, "SELECT subname FROM pg_subscription WHERE subname = 'sub_subscriber_db'").Scan(&subName)
		require.NoError(t, err)
		require.Equal(t, "sub_subscriber_db", subName)
	})

	t.Run("create_subscription_dry_run", func(t *testing.T) {
		// Setup publisher and subscriber containers
		publisher, subscriber := SetupPublisherSubscriberContainers(ctx, t)
		defer publisher.Cleanup(ctx, t)
		defer subscriber.Cleanup(ctx, t)

		// Wait for containers to be ready
		publisher.WaitForReadiness(ctx, t, 30*time.Second)
		subscriber.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges to both users
		publisher.ExecuteSQL(ctx, t, "ALTER ROLE publisher_user REPLICATION")
		subscriber.ExecuteSQL(ctx, t, "ALTER ROLE subscriber_user REPLICATION")

		// Create test table and publication in publisher
		publisher.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		publisher.ExecuteSQL(ctx, t, "CREATE PUBLICATION test_pub FOR TABLE test_users")

		// Create temporary configuration file for both containers
		CreateTempPgctlConfigForPublisherSubscriber(t, publisher, subscriber)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "subscription", "--on", "subscriber", "--from", "publisher", "--publication", "test_pub")

		// Should succeed in dry run mode
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "👉 Would create subscription sub_subscriber_db on subscriber from publisher on publication test_pub")

		// Verify the subscription was NOT actually created
		pool := subscriber.CreatePgxPool(ctx, t)
		defer pool.Close()

		var subName string
		err := pool.QueryRow(ctx, "SELECT subname FROM pg_subscription WHERE subname = 'sub_subscriber_db'").Scan(&subName)
		require.Error(t, err, "Subscription should not exist in dry run mode")
	})

	t.Run("create_subscription_publication_does_not_exist", func(t *testing.T) {
		// Setup publisher and subscriber containers
		publisher, subscriber := SetupPublisherSubscriberContainers(ctx, t)
		defer publisher.Cleanup(ctx, t)
		defer subscriber.Cleanup(ctx, t)

		// Wait for containers to be ready
		publisher.WaitForReadiness(ctx, t, 30*time.Second)
		subscriber.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges to both users
		publisher.ExecuteSQL(ctx, t, "ALTER ROLE publisher_user REPLICATION")
		subscriber.ExecuteSQL(ctx, t, "ALTER ROLE subscriber_user REPLICATION")

		// Create test table but NO publication in publisher
		publisher.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file for both containers
		CreateTempPgctlConfigForPublisherSubscriber(t, publisher, subscriber)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "subscription", "--on", "subscriber", "--from", "publisher", "--publication", "nonexistent_pub", "--apply")

		// Should fail because the publication doesn't exist
		result.AssertStdoutContains(t, "Subscription pre-checks failed: publication nonexistent_pub does not exist in publisher_db")
	})

	t.Run("create_subscription_wal_level_not_logical", func(t *testing.T) {
		// Setup publisher and subscriber containers with default WAL level (replica)
		publisher := SetupPostgreSQLContainer(ctx, t)
		subscriber := SetupPostgreSQLContainer(ctx, t)
		defer publisher.Cleanup(ctx, t)
		defer subscriber.Cleanup(ctx, t)

		// Wait for containers to be ready
		publisher.WaitForReadiness(ctx, t, 30*time.Second)
		subscriber.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges to both users
		publisher.ExecuteSQL(ctx, t, "ALTER ROLE testuser REPLICATION")
		subscriber.ExecuteSQL(ctx, t, "ALTER ROLE testuser REPLICATION")

		// Create test table and publication in publisher
		publisher.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		publisher.ExecuteSQL(ctx, t, "CREATE PUBLICATION test_pub FOR TABLE test_users")

		// Create temporary configuration file for both containers
		configContent := fmt.Sprintf(`publisher:
  database: testdb
  host: %s
  password: testpass
  port: %d
  role: testuser
subscriber:
  database: testdb
  host: %s
  password: testpass
  port: %d
  role: testuser
`,
			publisher.Config.Host, publisher.Config.Port,
			subscriber.Config.Host, subscriber.Config.Port)
		CreateTempConfigWithContent(t, configContent)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "subscription", "--on", "subscriber", "--from", "publisher", "--publication", "test_pub", "--apply")

		// Should fail because WAL level is not logical
		result.AssertStdoutContains(t, "Subscription pre-checks failed: wal level is not logical: replica")
	})

	t.Run("create_subscription_no_replication_grants", func(t *testing.T) {
		// Setup publisher and subscriber containers with logical WAL level
		publisher := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		subscriber := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer publisher.Cleanup(ctx, t)
		defer subscriber.Cleanup(ctx, t)

		// Wait for containers to be ready
		publisher.WaitForReadiness(ctx, t, 30*time.Second)
		subscriber.WaitForReadiness(ctx, t, 30*time.Second)

		// Create users without replication privileges
		publisher.ExecuteSQL(ctx, t, "CREATE USER testuser_no_privs WITH PASSWORD 'testpass'")
		publisher.ExecuteSQL(ctx, t, "GRANT CONNECT ON DATABASE testdb TO testuser_no_privs")
		publisher.ExecuteSQL(ctx, t, "GRANT USAGE ON SCHEMA public TO testuser_no_privs")
		subscriber.ExecuteSQL(ctx, t, "CREATE USER testuser_no_privs WITH PASSWORD 'testpass'")
		subscriber.ExecuteSQL(ctx, t, "GRANT CONNECT ON DATABASE testdb TO testuser_no_privs")
		subscriber.ExecuteSQL(ctx, t, "GRANT USAGE ON SCHEMA public TO testuser_no_privs")

		// Create test table and publication in publisher
		publisher.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) UNIQUE NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		publisher.ExecuteSQL(ctx, t, "CREATE PUBLICATION test_pub FOR TABLE test_users")

		// Create temporary configuration file for both containers with new users
		configContent := fmt.Sprintf(`publisher:
  database: testdb
  host: %s
  password: testpass
  port: %d
  role: testuser_no_privs
subscriber:
  database: testdb
  host: %s
  password: testpass
  port: %d
  role: testuser_no_privs
`,
			publisher.Config.Host, publisher.Config.Port,
			subscriber.Config.Host, subscriber.Config.Port)
		CreateTempConfigWithContent(t, configContent)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "subscription", "--on", "subscriber", "--from", "publisher", "--publication", "test_pub", "--apply")

		// Should fail because user doesn't have replication grants
		result.AssertStdoutContains(t, "Subscription pre-checks failed: user does not have replication grants")
	})
}

func TestCreatePublication(t *testing.T) {
	ctx := context.Background()

	t.Run("create_publication_dry_run_with_tables_flag", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create test tables
		pgContainer.CreateTestTable(ctx, t)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER REFERENCES test_users(id),
				amount DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_users,test_orders")

		// Should succeed in dry run mode
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "Would create publication pub_testdb")
		result.AssertStdoutContains(t, "test_users")
		result.AssertStdoutContains(t, "test_orders")
	})

	t.Run("create_publication_apply_with_tables_flag", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create test tables
		pgContainer.CreateTestTable(ctx, t)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER REFERENCES test_users(id),
				amount DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_users,test_orders", "--apply")

		// Should succeed and create the publication
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Publication pub_testdb created on testdb")
		result.AssertStdoutContains(t, "test_users")
		result.AssertStdoutContains(t, "test_orders")

		// Verify the publication was actually created
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		var pubName string
		err := pool.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_testdb'").Scan(&pubName)
		require.NoError(t, err)
		require.Equal(t, "pub_testdb", pubName)
	})

	t.Run("create_publication_dry_run_with_all_tables_flag", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create multiple test tables
		pgContainer.CreateTestTable(ctx, t)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER REFERENCES test_users(id),
				amount DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				price DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--all-tables")

		// Should succeed in dry run mode
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "Would create publication pub_testdb")
		// Should include all tables
		result.AssertStdoutContains(t, "test_users")
		result.AssertStdoutContains(t, "test_orders")
		result.AssertStdoutContains(t, "test_products")
	})

	t.Run("create_publication_apply_with_all_tables_flag", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create multiple test tables
		pgContainer.CreateTestTable(ctx, t)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_orders (
				id SERIAL PRIMARY KEY,
				user_id INTEGER REFERENCES test_users(id),
				amount DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				price DECIMAL(10,2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--all-tables", "--apply")

		// Should succeed and create the publication
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Publication pub_testdb created on testdb")
		// Should include all tables
		result.AssertStdoutContains(t, "test_users")
		result.AssertStdoutContains(t, "test_orders")
		result.AssertStdoutContains(t, "test_products")

		// Verify the publication was actually created with all tables
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		var pubName string
		err := pool.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_testdb'").Scan(&pubName)
		require.NoError(t, err)
		require.Equal(t, "pub_testdb", pubName)
	})

	t.Run("create_publication_with_nonexistent_table", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "nonexistent_table", "--apply")

		// Should fail because table doesn't exist
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Publication pre-checks failed: table nonexistent_table does not exist")
	})

	t.Run("create_publication_with_table_without_proper_replica_identity", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a table without primary key (default replica identity with no primary key)
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_logs (
				id INTEGER,
				message TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_logs", "--apply")

		// Should fail because table has default replica identity but no primary key
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Publication pre-checks failed")
		result.AssertStdoutContains(t, "test_logs")
	})

	t.Run("create_publication_with_table_replica_identity_nothing", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a table with replica identity set to nothing
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE IF NOT EXISTS test_logs (
				id SERIAL PRIMARY KEY,
				message TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
			ALTER TABLE test_logs REPLICA IDENTITY NOTHING;
		`)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_logs", "--apply")

		// Should fail because table has replica identity set to nothing
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Publication pre-checks failed: tables do not have proper replica identity")
		result.AssertStdoutContains(t, "test_logs")
	})

	t.Run("create_publication_without_wal_level_logical", func(t *testing.T) {
		// Setup PostgreSQL container with default WAL level (replica)
		pgContainer := SetupPostgreSQLContainer(ctx, t)
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create test table
		pgContainer.CreateTestTable(ctx, t)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_users", "--apply")

		// Should succeed because the implementation doesn't validate WAL level
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Publication pre-checks failed: wal level is not logical: replica")
	})

	t.Run("create_publication_with_invalid_database_alias", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "nonexistent_alias", "--tables", "test_users")

		// Should fail because alias doesn't exist
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Alias nonexistent_alias does not exist")
	})

	t.Run("create_publication_with_mixed_valid_and_invalid_tables", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create one valid table
		pgContainer.CreateTestTable(ctx, t)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_users,nonexistent_table", "--apply")

		// Should fail because one table doesn't exist
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Publication pre-checks failed: table nonexistent_table does not exist")

		// Should not create the publication
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		var pubName string
		err := pool.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_testdb'").Scan(&pubName)
		require.Error(t, err)
	})

	t.Run("create_publication_with_duplicate_table_names", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create test table
		pgContainer.CreateTestTable(ctx, t)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_users,test_users", "--apply")

		// Should succeed (PostgreSQL handles duplicates gracefully)
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Publication pre-checks failed: table test_users is listed multiple times")

		// Should not create the publication
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		var pubName string
		err := pool.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_testdb'").Scan(&pubName)
		require.Error(t, err)
	})

	t.Run("create_publication_without_replication_grants", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a new user without any special privileges
		pgContainer.ExecuteSQL(ctx, t, "CREATE USER testuser_no_privs WITH PASSWORD 'testpass'")
		pgContainer.ExecuteSQL(ctx, t, "GRANT CONNECT ON DATABASE testdb TO testuser_no_privs")
		pgContainer.ExecuteSQL(ctx, t, "GRANT USAGE ON SCHEMA public TO testuser_no_privs")

		// Create test table
		pgContainer.CreateTestTable(ctx, t)

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
		result := executor.Execute(ctx, t, "create", "publication", "--on", "testdb", "--tables", "test_users", "--apply")

		// Should fail because user doesn't have replication grants
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Publication pre-checks failed: user does not have replication grants")

		// Should not create the publication
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		var pubName string
		err := pool.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_testdb'").Scan(&pubName)
		require.Error(t, err)
	})
}
