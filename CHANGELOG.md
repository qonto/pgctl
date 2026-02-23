# v2.3.3

- Revert bug introduced in `v2.3.2`: pg_dump should be the same as target

# v2.3.2

- Avoid failure when pg_dump versions are different (useful for pg upgrades)

# v2.3.1

- Avoid (false) errors when running `pgctl init relocation` in dry run mode

# v2.3.0

- Add users check and improve messages

# v2.2.0

- Remove `--no-comments` flag from `pg_dump`

# v2.1.3

- Improve error message for subscription failure

# v2.1.2

- Fix pg_dump compatibility check to be more understandable

# v2.1.1

- Fix `pgctl create password` to not ask for a config file

# v2.1.0

- Add `--with-schema-prefix` to `pgctl list tables` (useful in case of different schemas usage)
- Fix relocation command to list tables without schema explicitely

# v2.0.0

- Add check for rds_replication grant if user has no replication grant **#RDS_SPECIFICS**
- `pgctl copy schema` : Remove support for specific tables as it breaks in case of triggers
- `pgctl create publication` and `subscription` : Fix name to use underscores if the database name has hyphens
- `pgctl create subscription`: Add check that user has pg_create_subscription role
- `pgctl copy sequences`: Add **if not exists** to create query, meaning that it'll update existing sequences

# v1.6.0

- Update logical replication pre-checks by checking for tables with misconfigured replica identity instead of tables without primary keys.


# v1.5.0

- Add `pgctl run relocation`which combines multiple checks and create commands to run a database relocation (lag wait, sequence copy, delete pub, delete sub)

# v1.4.0

- Add `pgctl init relocation` which combines multiple checks and create commands to initialise a database relocation (schema copy, pub, sub)

# v1.3.1

- Edit `pgctl copy schema` to send a warning rather than an error if pg version is different between source and target

# v1.3.0

- Add `pgctl list databases` command

# v1.2.0

- Add `pgctl check subscription-lag` command

# v1.1.0

- Add installation script

# v1.0.2

- Fix version not injected at release time

# v1.0.1

- Fix release process and cleanup tags

# v1.0.0

- Implement all database migrations commands

# v0.3.2

- Fix `update extensions --all-databases` command

# v0.3.1

- Fix `update extensions` command

# v0.3.0

- Add `list extensions` command

# v0.2.0

- Add `ping` command
- Add `update extensions` command
- Improve documentation and local environment

# v0.1.0

- Initialize pgctl project structure
- Expose a Cobra root command publicly
- Build our own CLI using the public root command

# v0.0.1

- Project initialization
