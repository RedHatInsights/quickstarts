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

### SQLite for Tests

Tests use ephemeral SQLite databases instead of PostgreSQL:

```go
cfg.Test = true
dbName = fmt.Sprintf("%d-services.db", time.Now().UnixNano())
cfg.DbName = dbName
database.Init()
```

Each test run creates a unique `.db` file (timestamped) and removes it in `tearDown()`. This ensures test isolation without external dependencies.

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

These features are skipped in tests (SQLite) and should have fallback behavior:
- **Advisory locks** (`pg_advisory_xact_lock`) — skipped on non-PostgreSQL
- **Fuzzy search** (`fuzzystrmatch` extension) — falls back to `ILIKE`
- **SSL/TLS** — not applicable in tests

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

Tests run as part of the Docker build (`make test` in Dockerfile) and in the Konflux CI pipeline (`quickstarts-on-pull-request`). The CI environment uses the same SQLite-based test setup.
