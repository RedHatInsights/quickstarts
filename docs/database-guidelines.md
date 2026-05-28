# Database Guidelines

## ORM: GORM

The project uses GORM (`gorm.io/gorm`) with PostgreSQL in production and SQLite for tests.

## Models

All models are in `pkg/models/` and embed `gorm.Model` (provides `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`):

| Model | Table | Purpose |
|-------|-------|---------|
| `Quickstart` | `quickstarts` | Learning resource content (JSON blob) |
| `HelpTopic` | `help_topics` | Help panel content |
| `Tag` | `tags` | Tag categories (many-to-many with quickstarts and help topics) |
| `FavoriteQuickstart` | `favorite_quickstarts` | User favorites (by account ID + quickstart name) |
| `QuickstartProgress` | `quickstart_progresses` | User progress tracking |

### Tag Associations

Tags use many-to-many associations:
```go
type Tag struct {
    gorm.Model
    Type        TagType
    Value       string
    Quickstarts []Quickstart `gorm:"many2many:quickstart_tags"`
    HelpTopics  []HelpTopic  `gorm:"many2many:help_topic_tags"`
}
```

## Database Initialization

`pkg/database/db.go` handles connection setup:
- PostgreSQL in production (via Clowder config)
- SQLite for tests (`cfg.Test = true`)
- Creates tables if they don't exist
- Enables `fuzzystrmatch` extension on PostgreSQL

The global `DB` variable holds the connection. In seeding functions, always use the transaction handle (`tx`) instead of `DB`.

## Seeding

Content seeding (`pkg/database/db_seed.go`) runs during migration:

### Flow

1. `SeedTags()` — entry point, wraps everything in a transaction
2. `clearOldContent(tx)` — hard-deletes all quickstarts, help topics, tags, favorites
3. `seedDefaultTags(tx)` — creates default tag entries per tag type
4. For each YAML template in `docs/`:
   - `seedQuickstart(tx, ...)` or `seedHelpTopic(tx, ...)`
   - Create/find tags and associate with content
5. `seedFavorites(tx, ...)` — restore previously saved favorites

### Concurrency Protection

Seeding uses a PostgreSQL advisory lock to serialize concurrent pod startups:

```go
const seedAdvisoryLockID = 42
tx.Exec("SELECT pg_advisory_xact_lock(?)", seedAdvisoryLockID)
```

The lock is transaction-scoped and auto-releases on commit/rollback. Skipped on non-PostgreSQL (SQLite tests).

### Content Format

Quickstart metadata files follow this structure:
```
docs/quickstarts/<name>/metadata.yaml
docs/quickstarts/<name>/<name>.yaml  (content)
```

The `findTags()` function scans `docs/` for `metadata.yaml` files and parses them into templates.

## Migrations

Schema migrations use GORM's `AutoMigrate`:

```go
DB.AutoMigrate(
    &models.Quickstart{},
    &models.QuickstartProgress{},
    &models.Tag{},
    &models.HelpTopic{},
    &models.FavoriteQuickstart{},
)
```

This runs in the `quickstarts-migrate` binary before each pod starts.

## Query Patterns

### Service Layer Queries

Services use GORM's query builder:

```go
// Basic query with preloading
db.Preload("Tags").Find(&quickstarts)

// Filtering with conditions
db.Where("name = ?", name).First(&quickstart)

// Pagination
db.Limit(limit).Offset(offset).Find(&quickstarts)

// Association queries for tags
db.Model(&tag).Association("Quickstarts").Find(&quickstarts)
```

### Soft Delete

Models have `DeletedAt` (soft delete), but seeding uses `Unscoped().Delete()` (hard delete) to fully remove old content before re-inserting.

## Error Handling in Transactions

All database operations inside transactions must check errors and return them to trigger rollback:

```go
if err := tx.Create(&record).Error; err != nil {
    return err  // triggers transaction rollback
}
```

Prefer returning errors over logging and continuing — partial commits inside transactions create inconsistent state.
