# Integration Tests

This directory contains comprehensive integration tests for the pgctl CLI application using PostgreSQL containers with TestContainers and real binary execution.

## Overview

The integration test suite provides complete end-to-end testing of pgctl commands with:
- **Real PostgreSQL containers** for authentic database operations
- **Actual pgctl binary execution** for true CLI integration testing
- **Coverage collection from binary execution** using Go 1.20+ instrumentation
- **Individual test files per command** for organized and maintainable tests
- **Comprehensive error handling and edge case testing**

## Test Architecture

### Current Test Structure

```
test/integration/
├── README.md              # This documentation
├── main_test.go          # Test suite orchestration and global setup
├── testutils.go          # Core utilities and infrastructure
├── ping_test.go          # Connection testing
├── config_test.go        # Configuration management
├── update_test.go        # Extension updates (with real upgrades)
├── list_test.go          # Resource listing (extensions, tables, etc.)
├── create_test.go        # Resource creation (publications, subscriptions)
├── check_test.go         # Database validation commands
├── copy_test.go          # Schema and data copying
├── drop_test.go          # Resource deletion
├── relocation_test.go    # Relocation full pipeline
```

## Core Infrastructure

### PostgreSQL Container Management (`testutils.go`)

```go
// Core container struct
type PostgreSQLContainer struct {
    Container testcontainers.Container
    Config    PostgreSQLConfig
}

// Main setup function - creates and starts PostgreSQL 16-alpine
func SetupPostgreSQLContainer(ctx context.Context, t *testing.T) *PostgreSQLContainer

// Connection management
func (p *PostgreSQLContainer) CreatePgxPool(ctx context.Context, t *testing.T) *pgxpool.Pool
func (p *PostgreSQLContainer) ExecuteSQL(ctx context.Context, t *testing.T, sql string)
func (p *PostgreSQLContainer) WaitForReadiness(ctx context.Context, t *testing.T, timeout time.Duration)

// Cleanup
func (p *PostgreSQLContainer) Cleanup(ctx context.Context, t *testing.T)
```

### Pgctl Binary Execution (`testutils.go`)

```go
// Binary execution with result capture
type PgctlExecutor struct {
    timeout     time.Duration
    environment map[string]string
    workingDir  string
}

type PgctlResult struct {
    ExitCode  int
    Stdout    string
    Stderr    string
    Duration  time.Duration
}

// Main execution methods
func NewPgctlExecutor(t *testing.T) *PgctlExecutor
func (e *PgctlExecutor) Execute(ctx context.Context, t *testing.T, args ...string) *PgctlResult

// Fluent assertion API
func (r *PgctlResult) AssertSuccess(t *testing.T) *PgctlResult
func (r *PgctlResult) AssertFailure(t *testing.T) *PgctlResult
func (r *PgctlResult) AssertStdoutContains(t *testing.T, expected string) *PgctlResult
func (r *PgctlResult) AssertStderrContains(t *testing.T, expected string) *PgctlResult
```

### Configuration Management (`testutils.go`)

```go
// Creates temporary .pgctl.yaml files for testing
func CreateTempPgctlConfig(t *testing.T, pgContainer *PostgreSQLContainer) string

// Test data and schema setup
func (p *PostgreSQLContainer) CreateTestTable(ctx context.Context, t *testing.T)
func (p *PostgreSQLContainer) InsertTestData(ctx context.Context, t *testing.T)
func (p *PostgreSQLContainer) CreateTestExtensions(ctx context.Context, t *testing.T)
func (p *PostgreSQLContainer) VerifyExtensionVersion(ctx context.Context, t *testing.T, name, version string)
```

## Running Tests

### Prerequisites

1. **Docker**: TestContainers requires Docker to be installed and running
2. **Go**: Go 1.20 or later (for coverage instrumentation)
3. **Make**: For building the pgctl binary

### Quick Start

```bash
# Build binary and run all integration tests with coverage
make test-integration

# Run specific test file
go test -v ./test/integration/ping_test.go ./test/integration/testutils.go ./test/integration/main_test.go

# Run specific test function
go test -v ./test/integration/... -run TestPingCommand
# or
go test ./test/integration -v -run TestCheckSubscriptionLag

# Run tests for specific command
go test -v ./test/integration/update_test.go ./test/integration/testutils.go ./test/integration/main_test.go
```

### Coverage Collection

The enhanced test system automatically:
1. **Builds pgctl binary with coverage instrumentation** (`-cover` flag)
2. **Collects coverage from actual binary execution** (via `GOCOVERDIR`)
3. **Generates both text and HTML reports**
4. **Measures real code paths executed by CLI commands**

