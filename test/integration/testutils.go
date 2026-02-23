package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Configuration constants to avoid repetition
const (
	// Standard test config with existing database
	TestConfigWithExistingDB = `
existing_db:
  database: existing
  host: localhost
  password: pass
  port: 5432
  role: user
`

	// Invalid config for testing connection failures
	TestConfigWithInvalidDB = `
invalid_db:
  database: nonexistent_db
  host: invalid-host-that-does-not-exist
  password: wrongpass
  port: 9999
  role: wronguser
`

	// Secure file permissions for config files (0600)
	SecureFilePerms = 0o600
)

// PostgreSQLContainer represents a test PostgreSQL container
type PostgreSQLContainer struct {
	Container     testcontainers.Container
	Config        PostgreSQLConfig
	NetworkName   string
	ContainerName string
}

// PostgreSQLConfig holds the configuration for connecting to PostgreSQL
type PostgreSQLConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// ConnectionString returns a formatted connection string for pgx
func (c *PostgreSQLConfig) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

// DSN returns a data source name for database/sql
func (c *PostgreSQLConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// InternalConnectionString returns a connection string using the container's internal hostname
// This is useful for subscriptions that need to connect within the container network
func (c *PostgreSQLConfig) InternalConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=%s",
		c.User, c.Password, 5432, c.Database, c.SSLMode)
}

// SetupPostgreSQLContainer creates and starts a PostgreSQL container for testing
func SetupPostgreSQLContainer(ctx context.Context, t *testing.T) *PostgreSQLContainer {
	t.Helper()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get container port")

	config := PostgreSQLConfig{
		Host:     host,
		Port:     port.Int(),
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	return &PostgreSQLContainer{
		Container: postgresContainer,
		Config:    config,
	}
}

// SetupPostgreSQLContainerWithWalLevel creates and starts a PostgreSQL container with specific WAL level
func SetupPostgreSQLContainerWithWalLevel(ctx context.Context, t *testing.T, walLevel string) *PostgreSQLContainer {
	t.Helper()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.Cmd = append(config.Cmd, "-c", fmt.Sprintf("wal_level=%s", walLevel))
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get container port")

	config := PostgreSQLConfig{
		Host:     host,
		Port:     port.Int(),
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	return &PostgreSQLContainer{
		Container: postgresContainer,
		Config:    config,
	}
}

// SetupPublisherSubscriberContainers creates two PostgreSQL containers for testing logical replication
// Returns publisher (source) and subscriber (target) containers
func SetupPublisherSubscriberContainers(ctx context.Context, t *testing.T) (*PostgreSQLContainer, *PostgreSQLContainer) {
	t.Helper()

	// Create a custom network for the containers to communicate
	dockerNetwork, err := network.New(ctx)
	require.NoError(t, err, "Failed to create network")

	// Create publisher container (source)
	publisherContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("publisher_db"),
		postgres.WithUsername("publisher_user"),
		postgres.WithPassword("publisher_pass"),
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.Cmd = append(config.Cmd, "-c", "wal_level=logical")
		}),
		network.WithNetworkName([]string{"publisher"}, dockerNetwork.Name),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start publisher PostgreSQL container")

	// Create subscriber container (target)
	subscriberContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("subscriber_db"),
		postgres.WithUsername("subscriber_user"),
		postgres.WithPassword("subscriber_pass"),
		testcontainers.WithConfigModifier(func(config *container.Config) {
			config.Cmd = append(config.Cmd, "-c", "wal_level=logical")
		}),
		network.WithNetworkName([]string{"subscriber"}, dockerNetwork.Name),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start subscriber PostgreSQL container")

	// Get publisher container name for internal networking
	publisherContainerName, err := publisherContainer.Name(ctx)
	require.NoError(t, err, "Failed to get publisher container name")

	// Get publisher connection details for external access
	pubHost, err := publisherContainer.Host(ctx)
	require.NoError(t, err, "Failed to get publisher container host")

	pubPort, err := publisherContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get publisher container port")

	publisherConfig := PostgreSQLConfig{
		Host:     pubHost,
		Port:     pubPort.Int(),
		Database: "publisher_db",
		User:     "publisher_user",
		Password: "publisher_pass",
		SSLMode:  "disable",
	}

	// Get subscriber connection details
	subHost, err := subscriberContainer.Host(ctx)
	require.NoError(t, err, "Failed to get subscriber container host")

	subPort, err := subscriberContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get subscriber container port")

	subscriberConfig := PostgreSQLConfig{
		Host:     subHost,
		Port:     subPort.Int(),
		Database: "subscriber_db",
		User:     "subscriber_user",
		Password: "subscriber_pass",
		SSLMode:  "disable",
	}

	publisher := &PostgreSQLContainer{
		Container:     publisherContainer,
		Config:        publisherConfig,
		NetworkName:   dockerNetwork.Name,
		ContainerName: publisherContainerName,
	}

	subscriber := &PostgreSQLContainer{
		Container:     subscriberContainer,
		Config:        subscriberConfig,
		NetworkName:   dockerNetwork.Name,
		ContainerName: "", // Will be set if needed
	}

	return publisher, subscriber
}

