# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the Red Hat Insights Quickstarts backend service written in Go. It serves quickstarts and help topics for the Red Hat Hybrid Cloud Console (console.redhat.com). The service provides REST APIs for managing quickstarts, help topics, progress tracking, and favorites.

## Common Commands

### Development Setup
- Copy environment variables: `cp .env.example .env`
- Start database infrastructure: `make infra`
- Run database migration and seeding: `make migrate`
- Start the server: `go run main.go`
- Stop infrastructure: `make stop-infra`

### Testing and Validation
- Run tests: `make test` or `go test ./...`
- Generate test coverage report: `make coverage`
- Validate quickstarts content: `make validate`
- Validate help topics: `make validate-topics`

### Build and Deployment
- Generate OpenAPI spec: `make generate-spec`
- Build Docker image: `docker build . -t quickstarts:latest`
- Security audit: `make audit`

### Content Creation
- Bootstrap new quickstart/help topic: `make create-resource` or `./make_item.sh`

## Architecture

### Directory Structure
- `main.go` - Application entry point with HTTP server setup
- `pkg/` - Core application packages
  - `database/` - Database configuration, migration, and seeding
  - `models/` - GORM data models (Quickstart, HelpTopic, Tag, etc.)
  - `routes/` - HTTP handlers and API endpoints
  - `logger/` - Logging configuration
- `cmd/` - Command-line utilities
  - `migrate/` - Database migration tool
  - `validate/` - Content validation tools
  - `spec/` - OpenAPI spec generation
- `docs/` - Content files
  - `quickstarts/` - Quickstart YAML content organized by directories
  - `help-topics/` - Help topic YAML content
- `cli/` - Shell scripts for content creation
- `config/` - Configuration management

### Key Models
- **Quickstart**: Interactive tutorials and learning resource cards
- **HelpTopic**: Contextual help content 
- **Tag**: Categorization with kind/value pairs (bundle, product-families, content, use-case)
- **QuickstartProgress**: User progress tracking
- **FavoriteQuickstart**: User favorites

### Content Structure
Each quickstart/help topic requires:
1. `metadata.yml` - Contains kind, name, and tags for categorization
2. `<name>.yml` - Content file matching the name in metadata

### Tags System
- `bundle` - Controls which Learning Resources page shows content (insights, openshift, ansible, etc.)
- `product-families` - Product categorization (rhel, iam, settings, etc.)
- `content` - Content type (quickstart, documentation, learningPath, otherResource)
- `use-case` - Optional filtering (automation, security, infrastructure, etc.)

### API Endpoints
- `/api/quickstarts/v1/quickstarts/` - Quickstarts with filtering by bundle, application
- `/api/quickstarts/v1/progress` - Progress tracking CRUD
- `/api/quickstarts/v1/topics/` - Help topics with similar filtering
- `/api/quickstarts/v1/favorites` - User favorites management

### Database
Uses GORM with PostgreSQL in production, SQLite for local development. Database seeding loads content from YAML files in `docs/` directories.

## Development Notes

### Testing
Tests use testify for assertions. Database tests use an in-memory SQLite instance.

### Content Validation
The `validate` commands check YAML structure and required fields. Use the CLI tools to bootstrap new content following the established patterns.

### Environment Variables
Required variables are documented in `.env.example`. Key variables include database connection, server configuration, and feature flags.