package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDropSubscription tests the complete subscription drop functionality
// Note: In the test environment, actual subscription drops will fail because PostgreSQL
// tries to connect to the publisher to clean up replication slots, but the test containers
// cannot communicate properly. That's why the golden path can't be properly tested.
func TestDropSubscription(t *testing.T) {
	ctx := context.Background()

	t.Run("drop_subscription_dry_run", func(t *testing.T) {
		// Setup a single PostgreSQL container with logical WAL level
		pgContainer := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer pgContainer.Cleanup(ctx, t)

		// Wait for container to be ready
		pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges
		pgContainer.ExecuteSQL(ctx, t, "ALTER ROLE testuser REPLICATION")

		// Create a second database for the subscription to connect to
		pgContainer.ExecuteSQL(ctx, t, "CREATE DATABASE source_db")

		// Create test table and publication in the source database
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

		// Verify the subscription was created
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		var subName string
		err = pool.QueryRow(ctx, "SELECT subname FROM pg_subscription WHERE subname = 'test_sub'").Scan(&subName)
		require.NoError(t, err)
		require.Equal(t, "test_sub", subName)

		// Now drop the subscription in dry run mode
		executor := NewPgctlExecutor(t)
		dropResult := executor.Execute(ctx, t, "drop", "subscription", "--on", "testdb", "--name", "test_sub")

		// Should succeed in dry run mode
		dropResult.AssertSuccess(t)
		dropResult.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		dropResult.AssertStdoutContains(t, "👉 Subscription test_sub would be dropped on target database testdb in testdb")

		// Verify the subscription was NOT actually dropped
		err = pool.QueryRow(ctx, "SELECT subname FROM pg_subscription WHERE subname = 'test_sub'").Scan(&subName)
		require.NoError(t, err, "Subscription should still exist in dry run mode")
		require.Equal(t, "test_sub", subName)
	})

	t.Run("drop_subscription_does_not_exist", func(t *testing.T) {
		// Setup subscriber container only
		subscriber := SetupPostgreSQLContainerWithWalLevel(ctx, t, "logical")
		defer subscriber.Cleanup(ctx, t)

		// Wait for container to be ready
		subscriber.WaitForReadiness(ctx, t, 30*time.Second)

		// Grant replication privileges
		subscriber.ExecuteSQL(ctx, t, "ALTER ROLE testuser REPLICATION")

		// Create temporary configuration file for subscriber only
		CreateTempPgctlConfig(t, subscriber)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "drop", "subscription", "--on", "testdb", "--name", "nonexistent_subscription", "--apply")

		// Should fail because the subscription doesn't exist
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "❌ Failed to drop subscription")
	})
}