// CreatePgxPool creates a pgx connection pool using the container configuration
func (p *PostgreSQLContainer) CreatePgxPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(ctx, p.Config.ConnectionString())
	require.NoError(t, err, "Failed to create pgx pool")

	// Test the connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	return pool
}

// ExecuteSQL executes SQL statements on the container database
func (p *PostgreSQLContainer) ExecuteSQL(ctx context.Context, t *testing.T, query string, args ...interface{}) {
	t.Helper()

	pool := p.CreatePgxPool(ctx, t)
	defer pool.Close()

	_, err := pool.Exec(ctx, query, args...)
	require.NoError(t, err, "Failed to execute SQL: %s", query)
}

// ExecuteSQLWithTimeout executes SQL statements on the container database with a timeout
func (p *PostgreSQLContainer) ExecuteSQLWithTimeout(ctx context.Context, t *testing.T, timeout time.Duration, query string, args ...interface{}) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool := p.CreatePgxPool(ctx, t)
	defer pool.Close()

	_, err := pool.Exec(ctx, query, args...)
	return err
}

// Cleanup terminates the PostgreSQL container
func (p *PostgreSQLContainer) Cleanup(ctx context.Context, t *testing.T) {
	t.Helper()

	if p.Container != nil {
		err := p.Container.Terminate(ctx)
		if err != nil {
			// Only fail if the error is not about container already being removed
			if !strings.Contains(err.Error(), "No such container") {
				require.NoError(t, err, "Failed to terminate PostgreSQL container")
			}
		}
		p.Container = nil // Set to nil to prevent double cleanup
	}
}

// WaitForReadiness waits for the PostgreSQL container to be ready
func (p *PostgreSQLContainer) WaitForReadiness(ctx context.Context, t *testing.T, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for PostgreSQL to be ready")
		case <-ticker.C:
			pool, err := pgxpool.New(ctx, p.Config.ConnectionString())
			if err != nil {
				continue
			}

			err = pool.Ping(ctx)
			pool.Close()
			if err == nil {
				return // Database is ready
			}
		}
	}
}

// CreateTestTable creates a sample table for testing purposes
func (p *PostgreSQLContainer) CreateTestTable(ctx context.Context, t *testing.T) {
	t.Helper()

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS test_users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	p.ExecuteSQL(ctx, t, createTableSQL)
}

// InsertTestData inserts sample data into the test table
func (p *PostgreSQLContainer) InsertTestData(ctx context.Context, t *testing.T) {
	t.Helper()

	insertSQL := `
		INSERT INTO test_users (name, email) VALUES
		('John Doe', 'john.doe@example.com'),
		('Jane Smith', 'jane.smith@example.com'),
		('Alice Johnson', 'alice.johnson@example.com')
		ON CONFLICT (email) DO NOTHING;
	`

	p.ExecuteSQL(ctx, t, insertSQL)
}

