package integration

import (
	"context"
	"testing"
	"time"
)

// TestPingCommand tests the pgctl ping functionality with a real PostgreSQL container
func TestPingCommand(t *testing.T) {
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

	// Test successful ping
	t.Run("successful_ping", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "ping", "testdb")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "✅ Successful connection for alias testdb")
	})

	// Test ping multiple aliases
	t.Run("ping_multiple_aliases", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "ping", "testdb", "nonexistent")

		result.AssertFailure(t)
		result.AssertStdoutContains(t, "✅ Successful connection for alias testdb\n❌ Alias nonexistent does not exist. Add it to your configuration file and retry.")
	})
}
