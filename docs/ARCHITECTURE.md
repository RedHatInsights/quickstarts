# Architecture

## System Overview

The quickstarts service is a Go backend API that serves learning resources (quickstarts, help topics) to the Hybrid Cloud Console (HCC) frontend. It provides CRUD operations for quickstarts, help topics, favorites, and progress tracking.

```
┌─────────────────┐     ┌──────────────────┐     ┌────────────┐
│  HCC Frontend   │────▶│  Quickstarts API  │────▶│ PostgreSQL │
│ (learning-      │     │  (chi + GORM)     │     │   (v13)    │
│  resources)     │◀────│                   │◀────│            │
└─────────────────┘     └──────────────────┘     └────────────┘
                              │
                              ▼
                        ┌──────────┐
                        │  /docs   │
                        │  YAML    │
                        │  content │
                        └──────────┘
```

## Request Flow

1. HTTP request arrives at chi router (`main.go`)
2. Middleware chain: request ID → real IP → recovery → logging → Prometheus metrics
3. `generated.HandlerFromMuxWithBaseURL` routes to the correct handler based on OpenAPI spec
4. `ServerAdapter` (implements `generated.ServerInterface`) parses parameters and delegates to services
5. Service layer executes GORM queries against the database
6. Response is formatted as JSON with `{"data": ...}` envelope

## Data Flow: Content Seeding

Content flows from YAML files to the database during deployment:

```
docs/quickstarts/*/metadata.yaml  ──┐
docs/help-topics/*/metadata.yaml  ──┤
                                    ▼
                            findTags() scans docs/
                                    │
                                    ▼
                          SeedTags() (transactional)
                            ├── clearOldContent()
                            ├── seedDefaultTags()
                            ├── seedQuickstart() per template
                            ├── seedHelpTopic() per template
                            └── seedFavorites() (restore)
```

The seeding process runs inside a PostgreSQL transaction with an advisory lock (`pg_advisory_xact_lock`) to prevent race conditions when multiple pods start simultaneously.

**Favorites preservation**: Before clearing content, `clearOldContent()` reads all `FavoriteQuickstart` records into memory. After seeding new content, `seedFavorites()` re-creates the favorites by matching each saved favorite's `QuickstartName` against the newly seeded quickstarts. Favorites whose quickstart no longer exists (removed from YAML) are silently dropped. See `pkg/database/db_seed.go` for the implementation.

## Deployment Architecture

### Kubernetes / ClowdApp

```yaml
ClowdApp (deploy/clowdapp.yml)
├── initContainer: quickstarts-migrate
│   └── Runs AutoMigrate + SeedTags()
└── container: quickstarts (HTTP server)
    ├── Port 8000 (API)
    ├── Liveness: GET /test
    ├── Readiness: GET /test
    └── Metrics: /metrics (separate port)
```

### Two Binaries

The Dockerfile produces two binaries from the same codebase:

| Binary | Source | Purpose |
|--------|--------|---------|
| `quickstarts` | `main.go` | HTTP API server |
| `quickstarts-migrate` | `cmd/migrate/migrate.go` | Schema migration + content seeding |

### Build Pipeline

The Dockerfile runs a multi-stage build:
1. Generate API code (`make generate`)
2. Validate API spec (`make validate-api`)
3. Download dependencies (`go get`)
4. Convert OpenAPI to JSON (`make openapi-json`)
5. Validate content (`make validate`)
6. Run tests (`make test`)
7. Build both binaries with `CGO_ENABLED=0`

## API Design

### Base Path

All API endpoints are under `/api/quickstarts/v1/`.

### Key Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/quickstarts/` | List/filter quickstarts |
| GET | `/quickstarts/{id}` | Get single quickstart |
| GET | `/helptopics/` | List/filter help topics |
| GET | `/helptopics/{name}` | Get help topic by name |
| POST | `/progress` | Create/update user progress |
| DELETE | `/progress/{id}` | Delete user progress |
| POST | `/favorites` | Toggle favorite status |
| GET | `/favorites` | List user favorites |

### Filtering

Quickstarts support tag-based filtering with multiple tag types: `bundle`, `application`, `product-families`, `use-case`, `content`, `kind`, `topic`. Tags are stored in a many-to-many relationship via the `Tag` model.

### Fuzzy Search

The API supports fuzzy search using PostgreSQL's `fuzzystrmatch` extension (Levenshtein distance). Falls back to `ILIKE` on SQLite (tests). Configurable via `FUZZY_SEARCH_DISTANCE_THRESHOLD` env var (default: 3).

## Database Schema

### Core Models

```
Quickstart (1) ──── (*) Tag (many-to-many via quickstart_tags)
HelpTopic  (1) ──── (*) Tag (many-to-many via help_topic_tags)
Quickstart (1) ──── (*) FavoriteQuickstart
Quickstart (1) ──── (*) QuickstartProgress
```

All models use `gorm.Model` which provides `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt` (soft delete). However, the seeding process uses hard delete (`Unscoped().Delete()`) to clear and re-insert content.

## Configuration

The service uses Clowder (`app-common-go`) for cloud configuration. When `CLOWDER_ENABLED=true`, database credentials, ports, and SSL settings come from the Clowder-injected config. For local development, standard PostgreSQL env vars are used (`PGSQL_USER`, `PGSQL_PASSWORD`, `PGSQL_HOSTNAME`, `PGSQL_PORT`, `PGSQL_DATABASE`).