// CreateTestExtensions creates test extensions that can be used for update testing
func (p *PostgreSQLContainer) CreateTestExtensions(ctx context.Context, t *testing.T) {
	t.Helper()

	type Extension struct {
		Name    string
		Version string
	}

	// Install some common extensions that are typically available
	extensions := []Extension{
		{
			Name:    "adminpack",
			Version: "1.1",
		},
		{
			Name:    "pg_trgm",
			Version: "1.5",
		},
	}

	for _, extension := range extensions {
		// First check if extension is available
		checkSQL := `
			SELECT COUNT(*) FROM pg_available_extensions
			WHERE name = $1 AND installed_version IS NULL;
		`

		pool := p.CreatePgxPool(ctx, t)
		defer pool.Close()

		var count int
		err := pool.QueryRow(ctx, checkSQL, extension.Name).Scan(&count)
		require.NoError(t, err)

		// Only install if extension is available and not already installed
		if count > 0 {
			installSQL := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\" WITH VERSION '%s';", extension.Name, extension.Version)
			p.ExecuteSQL(ctx, t, installSQL)
			t.Logf("Installed extension: %s with version %s", extension.Name, extension.Version)
		}
	}
}

// CreateExtensionWithOldVersion creates an extension and simulates it having an older version
// by manipulating the extension metadata (for testing purposes)
func (p *PostgreSQLContainer) CreateExtensionWithOldVersion(ctx context.Context, t *testing.T, extensionName string) {
	t.Helper()

	// Install the extension first
	installSQL := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\";", extensionName)
	p.ExecuteSQL(ctx, t, installSQL)

	// For testing purposes, we'll use the uuid-ossp extension which commonly has updates
	// We can check the current version and installed version
	t.Logf("Created extension %s for update testing", extensionName)
}

// VerifyExtensionVersion verifies that an extension has the expected installed version
func (p *PostgreSQLContainer) VerifyExtensionVersion(ctx context.Context, t *testing.T, extensionName, expectedVersion string) {
	t.Helper()

	pool := p.CreatePgxPool(ctx, t)
	defer pool.Close()

	var version string
	err := pool.QueryRow(ctx, `
		SELECT installed_version
		FROM pg_available_extensions
		WHERE name = $1
	`, extensionName).Scan(&version)
	require.NoError(t, err)
	require.Equal(t, expectedVersion, version, "%s extension should be at version %s", extensionName, expectedVersion)
	t.Logf("Verified extension %s is at version %s", extensionName, version)
}

// PgctlResult represents the result of executing pgctl binary
type PgctlResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// PgctlExecutor manages pgctl binary execution for integration tests
type PgctlExecutor struct {
	BinaryPath string
	WorkingDir string
	Env        []string
	Timeout    time.Duration
}

// NewPgctlExecutor creates a new pgctl executor with default settings
func NewPgctlExecutor(t *testing.T) *PgctlExecutor {
	t.Helper()

	// Get the project root directory
	projectRoot := getProjectRoot(t)

	return &PgctlExecutor{
		BinaryPath: filepath.Join(projectRoot, "bin", "pgctl"),
		WorkingDir: projectRoot,
		Env:        os.Environ(),
		Timeout:    30 * time.Second,
	}
}

// getProjectRoot finds the project root directory by looking for go.mod
func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current directory and walk up
	dir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	t.Fatal("Could not find project root directory (go.mod not found)")
	return ""
}

// Execute runs pgctl with the given arguments and returns the result
func (e *PgctlExecutor) Execute(ctx context.Context, t *testing.T, args ...string) *PgctlResult {
	t.Helper()

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, e.BinaryPath, args...) //nolint:gosec
	cmd.Dir = e.WorkingDir
	cmd.Env = e.Env

	t.Logf("Executing: %s %s", e.BinaryPath, strings.Join(args, " "))

	start := time.Now()
	stdout, stderr, exitCode := e.runCommand(cmd)
	duration := time.Since(start)

	result := &PgctlResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: duration,
	}

	t.Logf("pgctl execution completed in %v (exit code: %d)", duration, exitCode)
	if stdout != "" {
		t.Logf("stdout: %s", stdout)
	}
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}

	return result
}

