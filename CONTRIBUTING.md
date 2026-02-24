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

### Local Testing

To run all unit tests:
```bash
make test
```

### Local Postgres Server

Spawn a local Postgres server:
```bash
make local-postgres
```

Run your own commands directly with:
```bash
make local-connect #password is "hackme"
```

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
