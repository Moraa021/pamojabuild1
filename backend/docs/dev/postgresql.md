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

Every new schema change must add matching `<version>_<name>.up.sql` and `<version>_<name>.down.sql` files under `db/migrations`.

## Test PostgreSQL repositories

Tests create and remove isolated schemas inside a dedicated test database:

```bash
export TEST_DATABASE_URL='postgres://user:password@localhost:5432/pamoja_test?sslmode=disable'
make test-postgres
```

Normal `go test ./...` skips PostgreSQL integration tests when `TEST_DATABASE_URL` is absent. CI should always provide it.

The LND module brings an SQLite package transitively for LND internals. PamojaBuild does not import it, select it, or use SQLite as an application/test database.
