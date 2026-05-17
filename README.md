# dbmov — database export/import

`dbmov` discovers every schema your account can see, then shells out to `mysqldump` and the `mysql` client.

## Install
```bash
go install github.com/trevorak/dbmov@latest
```

## Build

```bash
go build -o dbmov .
```

## Usage

Connection flags are shared by both subcommands. Passwords can be passed with `--password`, or via `DBMOV_PASSWORD` / `MYSQL_PASSWORD`.

**Export** every user-visible database (excluding `information_schema`, `performance_schema`, `sys`, and `mysql` unless you opt in):

```bash
export DBMOV_PASSWORD='secret'
./dbmov export --to ./backup --user username --host db.example.com
```

**Import** from that directory (reads `manifest.json` if present, otherwise all `*.sql` and `*.sql.gz` files):

```bash
export DBMOV_PASSWORD='secret'
./dbmov import --from ./backup --user username --host localhost
```

### Useful flags

- `--include-mysql` — include the `mysql` system schema on export.
- `--include` / `--exclude` — repeated shell globs (same rules as `filepath.Match`). If `--include` is set, a database must match at least one include pattern.
- `--ssl-mode` — `disabled`, `preferred`, `skip-verify`, `required`, `verify-ca`, `verify-identity`.
- `--dump-column-statistics-off` — passes `mysqldump --column-statistics=0`. Use only when **MySQL 8+ mysqldump** warns or fails dumping an **older** server; **MariaDB and older mysqldump do not support this flag** (leave it unset, the default).
- `--gzip` (export) — write `*.sql.gz` via gzip; **import** decompresses `.gz` dumps automatically.
- `--continue-on-error` — continue after a failed dump or restore; exits non-zero if any step failed.
- `--mysqldump` / `--mysql` — override binary paths.


## Note

- **DEFINER** and privilege-dependent objects may fail on restore if the target server users do not exist or lack rights.
- **GTIDs**: `--set-gtid-purged` defaults to `OFF`; adjust for replication-aware restores.
- Credentials for subprocesses are passed through a temporary `my.cnf` fragment (mode `0600`).

