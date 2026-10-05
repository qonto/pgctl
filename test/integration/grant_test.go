package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type QueryRolesCanConnectCurrent struct {
	Grantee string
}

const TESTING_ROLE = "testgrant"

func TestGrantRevokeCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create temporary configuration file
	CreateTempPgctlConfig(t, pgContainer)

	// Create a user
	err := pgContainer.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, "CREATE ROLE testgrant WITH LOGIN PASSWORD 'testpass'")
	require.NoError(t, err, "Failed to create role")

	// Add necessary grants
	err = pgContainer.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, "GRANT CONNECT ON DATABASE testdb TO testgrant;")
	require.NoError(t, err)
	err = pgContainer.ExecuteSQLWithTimeout(ctx, t, 10*time.Second, "REVOKE CONNECT ON DATABASE testdb FROM public;")
	require.NoError(t, err)

	// Test revoke dryrun
	t.Run("dryrun_revoke", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "revoke", "connect", "--on", "testdb", "--role", "testgrant")

		result.AssertSuccess(t)

		// Verify the grant was actually revoked
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		rows, err := pool.Query(ctx, `SELECT s.grantee::regrole as Grantee FROM pg_database,
LATERAL aclexplode(case when datacl is null then acldefault('d', datdba) else
datacl end) s WHERE privilege_type='CONNECT' and datname = current_database()`)
		require.NoError(t, err)

		rolesThatCanConnect, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryRolesCanConnectCurrent])
		require.NoError(t, err)

		revoked := true
		for _, d := range rolesThatCanConnect {
			if d.Grantee == TESTING_ROLE {
				revoked = false
			}
		}

		require.Equal(t, false, revoked)
	})

	// Test successful revoke
	t.Run("successful_revoke", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "revoke", "connect", "--on", "testdb", "--role", "testgrant", "--apply")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ connect revoked")

		// Verify the grant was actually revoked
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		rows, err := pool.Query(ctx, `SELECT s.grantee::regrole as Grantee FROM pg_database,
LATERAL aclexplode(case when datacl is null then acldefault('d', datdba) else
datacl end) s WHERE privilege_type='CONNECT' and datname = current_database()`)
		require.NoError(t, err)

		rolesThatCanConnect, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryRolesCanConnectCurrent])
		require.NoError(t, err)

		revoked := true
		for _, d := range rolesThatCanConnect {
			if d.Grantee == TESTING_ROLE {
				revoked = false
			}
		}

		require.Equal(t, true, revoked)
	})

	// Test grant dryrun
	t.Run("dryrun_grant", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "grant", "connect", "--on", "testdb", "--role", "testgrant")

		result.AssertSuccess(t)

		// Verify the grant was actually granted back
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		rows, err := pool.Query(ctx, `SELECT s.grantee::regrole as Grantee FROM pg_database,
LATERAL aclexplode(case when datacl is null then acldefault('d', datdba) else
datacl end) s WHERE privilege_type='CONNECT' and datname = current_database()`)
		require.NoError(t, err)

		rolesThatCanConnect, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryRolesCanConnectCurrent])
		require.NoError(t, err)

		granted := false
		for _, d := range rolesThatCanConnect {
			if d.Grantee == TESTING_ROLE {
				granted = true
			}
		}

		require.Equal(t, false, granted)
	})

	// Test successful grant
	t.Run("successful_grant", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "grant", "connect", "--on", "testdb", "--role", "testgrant", "--apply")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ connect granted")

		// Verify the grant was actually granted back
		pool := pgContainer.CreatePgxPool(ctx, t)
		defer pool.Close()

		rows, err := pool.Query(ctx, `SELECT s.grantee::regrole as Grantee FROM pg_database,
LATERAL aclexplode(case when datacl is null then acldefault('d', datdba) else
datacl end) s WHERE privilege_type='CONNECT' and datname = current_database()`)
		require.NoError(t, err)

		rolesThatCanConnect, err := pgx.CollectRows(rows, pgx.RowToStructByName[QueryRolesCanConnectCurrent])
		require.NoError(t, err)

		granted := false
		for _, d := range rolesThatCanConnect {
			if d.Grantee == TESTING_ROLE {
				granted = true
			}
		}

		require.Equal(t, true, granted)
	})
}
