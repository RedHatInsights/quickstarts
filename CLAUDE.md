@AGENTS.md

## Build & Test Commands

```bash
# Install dependencies
go mod download

# Generate API code from OpenAPI spec (required before build/run)
make generate

# Start development server (generates API + runs server)
make dev

# Run all tests
make test

# Run specific package tests
go test ./pkg/routes -v
go test ./pkg/database -v

# Run specific test
go test ./pkg/routes -run TestGetQuickstarts -v

# Database migration + seeding
make migrate

# Convert OpenAPI YAML to JSON
make openapi-json

# Validate API responses against OpenAPI spec
make validate-api

# Validate quickstart/help-topic YAML content
make validate

# Build binary
go build -o quickstarts

# Local infrastructure (PostgreSQL)
make infra       # start
make stop-infra  # stop
```

## Pre-commit Checks

Before committing, always:
1. Run `make generate` if `spec/openapi.yaml` was changed
2. Run `make test` to verify all tests pass
3. Run `make validate` to check quickstart/help-topic content validity
