# Quickstarts — Agent Guide

## Project Overview

Backend API service for HCC (Hybrid Cloud Console) integrated quickstarts and help topics. Serves learning resources content (quickstarts, help topics, favorites, progress tracking) to the HCC frontend. Content is defined as YAML metadata files in `docs/` and seeded into PostgreSQL on deployment.

## Tech Stack

- **Language**: Go 1.25.7
- **HTTP Router**: chi v5 (`github.com/go-chi/chi/v5`)
- **ORM**: GORM (`gorm.io/gorm`) with PostgreSQL (prod) and SQLite (tests)
- **API Generation**: oapi-codegen — generates server interfaces and models from `spec/openapi.yaml`
- **Logging**: logrus (structured logging with `slog` in seeding code)
- **Metrics**: Prometheus (`/metrics` endpoint)
- **Config**: Clowder (`app-common-go`) for cloud deployment, env vars for local
- **Deployment**: ClowdApp on OpenShift, PostgreSQL 13, Konflux CI

## Directory Structure

```
.
├── main.go                  # HTTP server entry point
├── spec/
│   ├── openapi.yaml         # OpenAPI spec (source of truth)
│   └── openapi.json         # Generated JSON version
├── pkg/
│   ├── generated/api.go     # Auto-generated from OpenAPI (NEVER edit)
│   ├── routes/              # Server adapter + HTTP handlers
│   │   ├── server_adapter.go    # Implements generated.ServerInterface
│   │   ├── quickstarts_handlers.go
│   │   ├── favorites_handlers.go
│   │   ├── helptopics_handlers.go
│   │   ├── progress_handlers.go
│   │   ├── quickstarts_query.go # Query building helpers
│   │   └── metrics.go           # Prometheus middleware
│   ├── services/            # Business logic + data access
│   │   ├── quickstart_service.go
│   │   ├── helptopic_service.go
│   │   ├── favorite_service.go
│   │   └── progress_service.go
│   ├── database/            # DB init, seeding, migrations
│   │   ├── db.go            # Connection setup (PostgreSQL/SQLite)
│   │   ├── db_seed.go       # Content seeding from YAML metadata
│   │   └── db_test.go       # Seeding tests
│   ├── models/              # GORM models
│   │   ├── quickstart.go
│   │   ├── help_topic.go
│   │   ├── tag.go           # Many-to-many tag associations
│   │   ├── favorite_quickstart.go
│   │   └── quickstart_progress.go
│   ├── utils/               # Shared utilities
│   └── logger/              # Router logging middleware
├── cmd/
│   ├── migrate/migrate.go   # Migration + seeding binary (quickstarts-migrate)
│   ├── validate/            # Content validation CLI
│   ├── yaml-to-json/        # OpenAPI YAML→JSON converter
│   ├── check-openapi-json/  # API spec validation
│   └── favorite/            # Favorite testing utility
├── config/config.go         # Configuration (Clowder + env vars)
├── docs/                    # Content YAML files
│   ├── quickstarts/         # Quickstart definitions (metadata.yaml + content)
│   ├── help-topics/         # Help topic definitions
│   └── developers/          # Developer docs
├── deploy/clowdapp.yml      # ClowdApp deployment template
├── cli/                     # Shell scripts for creating new content
└── spec/openapi.yaml        # API contract (source of truth)
```

## Key Architecture Decisions

### Spec-First API Development

The OpenAPI spec (`spec/openapi.yaml`) is the single source of truth. Development flow:
1. Update `spec/openapi.yaml` with new endpoints/schemas
2. Run `make generate` to regenerate `pkg/generated/api.go`
3. Implement the generated `ServerInterface` methods in `pkg/routes/server_adapter.go`
4. Add business logic in `pkg/services/`

**Never edit `pkg/generated/api.go` directly** — it is overwritten on every `make generate`.

### Two-Binary Architecture

The Docker image contains two binaries:
- `quickstarts` — the HTTP API server (`main.go`)
- `quickstarts-migrate` — migration + seeding (`cmd/migrate/migrate.go`)

The migration binary runs as a Kubernetes initContainer before each pod starts. It runs `AutoMigrate` + `SeedTags()` to ensure the database schema is current and content is seeded.

### Content Seeding

Quickstart and help topic content lives as YAML files in `docs/quickstarts/` and `docs/help-topics/`. The seeding process (`pkg/database/db_seed.go`) scans these directories, parses metadata, and upserts content into the database. Seeding is wrapped in a transaction with a PostgreSQL advisory lock to prevent race conditions during concurrent pod startups.

## Cross-Cutting Conventions

### Commit Messages

Use conventional commits: `type(scope): description`
- `fix(scope):` for bug fixes
- `feat(scope):` for new features
- `chore(scope):` for maintenance
- `refactor(scope):` for refactoring

### Error Responses

All error responses use the generated types:
```go
w.WriteHeader(http.StatusBadRequest)
w.Header().Set("Content-Type", "application/json")
msg := "Error description"
resp := generated.BadRequest{Msg: &msg}
json.NewEncoder(w).Encode(resp)
```

### Parameter Handling

The API supports both standard and legacy array parameter formats for backward compatibility:
- Standard: `bundle=rhel&bundle=insights`
- Legacy: `bundle[]=rhel&bundle[]=insights`

Always support both formats when adding new array parameters.

### Response Format

Success responses wrap data in a `{"data": [...]}` envelope. Single-item responses may use `{"data": {...}}`.

## Common Pitfalls

1. **Forgetting `make generate`** after changing `spec/openapi.yaml` — the generated code will be stale and compilation may fail or behavior may diverge from the spec.
2. **Editing `pkg/generated/api.go`** — changes are lost on next generation. Implement behavior in `server_adapter.go`.
3. **SQLite vs PostgreSQL differences** — tests use SQLite. Features like `fuzzystrmatch`, advisory locks, and certain SQL syntax are PostgreSQL-only. Guard with dialect checks (`db.Dialector.Name() == "postgres"`).
4. **Seeding race conditions** — multiple pods seed concurrently. Always use the transaction handle (`tx`) inside seeding functions, not the global `DB`.
5. **Content YAML structure** — quickstart metadata files have a specific format. Use `make validate` to check content before committing.
6. **Legacy parameter support** — removing legacy `[]` parameter format breaks existing frontend clients. Always maintain backward compatibility.

## Documentation

See the [Documentation section in README.md](README.md#documentation) for the full index of project docs (architecture, testing, API development, database guidelines, content guides, etc.).
