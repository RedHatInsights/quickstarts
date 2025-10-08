help:
	@echo "Available commands:"
	@echo "------------------"
	@echo "dev		- generate API and start development server"
	@echo "test		- run tests"
	@echo "coverage	- open browser with detailed test coverage report"
	@echo "migrate		- run database migration"
	@echo	"validate-topics - run help topics validator"
	@echo  "infra           - start required infrastructure"
	@echo "stop-infra      - stop required infrastructure"
	@echo "audit 		- run grype audit on the docker image"
	@echo "create-resource	- a cli tool to bootstrap a new learning resource"
	@echo ""
	@echo "=== oapi-codegen Migration ==="
	@echo "setup-tools     - install oapi-codegen development tools"
	@echo "generate        - generate code from OpenAPI spec"
	@echo "openapi-json    - convert OpenAPI YAML to JSON"
	@echo "validate-api    - validate API responses against spec"
	@echo "clean-generated - clean generated files"

	
test:
	go test ./... -coverprofile=c.out

coverage:
	go tool cover -html=c.out

migrate:
	go run cmd/migrate/migrate.go 

validate:
	go run cmd/validate/*

infra:
	docker-compose -f local/db-compose.yaml up

stop-infra:
	docker-compose -f local/db-compose.yaml down

audit:
	docker build . -t quickstarts:audit
	grype quickstarts:audit --fail-on medium --only-fixed

create-resource:
	./make_item.sh

# === oapi-codegen Migration Targets ===

# Install development tools
setup-tools:
	@echo "Installing oapi-codegen..."
	go mod download
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Generate code from OpenAPI specification
generate:
	@echo "Generating code from OpenAPI specification..."
	mkdir -p pkg/generated
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yaml spec/openapi.yaml

# Convert OpenAPI spec from YAML to JSON
openapi-json:
	@echo "Converting OpenAPI YAML to JSON..."
	go run cmd/yaml-to-json/main.go spec/openapi.yaml spec/openapi.json
	@echo "Generated spec/openapi.json from spec/openapi.yaml"

dev: generate
	@echo "Starting development server..."
	go run .

# Validate API responses match OpenAPI spec
validate-api:
	go run cmd/check-openapi-json/*

# Clean generated files
clean-generated:
	rm -rf pkg/generated/
	@echo "Generated files cleaned"
