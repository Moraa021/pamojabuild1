# PostgreSQL Development and Migrations

PamojaBuild supports PostgreSQL only. The API requires an existing, migrated database and never changes the schema during startup.

## Configure

```bash
export DATABASE_URL='postgres://user:password@localhost:5432/pamoja?sslmode=disable'
```

Use TLS and managed secrets outside local development.

## Migrate

Run from `backend/`:

```bash
go run ./cmd/migrate up
go run ./cmd/migrate version
go run ./cmd/migrate down 1
```

`down` requires an explicit step count. `force` only repairs migration metadata after an operator inspects a dirty migration:

```bash
go run ./cmd/migrate force <version>
```

Migration versions use a 14-digit UTC timestamp (`YYYYMMDDHHMMSS`) so independently created team migrations are unlikely to collide. Create a matching up/down pair from `backend/` with:

```bash
make migration-create NAME=add_example_table
```

This calls the `golang-migrate` CLI and produces:

```text
db/migrations/<UTC timestamp>_add_example_table.up.sql
db/migrations/<UTC timestamp>_add_example_table.down.sql
```

Do not use `-seq` or manually invent a sequence number. If two migrations still receive the same timestamp, regenerate one before committing.

## Test PostgreSQL repositories

Tests create and remove isolated schemas inside a dedicated test database:

```bash
export TEST_DATABASE_URL='postgres://user:password@localhost:5432/pamoja_test?sslmode=disable'
make test-postgres
```

Normal `go test ./...` skips PostgreSQL integration tests when `TEST_DATABASE_URL` is absent. CI should always provide it.

The LND module brings an SQLite package transitively for LND internals. PamojaBuild does not import it, select it, or use SQLite as an application/test database.
