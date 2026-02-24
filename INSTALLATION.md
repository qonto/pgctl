# Installation Guide for pgctl Dependencies

## pg_dump Installation

The `pgctl copy schema` command requires `pg_dump` to be installed and available in your system PATH. This tool is part of the PostgreSQL client tools package.

### macOS

#### Option 1: Using Homebrew (Recommended)

```bash
# Install PostgreSQL (includes pg_dump)
brew install postgresql

# Or install just the client tools
brew install libpq

# Add PostgreSQL bin directory to PATH if not already added
echo 'export PATH="/opt/homebrew/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

#### Option 2: Using MacPorts

```bash
sudo port install postgresql16 +universal
```

#### Option 3: Using PostgreSQL.app

1. Download and install [PostgreSQL.app](https://postgresapp.com/)
2. Add the command line tools to your PATH:

```bash
echo 'export PATH="/Applications/Postgres.app/Contents/Versions/latest/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Linux

#### Ubuntu/Debian

```bash
# Update package list
sudo apt update

# Install PostgreSQL client tools
sudo apt install postgresql-client

# For a specific version (e.g., PostgreSQL 16)
sudo apt install postgresql-client-16
```

#### CentOS/RHEL/Fedora

```bash
# CentOS/RHEL 8+
sudo dnf install postgresql

# CentOS/RHEL 7
sudo yum install postgresql

# Fedora
sudo dnf install postgresql
```

#### Arch Linux

```bash
sudo pacman -S postgresql
```

#### Alpine Linux

```bash
apk add postgresql-client
```

### Windows

#### Option 1: Using the Official PostgreSQL Installer

1. Download the PostgreSQL installer from [postgresql.org](https://www.postgresql.org/download/windows/)
2. Run the installer and make sure to select "Command Line Tools" during installation
3. Add the PostgreSQL bin directory to your PATH:
   - Typically located at: `C:\Program Files\PostgreSQL\16\bin`

#### Option 2: Using Package Managers

```powershell
# Using Chocolatey
choco install postgresql
# Using Scoop
scoop install postgresql
```

## Verification

After installation, verify that `pg_dump` is available:

```bash
pg_dump --version
```

You should see output similar to:
```
pg_dump (PostgreSQL) 16.1
```

## Version Compatibility

The `pgctl copy schema` command includes automatic version compatibility checking. It's recommended to use a `pg_dump` version that is the same or newer than your PostgreSQL server version for best compatibility.

### Supported Combinations

| PostgreSQL Server | pg_dump Version | Status |
|------------------|----------------|---------|
| 13.x | 13.x+ | ✅ Supported |
| 14.x | 14.x+ | ✅ Supported |
| 15.x | 15.x+ | ✅ Supported |
| 16.x | 16.x+ | ✅ Supported |

## Troubleshooting

### Command Not Found

If you get a "command not found" error for `pg_dump`:

1. **Check if PostgreSQL is installed**: Run `which pg_dump` or `where pg_dump` (Windows)
2. **Update PATH**: Make sure the PostgreSQL bin directory is in your PATH
3. **Reinstall**: Try reinstalling PostgreSQL client tools

### Permission Issues

If you encounter permission issues:

```bash
# On macOS/Linux, ensure you have proper permissions
sudo chown -R $(whoami) /usr/local/var/postgresql/
```

### Version Mismatch

If you get version compatibility warnings:

1. **Update pg_dump**: Install a newer version of PostgreSQL client tools
2. **Check server version**: Connect to your PostgreSQL server and run `SELECT version();`
3. **Use compatible versions**: Ensure pg_dump version ≥ PostgreSQL server version

## Docker Users

If you're using PostgreSQL in Docker, you can also use pg_dump from a Docker container:

```bash
# Run pg_dump from PostgreSQL Docker image
docker run --rm -it postgres:16 pg_dump --version

# Use it in place of local pg_dump (create an alias)
alias pg_dump='docker run --rm -i postgres:16 pg_dump'
```

## Getting Help

If you continue to experience issues with pg_dump installation:

1. Check the [PostgreSQL documentation](https://www.postgresql.org/docs/current/app-pgdump.html)
2. Verify your operating system's package manager documentation
3. Ensure you have the necessary permissions to install software
4. Consider using Docker as an alternative installation method

For pgctl-specific issues, please refer to the main project documentation or open an issue in the project repository.