// ExecuteWithInput runs pgctl with the given arguments and stdin input
func (e *PgctlExecutor) ExecuteWithInput(ctx context.Context, t *testing.T, stdin string, args ...string) *PgctlResult {
	t.Helper()

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, e.BinaryPath, args...) //nolint:gosec
	cmd.Dir = e.WorkingDir
	cmd.Env = e.Env

	// Set up stdin
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	t.Logf("Executing with input: %s %s", e.BinaryPath, strings.Join(args, " "))
	t.Logf("stdin: %s", stdin)

	start := time.Now()
	stdout, stderr, exitCode := e.runCommand(cmd)
	duration := time.Since(start)

	result := &PgctlResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: duration,
	}

	t.Logf("pgctl execution completed in %v (exit code: %d)", duration, exitCode)
	if stdout != "" {
		t.Logf("stdout: %s", stdout)
	}
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}

	return result
}

// runCommand executes the command and captures output
func (e *PgctlExecutor) runCommand(cmd *exec.Cmd) (stdout, stderr string, exitCode int) {
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1 // Default to 1 for other errors
		}
	} else {
		exitCode = 0
	}

	return stdout, stderr, exitCode
}

// WithTimeout sets a custom timeout for pgctl execution
func (e *PgctlExecutor) WithTimeout(timeout time.Duration) *PgctlExecutor {
	e.Timeout = timeout
	return e
}

// Helper methods for common assertions on PgctlResult

// AssertSuccess asserts that pgctl executed successfully (exit code 0)
func (r *PgctlResult) AssertSuccess(t *testing.T) *PgctlResult {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "Expected pgctl to succeed\nstdout: %s\nstderr: %s", r.Stdout, r.Stderr)
	return r
}

// AssertFailure asserts that pgctl failed (non-zero exit code)
func (r *PgctlResult) AssertFailure(t *testing.T) *PgctlResult {
	t.Helper()
	require.NotEqual(t, 0, r.ExitCode, "Expected pgctl to fail\nstdout: %s\nstderr: %s", r.Stdout, r.Stderr)
	return r
}

// AssertExitCode asserts the specific exit code
func (r *PgctlResult) AssertExitCode(t *testing.T, expectedCode int) *PgctlResult {
	t.Helper()
	require.Equal(t, expectedCode, r.ExitCode, "Expected exit code %d\nstdout: %s\nstderr: %s", expectedCode, r.Stdout, r.Stderr)
	return r
}

// AssertStdoutContains asserts that stdout contains the given text
func (r *PgctlResult) AssertStdoutContains(t *testing.T, text string) *PgctlResult {
	t.Helper()
	require.Contains(t, r.Stdout, text, "Expected stdout to contain '%s'\nstdout: %s", text, r.Stdout)
	return r
}

// AssertStdoutContainsOrContains asserts that stdout contains at least one of the given texts
func (r *PgctlResult) AssertStdoutContainsOrContains(t *testing.T, text string, text2 string) *PgctlResult {
	t.Helper()
	containsText := strings.Contains(r.Stdout, text)
	containsText2 := strings.Contains(r.Stdout, text2)
	require.True(t, containsText || containsText2,
		"Expected stdout to contain either '%s' or '%s'\nstdout: %s",
		text, text2, r.Stdout)
	return r
}

// AssertStderrContains asserts that stderr contains the given text
func (r *PgctlResult) AssertStderrContains(t *testing.T, text string) *PgctlResult {
	t.Helper()
	require.Contains(t, r.Stderr, text, "Expected stderr to contain '%s'\nstderr: %s", text, r.Stderr)
	return r
}

// AssertStdoutEmpty asserts that stdout is empty
func (r *PgctlResult) AssertStdoutEmpty(t *testing.T) *PgctlResult {
	t.Helper()
	require.Empty(t, strings.TrimSpace(r.Stdout), "Expected stdout to be empty\nstdout: %s", r.Stdout)
	return r
}

// AssertStderrEmpty asserts that stderr is empty
func (r *PgctlResult) AssertStderrEmpty(t *testing.T) *PgctlResult {
	t.Helper()
	require.Empty(t, strings.TrimSpace(r.Stderr), "Expected stderr to be empty\nstderr: %s", r.Stderr)
	return r
}

