package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigShowCommand tests the pgctl config show functionality
func TestConfigShowCommand(t *testing.T) {
	ctx := context.Background()

	// Get project root for pgctl binary execution
	projectRoot := getProjectRoot(t)

	t.Run("show_config_with_valid_file", func(t *testing.T) {
		// Create a test configuration file
		configContent := `
production:
  database: test1
  host: pg.production.example.com
  password: secret123
  port: 5432
  role: readonly

staging:
  database: test2
  host: pg.staging.example.com
  password: staging_pass
  port: 5433
  role: developer
`
		CreateTempConfigWithContent(t, configContent)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "show")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "production:")
		result.AssertStdoutContains(t, "host: pg.production.example.com")
		result.AssertStdoutContains(t, "database: test1")
		result.AssertStdoutContains(t, "staging:")
		result.AssertStdoutContains(t, "host: pg.staging.example.com")
		result.AssertStdoutContains(t, "database: test2")
	})

	t.Run("show_config_without_file", func(t *testing.T) {
		// Ensure no config file exists
		configPath := filepath.Join(projectRoot, ".pgctl.yaml")
		err := os.Remove(configPath) // Remove if exists
		if err != nil && !os.IsNotExist(err) {
			require.NoError(t, err)
		}

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "show")

		// Should fail due to missing config file
		result.AssertFailure(t)
		result.AssertStdoutContains(t, "Config File \".pgctl\" Not Found")
	})

	t.Run("show_config_with_empty_file", func(t *testing.T) {
		// Create an empty configuration file
		CreateTempConfigWithContent(t, "")

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "show")

		// Should succeed but show no databases
		result.AssertSuccess(t)
		// Output should be minimal since no databases are configured
	})
}

// TestConfigFileFormats tests various configuration file formats and edge cases
func TestConfigFileFormats(t *testing.T) {
	ctx := context.Background()

	t.Run("config_with_special_characters", func(t *testing.T) {
		// Test configuration with special characters in values
		configContent := `
test_db:
  database: "test-db_name"
  host: test-host.example.com
  port: 5432
  role: "test-user"
  password: "p@ssw0rd!#$"
`
		CreateTempConfigWithContent(t, configContent)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "show")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "test_db:")
		result.AssertStdoutContains(t, "host: test-host.example.com")
		result.AssertStdoutContains(t, "database: test-db_name")
		result.AssertStdoutContains(t, "role: test-user")
	})

	t.Run("config_with_invalid_yaml", func(t *testing.T) {
		// Test configuration with invalid YAML syntax
		configContent := `
invalid_yaml:
  database: test
  host: localhost
  port: not_a_number
  role: user
    invalid_indent: value
`
		CreateTempConfigWithContent(t, configContent)

		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "show")

		// Should fail due to invalid YAML
		result.AssertFailure(t)
	})
}

// TestConfigInitHelp tests the help functionality for config init
func TestConfigInitHelp(t *testing.T) {
	ctx := context.Background()

	t.Run("config_init_help", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "init", "--help")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "Initialize a new configuration file")
		result.AssertStdoutContains(t, "Usage:")
		result.AssertStdoutContains(t, "pgctl config init")
	})

	t.Run("config_help_shows_init", func(t *testing.T) {
		executor := NewPgctlExecutor(t)
		result := executor.Execute(ctx, t, "config", "--help")

		result.AssertSuccess(t)
		result.AssertStdoutContains(t, "Manage the configuration")
		result.AssertStdoutContains(t, "init")
		result.AssertStdoutContains(t, "Initialize a new configuration file")
		result.AssertStdoutContains(t, "show")
		result.AssertStdoutContains(t, "Display the current configuration")
	})
}
