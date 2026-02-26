# Contributing

Architecture of this project:

* `internal/config`: abstracting viper
* `internal/postgres`: logic for interacting with PG
* `pkg/cli`: cobra command definition and binding
* `pkg/pgctl`: "service" level the actual UI + orchestration logic

## How to setup a local environment?

**Requirements:**
* Go 1.23 or higher
* `golangci-lint` 1.59 or higher
  * `brew install golangci-lint` on MacOS
* `pg_dump` (see [INSTALLATION.md](./INSTALLATION.md))
  ```bash
  brew install postgresql@14
  brew install postgresql@15
  brew install postgresql@16
  brew install postgresql@17
  # and when you want to switch
  brew unlink postgresql@14
  brew link postgresql@16 --force
  ```
* Make
* **Docker**: Required for PostgreSQL containers via TestContainers


## Testing

This project includes comprehensive testing to ensure reliability and correctness.

### Local Postgres Server

Spawn a local Postgres server:
```bash
make local-postgres
```

Run your own commands directly with:
```bash
make local-connect #password is "hackme"
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

```sh
# Run integration tests only
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

- ✅ **Implemented commands**: `ping`, `list extensions`, `update extensions` (with real database operations), etc
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


## Release Process

New releases are triggered by changes to the CHANGELOG.md file. Here's the process:

1. Update CHANGELOG.md:
   * Add a new version section at the top
   * Document all significant changes

2. Create a pull request with your changes to the main branch, a code owner will review it.

3. Automated Release Process:
   * CI will detect the new version in CHANGELOG.md
   * A new git tag will be created automatically
   * Build pipeline will:
     * Build the project
     * Run all tests
     * Push Docker images to registries
     * Tag images with the new version

**Note:** If a git tag already exists for the version specified in CHANGELOG.md, the build process will be skipped.