```bash
# Run with coverage (automatic with make test-integration)
make test-integration

# View coverage report
open coverage/coverage-integrations-tests.html

# Check coverage percentage
go tool cover -func=coverage/coverage-integrations-tests.out | tail -1
```

### Test Execution Options

```bash
# Run tests in parallel (when safe)
go test -v ./test/integration/... -parallel 4

# Run with race detection
go test -v ./test/integration/... -race

# Skip slow tests
go test -v ./test/integration/... -short

# Run with timeout
go test -v ./test/integration/... -timeout 10m
```

## Adding New Tests

### 1. For New Commands or Enhanced Functionality

Create a new test file following the pattern `{command}_test.go`:
```go
package integration

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

// Help functionality
func TestMyCommandHelp(t *testing.T) {
    ctx := context.Background()

    t.Run("mycommand_help", func(t *testing.T) {
        executor := NewPgctlExecutor(t)
        result := executor.Execute(ctx, t, "mycommand", "--help")

        result.AssertSuccess(t)
        result.AssertStdoutContains(t, "Expected help text")
    })
}

// Validation testing
func TestMyCommandValidation(t *testing.T) {
    ctx := context.Background()

    t.Run("missing_required_flag", func(t *testing.T) {
        executor := NewPgctlExecutor(t)
        result := executor.Execute(ctx, t, "mycommand", "subcommand")

        result.AssertFailure(t)
        result.AssertStderrContains(t, "required flag")
    })
}

// Command execution testing
func TestMyCommandExecution(t *testing.T) {
    ctx := context.Background()

    pgContainer := SetupPostgreSQLContainer(ctx, t)
    defer pgContainer.Cleanup(ctx, t)
    pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

    CreateTempPgctlConfig(t, pgContainer)

    executor := NewPgctlExecutor(t)
    result := executor.Execute(ctx, t, "mycommand", "subcommand", "--on", "testdb")
    // Test the actual command behavior
    if result.ExitCode == 0 {
        result.AssertStdoutContains(t, "Expected success message")
        // Add database verification if needed
    } else {
        t.Logf("Command failed with: %s", result.Stderr)
        // May be expected for commands under development
    }
}
```

### 2. For Commands with Database Operations

Add real functionality testing with database verification:

```go
func TestMyCommandWithDatabase(t *testing.T) {
    ctx := context.Background()

    // Setup real database
    pgContainer := SetupPostgreSQLContainer(ctx, t)
    defer pgContainer.Cleanup(ctx, t)
    pgContainer.WaitForReadiness(ctx, t, 30*time.Second)

    // Setup test data
    pgContainer.CreateTestTable(ctx, t)
    pgContainer.InsertTestData(ctx, t)

    CreateTempPgctlConfig(t, pgContainer)

    t.Run("mycommand_success", func(t *testing.T) {
        executor := NewPgctlExecutor(t)
        result := executor.Execute(ctx, t, "mycommand", "--on", "testdb")

        result.AssertSuccess(t)
        result.AssertStdoutContains(t, "Expected success message")

        // Verify database state
        pool := pgContainer.CreatePgxPool(ctx, t)
        defer pool.Close()

        var count int
        err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM expected_table").Scan(&count)
        require.NoError(t, err)
        require.Equal(t, expectedCount, count)
    })
}
```

### 3. Test Utilities

Add helper functions to `testutils.go` when needed:

```go
// Example: Add helper for specific test data setup
func (p *PostgreSQLContainer) CreateMyTestData(ctx context.Context, t *testing.T) {
    t.Helper()

    sql := `
        INSERT INTO my_table (col1, col2) VALUES
        ('value1', 'value2'),
        ('value3', 'value4');
    `
    p.ExecuteSQL(ctx, t, sql)
    t.Log("Created test data for my feature")
}

// Example: Add verification helper
func (p *PostgreSQLContainer) VerifyMyState(ctx context.Context, t *testing.T, expected string) {
    t.Helper()

    pool := p.CreatePgxPool(ctx, t)
    defer pool.Close()

    var actual string
    err := pool.QueryRow(ctx, "SELECT state FROM my_table WHERE id = 1").Scan(&actual)
    require.NoError(t, err)
    require.Equal(t, expected, actual, "State verification failed")
}
```

## Test Organization Patterns

### Individual Test Functions

Each command has individual test functions to enable focused testing:

```go
// ✅ Good: Individual test functions
func TestCreatePublication(t *testing.T) { /* ... */ }
func TestCreateSubscription(t *testing.T) { /* ... */ }
func TestCreateReplication(t *testing.T) { /* ... */ }

// ❌ Avoid: Loop-based tests (harder to debug failures)
func TestCreateCommands(t *testing.T) {
    commands := []string{"publication", "subscription", "replication"}
    for _, cmd := range commands { /* ... */ }
}
```

