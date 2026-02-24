package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInitRelocation(t *testing.T) {
	ctx := context.Background()

	t.Run("init_relocation_golden_path", func(t *testing.T) {
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
		// Create temporary configuration file for both containers
		CreateTempPgctlConfigForPublisherSubscriber(t, publisher, subscriber)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "init", "relocation", "--from", "publisher", "--to", "subscriber", "--apply", "--no-ddl-confirmed")

		// Should succeed and create the subscription
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Relocation successfully initialized")

		// Verify the subscription was actually created
		pool := subscriber.CreatePgxPool(ctx, t)
		defer pool.Close()

		var subName string
		err := pool.QueryRow(ctx, "SELECT subname FROM pg_subscription WHERE subname = 'sub_subscriber_db'").Scan(&subName)
		require.NoError(t, err)
		require.Equal(t, "sub_subscriber_db", subName)

		// Verify the publication was actually created with all tables
		pool2 := publisher.CreatePgxPool(ctx, t)
		defer pool2.Close()

		var pubName string
		err = pool2.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_publisher_db'").Scan(&pubName)
		require.NoError(t, err)
		require.Equal(t, "pub_publisher_db", pubName)
	})
}

func TestRunRelocation(t *testing.T) {
	ctx := context.Background()

	t.Run("run_relocation_golden_path", func(t *testing.T) {
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
		// Create temporary configuration file for both containers
		CreateTempPgctlConfigForPublisherSubscriber(t, publisher, subscriber)

		executor := NewPgctlExecutor(t)
		pre := executor.Execute(ctx, t, "init", "relocation", "--from", "publisher", "--to", "subscriber", "--apply", "--no-ddl-confirmed")
		pre.AssertStdoutContains(t, "✅ Relocation successfully initialized")

		// Verify the subscription was actually created
		pool := subscriber.CreatePgxPool(ctx, t)
		defer pool.Close()

		var subName string
		err := pool.QueryRow(ctx, "SELECT subname FROM pg_subscription WHERE subname = 'sub_subscriber_db'").Scan(&subName)
		require.NoError(t, err)
		require.Equal(t, "sub_subscriber_db", subName)

		// Verify the publication was actually created with all tables
		pool2 := publisher.CreatePgxPool(ctx, t)
		defer pool2.Close()

		var pubName string
		err = pool2.QueryRow(ctx, "SELECT pubname FROM pg_publication WHERE pubname = 'pub_publisher_db'").Scan(&pubName)
		require.NoError(t, err)
		require.Equal(t, "pub_publisher_db", pubName)

		result := executor.Execute(ctx, t, "run", "relocation", "--from", "publisher", "--to", "subscriber", "--apply", "--no-ddl-confirmed", "--no-writes-confirmed")

		// Should succeed and create the subscription
		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Relocation successfully ended")
	})
}
