package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListTables(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL
			);
		`)

	// Create temporary configuration file using the container
	CreateTempPgctlConfig(t, pgContainer)

	executor := NewPgctlExecutor(t)

	t.Run("list_tables_basic", func(t *testing.T) {
		result := executor.Execute(ctx, t, "list", "tables", "--on", "testdb")

		// Should succeed and show table
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "test_users")
	})

	t.Run("list_tables_with_schema", func(t *testing.T) {
		result := executor.Execute(ctx, t, "list", "tables", "--on", "testdb", "--with-schema-prefix")

		// Should succeed and show table with schema
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "public.test_users")
	})
}

// TestListExtensionsImplemented tests that list extensions command is implemented and works
func TestListExtensions(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test extensions to ensure we have extensions to list
	pgContainer.CreateTestExtensions(ctx, t)

	// Create temporary configuration file using the container
	CreateTempPgctlConfig(t, pgContainer)

	executor := NewPgctlExecutor(t)

	t.Run("list_extensions_basic", func(t *testing.T) {
		result := executor.Execute(ctx, t, "list", "extensions", "--on", "testdb")

		// Should succeed and show extensions
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "Found extensions for")
		result.AssertStdoutContains(t, "testdb")
	})

	t.Run("list_extensions_all_databases", func(t *testing.T) {
		result := executor.Execute(ctx, t, "list", "extensions", "--on", "testdb", "--all-databases")

		// Should succeed and show extensions for all databases
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "Found extensions for")
		result.AssertStdoutContains(t, "testdb")
	})
}

// TestListSubscriptions tests the list subscriptions functionality
func TestListSubscriptions(t *testing.T) {
	ctx := context.Background()

	t.Run("list_subscriptions_no_subscriptions", func(t *testing.T) {
		// Setup PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "list", "subscriptions", "--on", "testdb")

		// Should succeed and show no subscriptions message
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "❌ No subscriptions found on testdb on database testdb")
	})

	t.Run("list_subscriptions_with_subscriptions", func(t *testing.T) {
		// Setup a single PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create a second database for the subscription to connect to
		pgContainer.ExecuteSQL(ctx, t, "CREATE DATABASE source_db")

		// Create test table and publication in the source database
		// Create table in source database
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL
			);
		`)

		// Create publication in source database
		pgContainer.ExecuteSQL(ctx, t, "CREATE PUBLICATION test_pub FOR TABLE test_users")

		// Create a subscription in the main database that connects to the source database
		subscriptionSQL := fmt.Sprintf(`
			CREATE SUBSCRIPTION test_sub
			CONNECTION 'host=%s port=%d dbname=source_db user=%s password=%s'
			PUBLICATION test_pub
			WITH (connect = false)
		`, pgContainer.Config.Host, pgContainer.Config.Port, pgContainer.Config.User, pgContainer.Config.Password)

		err := pgContainer.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, subscriptionSQL)
		require.NoError(t, err, "Failed to create subscription")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "list", "subscriptions", "--on", "testdb")

		// Should succeed and show the subscription
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Subscriptions on testdb on database testdb:")
		result.AssertStdoutContains(t, "test_sub")
	})

	t.Run("list_subscriptions_all_databases", func(t *testing.T) {
		// Setup a single PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for the container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Create additional databases
		pgContainer.ExecuteSQL(ctx, t, "CREATE DATABASE testdb2")
		pgContainer.ExecuteSQL(ctx, t, "CREATE DATABASE testdb3")
		pgContainer.ExecuteSQL(ctx, t, "CREATE DATABASE source_db")

		// Create test table and publication in source database
		pgContainer.ExecuteSQL(ctx, t, `
			CREATE TABLE test_users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL
			);
		`)
		pgContainer.ExecuteSQL(ctx, t, "CREATE PUBLICATION test_pub FOR TABLE test_users")

		// Create subscriptions in the main database that connect to the source database
		subscriptionSQL1 := fmt.Sprintf(`
			CREATE SUBSCRIPTION test_sub1
			CONNECTION 'host=%s port=%d dbname=source_db user=%s password=%s'
			PUBLICATION test_pub
			WITH (connect = false)
		`, pgContainer.Config.Host, pgContainer.Config.Port, pgContainer.Config.User, pgContainer.Config.Password)

		subscriptionSQL2 := fmt.Sprintf(`
			CREATE SUBSCRIPTION test_sub2
			CONNECTION 'host=%s port=%d dbname=source_db user=%s password=%s'
			PUBLICATION test_pub
			WITH (connect = false)
		`, pgContainer.Config.Host, pgContainer.Config.Port, pgContainer.Config.User, pgContainer.Config.Password)

		// Create subscriptions with timeout
		err1 := pgContainer.ExecuteSQLWithTimeout(ctx, t, 5*time.Second, subscriptionSQL1)
		require.NoError(t, err1, "Failed to create first subscription")

		err2 := pgContainer.ExecuteSQLWithTimeout(ctx, t, 5*time.Second, subscriptionSQL2)
		require.NoError(t, err2, "Failed to create second subscription")

		// Create temporary configuration file using the container
		CreateTempPgctlConfig(t, pgContainer)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "list", "subscriptions", "--on", "testdb", "--all-databases")

		// Should succeed and show the subscriptions
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "👉 Will run on all databases on testdb")
		result.AssertStdoutContains(t, "✅ Subscriptions on testdb on database testdb:")
		result.AssertStdoutContains(t, "test_sub1")
		result.AssertStdoutContains(t, "test_sub2")
	})
}

func TestListSequences(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL containers for source and target
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	CreateTempPgctlConfig(t, pgContainer)

	t.Run("list_sequences_empty_database", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "list", "sequences", "--on", "testdb")

		result.AssertStdoutContains(t, "No sequences found in database")
	})

	t.Run("list_sequences_basic", func(t *testing.T) {
		pgContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_1 START 1")
		pgContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE test_seq_2 START 200")
		pgContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_1', 35, true)")
		pgContainer.ExecuteSQL(ctx, t, "SELECT setval('test_seq_2', 222, true)")

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "list", "sequences", "--on", "testdb")

		result.AssertStdoutContains(t, "Found 2 sequences in database")
		result.AssertStdoutContains(t, "test_seq_1 (last_value: 35)")
		result.AssertStdoutContains(t, "test_seq_2 (last_value: 222)")
	})

	t.Run("list_sequences_with_large_values", func(t *testing.T) {
		pgContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE large_seq_1 START 1")
		pgContainer.ExecuteSQL(ctx, t, "CREATE SEQUENCE large_seq_2 START 1000000")
		pgContainer.ExecuteSQL(ctx, t, "SELECT setval('large_seq_1', 999999999, true)")
		pgContainer.ExecuteSQL(ctx, t, "SELECT setval('large_seq_2', 2000000000, true)")

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "list", "sequences", "--on", "testdb")

		result.AssertStdoutContains(t, "Found 4 sequences in database")
		result.AssertStdoutContains(t, "large_seq_1 (last_value: 999999999)")
		result.AssertStdoutContains(t, "large_seq_2 (last_value: 2000000000)")
	})
}