### Descriptive Test Names

```go
// ✅ Good: Clear, descriptive names
func TestUpdateExtensionsActualUpgrade(t *testing.T)
func TestListExtensionsWithInvalidConnection(t *testing.T)
func TestPingCommandWithNonexistentAlias(t *testing.T)

// ❌ Avoid: Generic names
func TestUpdate(t *testing.T)
func TestList(t *testing.T)
```

### Grouped Test Scenarios

```go
func TestMyCommandValidation(t *testing.T) {
    ctx := context.Background()

    t.Run("missing_on_flag", func(t *testing.T) { /* ... */ })
    t.Run("invalid_alias", func(t *testing.T) { /* ... */ })
    t.Run("missing_config_file", func(t *testing.T) { /* ... */ })
}
```

## Best Practices

### 1. Resource Management

```go
// Always cleanup containers
pgContainer := SetupPostgreSQLContainer(ctx, t)
defer pgContainer.Cleanup(ctx, t)

CreateTempPgctlConfig(t, pgContainer)

// Wait for container readiness
pgContainer.WaitForReadiness(ctx, t, 30*time.Second)
```

### 2. Error Handling and Assertions

```go
// Use specific assertions
result.AssertSuccess(t)  // More specific than checking ExitCode == 0
result.AssertFailure(t)  // More specific than checking ExitCode != 0

// Chain assertions for fluent testing
result.AssertSuccess(t).
    AssertStdoutContains(t, "expected").
    AssertStderrEmpty(t)

// Use require for test setup, assert for verifications
require.NoError(t, err, "Setup should not fail")
assert.Equal(t, expected, actual, "Result should match")
```

### 3. Test Isolation

```go
// Each test gets its own container (slower but more reliable)
func TestMyFeature(t *testing.T) {
    pgContainer := SetupPostgreSQLContainer(ctx, t)
    defer pgContainer.Cleanup(ctx, t)
    // Test code here
}

// Use t.Helper() in utility functions
func myTestHelper(t *testing.T, container *PostgreSQLContainer) {
    t.Helper()  // Marks this as helper for better error reporting
    // Helper code here
}
```

## Troubleshooting

### Common Issues

#### Docker Not Running
```bash
# Check Docker status
docker ps
sudo systemctl start docker  # Linux
open -a Docker               # macOS
```

#### Port Conflicts
TestContainers automatically finds available ports, but check:
```bash
# See what's using PostgreSQL default port
lsof -i :5432
netstat -an | grep 5432
```

#### Binary Build Issues
```bash
# Manually build and test
make build
./bin/pgctl --help

# Check binary permissions
ls -la bin/pgctl
chmod +x bin/pgctl
```

#### Coverage Collection Issues
```bash
# Clean coverage data
make test-integration-clean

# Check Go version (requires 1.20+)
go version

# Verify coverage files are generated
ls -la coverage/
```

### Debugging Test Failures

```go
// Add debug logging
t.Logf("Container config: %+v", pgContainer.Config)
t.Logf("Command result: %+v", result)

// Print command output on failure
if result.ExitCode != 0 {
    t.Logf("Command failed: %s", result.Stderr)
    t.Logf("Stdout: %s", result.Stdout)
}
```

### Performance Issues

```bash
# Run specific tests only
go test -v ./test/integration/ping_test.go ./test/integration/testutils.go ./test/integration/main_test.go

# Increase timeouts for slow systems
export INTEGRATION_TEST_TIMEOUT=2m

# Use faster PostgreSQL startup
export POSTGRES_VERSION=16-alpine  # Smaller image
```

## Contributing Guidelines

When contributing new integration tests:

1. **Follow existing patterns** - Use the same structure as existing test files
2. **Individual test functions** - Don't use loops for multiple similar tests
3. **Comprehensive coverage** - Test help, validation, success, and error cases
4. **Real database operations** - Use actual containers when testing implemented features
5. **Proper cleanup** - Always defer cleanup for resources
6. **Clear naming** - Use descriptive test and file names
7. **Documentation** - Add comments for complex test scenarios
8. **Performance consideration** - Be mindful of test execution time

## Makefile Integration

Available targets:

```bash
make test-integration        # Run all integration tests with coverage
make test-integration-clean  # Clean coverage data
make build-with-coverage     # Build binary with coverage instrumentation
```

The integration tests are also included in:
```bash
make test-all               # Run both unit and integration tests
```

## CI/CD Integration

Integration tests run automatically in GitHub Actions with:
- Docker-in-Docker support for TestContainers
- Coverage collection and reporting
- Artifact storage for coverage reports
- Parallel execution for faster feedback

See `.github/workflows/checks.yml` for the complete CI configuration.
