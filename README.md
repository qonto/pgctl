# pgctl

`pgctl` is a CLI tool that helps Engineers perform PostgreSQL maintenance operations safely and efficiently.

## Features

- Execute common maintenance operations on PostgreSQL databases
- Support for multiple database configurations
- Safe execution with dry-run capabilities
- Multiple configuration file formats (YAML, JSON, TOML)

## Installation

### Prerequisites

- Go 1.23 or higher
- Make
- pgdump (See [INSTALLATION.md](./INSTALLATION.md))

### Building from source

To build the project:

```sh
make build
```

This will create a `pgctl` binary within the `/bin` directory.

## Configuration

`pgctl` requires a configuration file containing connection details for your target databases.

Supported configuration formats:
- YAML
- JSON
- TOML

Example configuration (YAML):

```yaml
cool-alias:
  host: db1.host.com
  port: 5432
  database: db1
  user: user_db1
  password: hackme1

second-cool-alias:
  host: db2.host.com
  port: 5432
  database: db2
  user: user_db2
  password: hackme2
```

To initialize this configuration file, use:

```sh
./pgctl config init
```

> Note: alias are what you will use to refer to a connection string in your commands. You can put whatever you want and it isn't necessarily an server name or a database name.

## Usage

View available commands:
```sh
./pgctl
```

### Examples

```sh
pgctl ping
pgctl update extensions --on cool-alias --all-databases
```

## Safety Features

- Any changes are set to be dry runs by default, which can be then run for real with `--apply` flag

## Testing

This project includes comprehensive testing to ensure reliability and correctness.

### Quick Testing

```sh
# Run unit tests only
make test

# Run integration tests only
make test-integration
```

### Unit Tests

Unit tests cover individual functions and components:

```sh
# Run unit tests with coverage
make test

# Run specific package
go test ./pkg/cli/...

# Run with verbose output
go test -v ./pkg/...
```

### Integration Tests

Integration tests provide end-to-end testing with real PostgreSQL containers and actual pgctl binary execution.

#### Prerequisites

- **Docker**: Required for PostgreSQL containers via TestContainers
- **Go 1.23+**: For coverage instrumentation from binary execution

#### Running Integration Tests

```sh
# Run all integration tests with enhanced coverage collection
make test-integration

# Run specific command tests
go test -v ./test/integration/ping_test.go ./test/integration/testutils.go ./test/integration/main_test.go

# Run specific test function
go test -v ./test/integration/... -run TestPingCommand
```

#### Coverage Reports

Integration tests generate comprehensive coverage reports from actual binary execution:

```sh
# View coverage percentage
go tool cover -func=coverage/coverage-integrations-tests.out | tail -1

# Open HTML coverage report
open coverage/coverage-integrations-tests.html

# Clean coverage data
make test-integration-clean
```

#### Test Coverage

The integration test suite covers:

- ✅ **Implemented commands**: `ping`, `list extensions`, `update extensions` (with real database operations)
- 🔧 **CLI structure testing**: All commands include help, validation, and error handling tests
- 🔧 **Infrastructure**: PostgreSQL containers, configuration management, binary execution
- 📊 **Coverage**: We aim for >80% coverage

#### Test Organization

- **Individual test files per command**: `ping_test.go`, `config_test.go`, `update_test.go`, etc.
- **Real PostgreSQL containers**: Using TestContainers for authentic database operations
- **Binary instrumentation**: Coverage collection from actual pgctl binary execution
- **Comprehensive scenarios**: Help, validation, success, failure, and edge cases

For detailed integration test documentation, patterns, and contribution guidelines, see:

**📖 [Integration Test Documentation](./test/integration/README.md)**

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md)