// AssertDurationLessThan asserts that execution took less than the given duration
func (r *PgctlResult) AssertDurationLessThan(t *testing.T, maxDuration time.Duration) *PgctlResult {
	t.Helper()
	require.Less(t, r.Duration, maxDuration, "Expected execution to take less than %v, took %v", maxDuration, r.Duration)
	return r
}

// Convenience functions for quick pgctl execution

// ExecutePgctl is a convenience function to quickly execute pgctl with default settings
func ExecutePgctl(ctx context.Context, t *testing.T, args ...string) *PgctlResult {
	t.Helper()
	executor := NewPgctlExecutor(t)
	return executor.Execute(ctx, t, args...)
}

// ExecutePgctlWithInput is a convenience function to execute pgctl with stdin input
func ExecutePgctlWithInput(ctx context.Context, t *testing.T, stdin string, args ...string) *PgctlResult {
	t.Helper()
	executor := NewPgctlExecutor(t)
	return executor.ExecuteWithInput(ctx, t, stdin, args...)
}

// createTempPgctlConfig creates a temporary .pgctl.yaml configuration file
// using the provided PostgreSQL container configuration
func CreateTempPgctlConfig(t *testing.T, pgContainer *PostgreSQLContainer) string {
	t.Helper()

	// Get project root
	projectRoot := getProjectRoot(t)
	configPath := filepath.Join(projectRoot, ".pgctl.yaml")

	// Create configuration content
	configContent := fmt.Sprintf(`testdb:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		pgContainer.Config.Database,
		pgContainer.Config.Host,
		pgContainer.Config.Password,
		pgContainer.Config.Port,
		pgContainer.Config.User,
	)

	// Write configuration file
	err := os.WriteFile(configPath, []byte(configContent), SecureFilePerms)
	require.NoError(t, err, "Failed to create temporary pgctl config file")

	t.Logf("Created temporary config file: %s", configPath)
	t.Logf("Config content:\n%s", configContent)

	return configPath
}

func CreateTempConfigWithContent(t *testing.T, configContent string) string {
	t.Helper()

	projectRoot := getProjectRoot(t)
	configPath := filepath.Join(projectRoot, ".pgctl.yaml")

	err := os.WriteFile(configPath, []byte(configContent), SecureFilePerms)
	require.NoError(t, err, "Failed to create temporary pgctl config file")

	t.Logf("Created temporary config file: %s", configPath)
	t.Logf("Config content:\n%s", configContent)

	return configPath
}

// CreateTempPgctlConfigForPublisherSubscriber creates a configuration file for publisher-subscriber testing
func CreateTempPgctlConfigForPublisherSubscriber(t *testing.T, publisher, subscriber *PostgreSQLContainer) string {
	t.Helper()

	// Get project root
	projectRoot := getProjectRoot(t)
	configPath := filepath.Join(projectRoot, ".pgctl.yaml")

	// Create configuration content with both publisher and subscriber
	configContent := fmt.Sprintf(`publisher:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
subscriber:
  database: %s
  host: %s
  password: %s
  port: %d
  role: %s
`,
		publisher.Config.Database,
		publisher.Config.Host,
		publisher.Config.Password,
		publisher.Config.Port,
		publisher.Config.User,
		subscriber.Config.Database,
		subscriber.Config.Host,
		subscriber.Config.Password,
		subscriber.Config.Port,
		subscriber.Config.User,
	)

	// Write configuration file
	err := os.WriteFile(configPath, []byte(configContent), SecureFilePerms)
	require.NoError(t, err, "Failed to create temporary pgctl config file")

	t.Logf("Created temporary config file: %s", configPath)
	t.Logf("Config content:\n%s", configContent)

	return configPath
}

// SetupPostgreSQLContainerWithVersion creates and starts a PostgreSQL container with a specific version for testing
func SetupPostgreSQLContainerWithVersion(ctx context.Context, t *testing.T, image string) *PostgreSQLContainer {
	t.Helper()

	// Create PostgreSQL container with specified image
	postgresContainer, err := postgres.Run(ctx,
		image,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get container host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get container port")

	config := PostgreSQLConfig{
		Host:     host,
		Port:     port.Int(),
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	return &PostgreSQLContainer{
		Container: postgresContainer,
		Config:    config,
	}
}
