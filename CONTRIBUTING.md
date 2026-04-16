# Contributing to Quickstarts

## Prerequisites

- Go 1.25.7+
- Docker (for local PostgreSQL)
- Make

## Setup

1. Clone the repository
2. Copy `.env.example` to `.env` and configure database credentials
3. Install dependencies: `go mod download`
4. Start local PostgreSQL: `make infra`
5. Run migrations and seed content: `make migrate`
6. Start the development server: `make dev`

## Development Workflow

### API Changes (Spec-First)

1. Edit `spec/openapi.yaml` to define the API contract
2. Run `make generate` to regenerate `pkg/generated/api.go`
3. Run `make openapi-json` to update the JSON spec
4. Implement the new interface methods in `pkg/routes/server_adapter.go`
5. Add business logic in `pkg/services/`
6. Add tests
7. Run `make test` to verify

### Content Changes

Quickstart and help topic YAML files live in `docs/quickstarts/` and `docs/help-topics/`. Use the CLI tool to scaffold new content:

```bash
make create-resource
```

After editing content, validate:

```bash
make validate
```

### Non-API Changes

1. Make code changes
2. Run `make test`
3. Commit with conventional commit message

## Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): short description

TICKET-KEY (if applicable)
Longer explanation if needed.
```

**Types**: `fix`, `feat`, `chore`, `refactor`, `docs`, `test`

**Scopes**: `fuzzy`, `seeding`, `api`, `progress`, `favorites`, `helptopics`, or omit for broad changes

**Examples**:
```
fix(seeding): wrap database seeding in transaction
feat(fuzzy): add Levenshtein distance matching
chore: fix dependencies
```

## Pull Request Guidelines

- Keep PRs focused — one logical change per PR
- Include the Jira ticket key in the PR body
- Ensure all CI checks pass (tests, validation, security scan)
- Update `spec/openapi.yaml` if API behavior changes
- Add tests for new functionality
- Maintain backward compatibility for API parameter formats

## Testing

```bash
# Run all tests
make test

# Run specific package
go test ./pkg/routes -v

# Run specific test
go test ./pkg/database -run TestSeedTags -v

# View coverage
make coverage
```

Tests use SQLite in-memory databases. See [Testing Guidelines](docs/testing-guidelines.md) for patterns and conventions.

## Project Structure

See [AGENTS.md](AGENTS.md) for the complete directory layout and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for system design.
