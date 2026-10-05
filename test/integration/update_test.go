package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpdateExtensionsUpgrade(t *testing.T) {
	ctx := context.Background()

	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	// Wait for the container to be ready
	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	// Create test extensions:
	//  - btree_gist @ 1.6 — minor-updatable, must be applied
	//  - pgctl_bump @ 1.0 — synthetic major bump to 2.0, must be listed as
	//    "Ignored because it needs a major update" unless --include-major-versions
	pgContainer.CreateTestExtensions(ctx, t)

	// Sanity-check the initial state so failures below can be attributed to
	// pgctl behaviour rather than the install helpers.
	pgContainer.VerifyExtensionVersion(ctx, t, "btree_gist", "1.6")
	pgContainer.VerifyExtensionVersion(ctx, t, "pgctl_bump", "1.0")

	// Create temporary configuration file using the container
	CreateTempPgctlConfig(t, pgContainer)

	t.Run("update_extensions_dry_run", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "Retrieving updatable extensions")
		result.AssertStdoutContains(t, "✅ Updatable extension found btree_gist : 1.6 ➚ 1.8")
		result.AssertStdoutContains(t, "| Ignored extension because it needs a major update pgctl_bump : 1.0 ➚ 2.0")
	})

	t.Run("update_extensions_all_databases", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--all-databases")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "👉 Will run on all databases on testdb")
		result.AssertStdoutContains(t, "Retrieving updatable extensions on testdb for testdb")
		result.AssertStdoutContains(t, "✅ Updatable extension found btree_gist : 1.6 ➚ 1.8")
		result.AssertStdoutContains(t, "| Ignored extension because it needs a major update pgctl_bump : 1.0 ➚ 2.0")
		result.AssertStdoutContains(t, "Retrieving updatable extensions on testdb for postgres")
		result.AssertStdoutContains(t, "No extensions to update for postgres")
	})

	t.Run("update_extensions_apply", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		applyResult := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--apply")

		applyResult.AssertSuccess(t)
		require.NotContains(t, applyResult.Stdout, "🚧 DRY RUN MODE ACTIVATED 🚧")
		applyResult.AssertStdoutContains(t, "Retrieving updatable extensions")
		applyResult.AssertStdoutContains(t, "✅ Updatable extension found btree_gist : 1.6 ➚ 1.8")
		applyResult.AssertStdoutContains(t, "| Ignored extension because it needs a major update pgctl_bump : 1.0 ➚ 2.0")
		applyResult.AssertStdoutContains(t, "| Updating on testdb for testdb: [btree_gist]")
		applyResult.AssertStdoutContains(t, "✅ All extensions updated on testdb for testdb : [btree_gist]")

		// Verify the minor-upgradable extension was actually updated to its
		// latest version and pgctl_bump stayed at 1.0 (ignored because its
		// available bump is a major-version one).
		pgContainer.VerifyExtensionVersion(ctx, t, "btree_gist", "1.8")
		pgContainer.VerifyExtensionVersion(ctx, t, "pgctl_bump", "1.0")
	})
}

// TestUpdateExtensionsMajorVersionUpgrade exercises the `--include-major-versions`
// code path, which only diverges from the default when an installed extension
// has a major (1.x → 2.x) upgrade available. This is verified against a
// synthetic pgctl_bump extension shipped under testdata/pgctl_bump, so the test
// runs against the default PostgreSQL image and does not depend on adminpack
// (which was removed in PG 17).
func TestUpdateExtensionsMajorVersionUpgrade(t *testing.T) {
	ctx := context.Background()

	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	pgContainer.InstallPgctlBumpExtension(ctx, t)
	pgContainer.VerifyExtensionVersion(ctx, t, "pgctl_bump", "1.0")

	CreateTempPgctlConfig(t, pgContainer)

	t.Run("update_extensions_apply_major_versions", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--include-major-versions", "--apply")

		result.AssertSuccess(t)
		require.NotContains(t, result.Stdout, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "Retrieving updatable extensions")
		result.AssertStdoutContains(t, "✅ Updatable extension found pgctl_bump : 1.0 ➚ 2.0")
		result.AssertStdoutContains(t, "| Updating on testdb for testdb: [pgctl_bump]")
		result.AssertStdoutContains(t, "✅ All extensions updated on testdb for testdb : [pgctl_bump]")

		pgContainer.VerifyExtensionVersion(ctx, t, "pgctl_bump", "2.0")
	})

	t.Run("update_extensions_default_ignores_major", func(t *testing.T) {
		// Re-create the container so pgctl_bump is back at 1.0 for this subtest.
		fresh := SetupPostgreSQLContainer(ctx, t)
		defer fresh.Cleanup(ctx, t)
		fresh.WaitForReadiness(ctx, t, 30*time.Second)
		fresh.InstallPgctlBumpExtension(ctx, t)
		fresh.VerifyExtensionVersion(ctx, t, "pgctl_bump", "1.0")
		CreateTempPgctlConfig(t, fresh)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "| Ignored extension because it needs a major update pgctl_bump : 1.0 ➚ 2.0")
	})
}

// TestUpdateExtensionsCombinedFlags checks the full behaviour of the
// --all-databases + --include-major-versions + --apply combo against a mix of:
//   - btree_gist @ 1.6 — minor bump available (1.6 → 1.8)
//   - pgctl_bump  @ 1.0 — major bump available (1.0 → 2.0)
//
// Both must be upgraded to their latest versions, and neither must appear in
// an "Ignored because it needs a major update" line since --include-major-versions
// is passed.
func TestUpdateExtensionsCombinedFlags(t *testing.T) {
	ctx := context.Background()

	pgContainer := SetupPostgreSQLContainer(ctx, t)
	defer pgContainer.Cleanup(ctx, t)

	pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

	pgContainer.CreateTestExtensions(ctx, t)
	pgContainer.VerifyExtensionVersion(ctx, t, "btree_gist", "1.6")
	pgContainer.VerifyExtensionVersion(ctx, t, "pgctl_bump", "1.0")

	CreateTempPgctlConfig(t, pgContainer)

	t.Run("update_extensions_combined_flags", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "update", "extensions", "--on", "testdb", "--all-databases", "--include-major-versions", "--apply")

		result.AssertSuccess(t)
		require.NotContains(t, result.Stdout, "🚧 DRY RUN MODE ACTIVATED 🚧")
		result.AssertStdoutContains(t, "👉 Will run on all databases on testdb")
		result.AssertStdoutContains(t, "Retrieving updatable extensions")
		result.AssertStdoutContains(t, "✅ Updatable extension found btree_gist : 1.6 ➚ 1.8")
		result.AssertStdoutContains(t, "✅ Updatable extension found pgctl_bump : 1.0 ➚ 2.0")
		// --include-major-versions must suppress the "ignored" branch entirely.
		require.NotContains(t, result.Stdout, "Ignored extension because it needs a major update")

		pgContainer.VerifyExtensionVersion(ctx, t, "btree_gist", "1.8")
		pgContainer.VerifyExtensionVersion(ctx, t, "pgctl_bump", "2.0")
	})
}
