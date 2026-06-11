# Testing Guidelines

## Test Infrastructure

### TestMain Pattern

Every test package uses a `TestMain` function in `main_test.go` for setup/teardown:

```go
func TestMain(m *testing.M) {
    setUp()
    retCode := m.Run()
    tearDown()
    os.Exit(retCode)
}
```

### Database Backend for Tests

Tests support two database backends:

#### SQLite (default)

When no `TEST_DATABASE_URL` environment variable is set, tests use ephemeral SQLite databases:

```go
cfg.Test = true
dbName = fmt.Sprintf("%d-services.db", time.Now().UnixNano())
cfg.DbName = dbName
database.Init()
```

Each test run creates a unique `.db` file (timestamped) and removes it in `tearDown()`. This ensures test isolation without external dependencies.

#### PostgreSQL (recommended for full coverage)

Set `TEST_DATABASE_URL` to run tests against PostgreSQL, enabling full coverage of fuzzy search (fuzzystrmatch/Levenshtein), advisory locks, and other PostgreSQL-specific features:

```bash
# Start local PostgreSQL
make infra

# Run tests with PostgreSQL
make test-pg
```

Or set the variable directly:

```bash
TEST_DATABASE_URL="host=localhost user=quickstarts password=quickstarts dbname=quickstarts_test port=5432 sslmode=disable" go test -p 1 ./... -v
```

**Important**: Use `-p 1` (or `make test-pg`) when running against PostgreSQL. By default `go test ./...` runs packages in parallel, which causes race conditions on the shared test database (concurrent `TRUNCATE`/`INSERT` across packages). The `-p 1` flag serialises package execution.

When using PostgreSQL, `CleanTestTables()` truncates all tables at the start of each test run to ensure a clean state (SQLite achieves this by creating a fresh file).

### Schema Setup

Tests run `AutoMigrate` in `setUp()` to create tables:

```go
database.DB.AutoMigrate(
    &models.Quickstart{},
    &models.QuickstartProgress{},
    &models.HelpTopic{},
    &models.FavoriteQuickstart{},
    &models.Tag{},
)
```

The `database` package tests also call `SeedTags()` after migration to test the seeding flow end-to-end.

## Test Packages

| Package | What it tests | SQLite DB? | Seeds data? |
|---------|--------------|------------|-------------|
| `pkg/routes` | HTTP handlers, parameter parsing, API responses | Yes | No (inserts test data per test) |
| `pkg/database` | Seeding logic, content parsing, tag associations | Yes | Yes (`SeedTags()`) |

## Writing Tests

### HTTP Handler Tests

Use `net/http/httptest` to test handlers:

```go
func TestSomeEndpoint(t *testing.T) {
    // Insert test data
    database.DB.Create(&models.Quickstart{...})

    // Create request
    req := httptest.NewRequest("GET", "/api/quickstarts/v1/quickstarts/", nil)
    w := httptest.NewRecorder()

    // Execute
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Assertion Library

Use `github.com/stretchr/testify`:
- `assert.Equal(t, expected, actual)`
- `assert.NoError(t, err)`
- `assert.Contains(t, string, substring)`

### Test Data Cleanup

Since each test package gets its own ephemeral database, there's no need to clean up between individual tests within the same package. However, if a test modifies shared state (like seeded content), be aware of ordering effects since tests in Go run sequentially within a package by default.

## PostgreSQL-Only Features

Some features are PostgreSQL-only and must be guarded in code:

```go
if db.Dialector.Name() == "postgres" {
    // PostgreSQL-specific code (advisory locks, fuzzystrmatch, etc.)
}
```

These features have fallback behavior when running with SQLite:
- **Advisory locks** (`pg_advisory_xact_lock`) — skipped on non-PostgreSQL
- **Fuzzy search** (`fuzzystrmatch` extension) — falls back to `ILIKE`
- **SSL/TLS** — not applicable in tests

When running tests with `TEST_DATABASE_URL` (PostgreSQL), all of these features are fully tested including fuzzystrmatch-based fuzzy search.

## Running Tests

```bash
# All tests with coverage
make test

# Specific package with verbose output
go test ./pkg/routes -v

# Specific test function
go test ./pkg/database -run TestSeedTags -v

# Coverage report in browser
make coverage
```

## CI Testing

Tests run as part of the Docker build (`make test` in Dockerfile) and in the Konflux CI pipeline (`quickstarts-on-pull-request`). By default, CI uses SQLite. To enable PostgreSQL testing in Konflux, add a PostgreSQL sidecar container to the Tekton pipeline and set the `TEST_DATABASE_URL` environment variable during the build step.
