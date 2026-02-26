# pgctl

`pgctl` is a CLI tool that helps Engineers perform PostgreSQL maintenance operations safely and efficiently.

- Execute common maintenance operations on PostgreSQL databases
- Support for multiple database configurations
- Safe execution with dry-run capabilities
- Multiple configuration file formats (YAML, JSON, TOML)

## Install

### Prerequisites

Your environment must have installed:
- [Go 1.23 or higher](https://go.dev/doc/install)
- [Make](https://www.gnu.org/software/make/)
- pgdump (See [INSTALLATION.md](./INSTALLATION.md))

### Build from source

```sh
make build
```
This builds the project, creating a `pgctl` binary within the `/bin` directory.

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

To initialize this configuration in interactive mode, use:

```sh
./pgctl config init
```

> Note: alias are what you will use to refer to a connection string in your commands. You can put whatever you want and it isn't necessarily an server name or a database name.

## Usage

Most commands read as natural english language orders.

*Common flags:*

| Flag      | Meaning                                                                 |
| :-------- | ----------------------------------------------------------------------- |
| `--help`  | get details about the command                                           |
| `--on`    | alias selected                                                          |
| `--from`  | to select alias from, sometimes used in case of 2 alias needed          |
| `--to`    | to select alias toward which command should run, used in case of `copy` |
| `--apply` | remove dry-run mode (safety feature: dry run is the default)            |

*Commands*

```sh
# Configuration
./pgctl ping
./pgctl config init
./pgctl config show

# Basics
./pgctl list tables
./pgctl list databases

# Extensions management
./pgctl list extensions
./pgctl update extensions--all-databases

# Sequences management
./pgctl list sequences
./pgctl copy sequences --from alias --to alias2

# Pub-Sub
./pgctl list publications
./pgctl list subscriptions
./pgctl create publication --all-tables
./pgctl create publication --tables table1,table2
./pgctl create subscription --on alias2 --from alias --publication pubname
./pgctl drop publication --name pubname
./pgctl drop subscription --name subname

# Schema
./pgctl copy schema --from alias --to alias2 --all-tables #selecting specific tables not supported yet

# Relocation
./pgctl init relocation --from alias --to alias2 --no-ddl-confirmed
./pgctl run relocation --from alias --to alias2 --no-ddl-confirmed --no-writes-confirmed

# Checks
./pgctl check database-is-empty
./pgctl check have-similar-sequences
./pgctl check subscription-lag
./pgctl check tables-have-proper-replica-identity #used for logical replication
./pgctl check user-has-replication-grants
./pgctl check wal-level-is-logical
```
### Relocation

Relocation commands runs checks, creations and copy needed to move a database from one alias to another.
* The target database must exist.
* This does not copy users.
* It supports major upgrades (ex: from PG14 to PG18)

It reduces the task list to move all tables from db1 to db2 to:
1. configure pgctl with db1 and db2 alias (must be owner of the databases)
2. `pgctl init relocation`
3. wait and monitor
4. cut connection between your application(s)
5. `pgctl run relocation`
6. update connections strings and reconnect your application(s)

> ![!WARNING]
> This does not yet support other schemas than public.


## Safety Features

- Any changes are set to be dry runs by default, which can be then run for real with `--apply` flag
- Checks are automated for some commands that have requirements

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md)

## License

MIT
