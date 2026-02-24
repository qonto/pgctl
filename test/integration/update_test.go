package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpdateExtensionsUpgrade(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test extensions
	pgContainer.CreateTestExtensions(ctx, t)

	// Create temporary configuration file using the container
	CreateTempPgctlConfig(t, pgContainer)

	t.Run("update_extensions_dry_run", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "Retrieving updatable extensions")
		result.AssertStdoutContains(t, "Ignored extension because it needs a major update adminpack : 1.1 ➚ 2.1")
		result.AssertStdoutContains(t, "✅ Updatable extension found pg_trgm : 1.5 ➚ 1.6")
	})

	t.Run("update_extensions_all_databases", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--all-databases")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "👉 Will run on all databases on testdb")
		result.AssertStdoutContains(t, "Retrieving updatable extensions on testdb for testdb")
		result.AssertStdoutContains(t, "Ignored extension because it needs a major update adminpack : 1.1 ➚ 2.1")
		result.AssertStdoutContains(t, "✅ Updatable extension found pg_trgm : 1.5 ➚ 1.6")
		result.AssertStdoutContains(t, "Retrieving updatable extensions on testdb for postgres")
		result.AssertStdoutContains(t, "No extensions to update for postgres")
		result.AssertStdoutContains(t, "Retrieving updatable extensions on testdb for testdb")
		result.AssertStdoutContains(t, "Ignored extension because it needs a major update adminpack : 1.1 ➚ 2.1")
		result.AssertStdoutContains(t, "✅ Updatable extension found pg_trgm : 1.5 ➚ 1.6")
	})

	t.Run("update_extensions_apply", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		applyResult := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--apply")

		applyResult.AssertSuccess(t)
		require.NotContains(t, applyResult.Stdout, "🚧 DRY RUN MODE ACTIVATED 🚧")
		applyResult.AssertStdoutContains(t, "Retrieving updatable extensions")
		applyResult.AssertStdoutContains(t, "Ignored extension because it needs a major update adminpack : 1.1 ➚ 2.1")
		applyResult.AssertStdoutContains(t, "✅ Updatable extension found pg_trgm : 1.5 ➚ 1.6")
		applyResult.AssertStdoutContains(t, "| Updating on testdb for testdb: [pg_trgm]")
		applyResult.AssertStdoutContains(t, "✅ All extensions updated on testdb for testdb : [pg_trgm]")

		// Verify the extension was actually updated
		pgContainer.VerifyExtensionVersion(ctx, t, "pg_trgm", "1.6")
	})
}

func TestUpdateExtensionsMajorVersionUpgrade(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test extensions
	pgContainer.CreateTestExtensions(ctx, t)

	// Create temporary configuration file using the container
	CreateTempPgctlConfig(t, pgContainer)

	t.Run("update_extensions_apply_major_versions", func(t *testing.T) {
		// Test with --include-major-versions flag
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--include-major-versions", "--apply")

		result.AssertSuccess(t)
		require.NotContains(t, result.Stdout, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "Retrieving updatable extensions")
		result.AssertStdoutContains(t, "✅ Updatable extension found adminpack : 1.1 ➚ 2.1")
		result.AssertStdoutContainsOrContains(t,
			"| Updating on testdb for testdb: [adminpack pg_trgm]",
			"| Updating on testdb for testdb: [pg_trgm adminpack]")
		result.AssertStdoutContainsOrContains(t,
			"✅ All extensions updated on testdb for testdb : [adminpack pg_trgm]",
			"✅ All extensions updated on testdb for testdb : [pg_trgm adminpack]")

		// Verify adminpack was updated (pg_trgm was already updated in previous test)
		pgContainer.VerifyExtensionVersion(ctx, t, "adminpack", "2.1")
		pgContainer.VerifyExtensionVersion(ctx, t, "pg_trgm", "1.6")
	})
}

func TestUpdateExtensionsCombinedFlags(t *testing.T) {
	ctx := context.Background()

	// Setup PostgreSQL container
	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test extensions
	pgContainer.CreateTestExtensions(ctx, t)

	// Create temporary configuration file using the container
	CreateTempPgctlConfig(t, pgContainer)

	t.Run("update_extensions_combined_flags", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--all-databases", "--include-major-versions", "--apply")

		result.AssertSuccess(t)
		// Should not show dry run mode when --apply is used
		require.NotContains(t, result.Stdout, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "👉 Will run on all databases on testdb")
		result.AssertStdoutContains(t, "Retrieving updatable extensions")
		// By this point, all extensions should already be updated in previous tests
		result.AssertStdoutContains(t, "No extensions to update for testdb")

		// Verify final state - both extensions should be at their latest versions
		pgContainer.VerifyExtensionVersion(ctx, t, "adminpack", "2.1")
		pgContainer.VerifyExtensionVersion(ctx, t, "pg_trgm", "1.6")
	})
}
