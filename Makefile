help:
	@echo "Availabe commands:"
	@echo "------------------"
	@echo "test		        - run tests"
	@echo "coverage	        - open browser with detailed test coverage report"
	@echo "migrate		    - run database migration"
	@echo "generate-spec	- run openAPI3 generator"
	@echo "validate-topics  - run help topics validator"
	@echo "infra            - start required infrastructure"
	@echo "stop-infra       - stop required infrastructure"
	@echo "audit 		    - run grype audit on the docker image"
	@echo "create-resource	- a cli tool to bootstrap a new learning resource"
	@echo "run_local        - run the API locally for manual debugging or testing"

	
test:
	go test ./... -coverprofile=c.out

coverage:
	go tool cover -html=c.out

migrate:
	go run cmd/migrate/migrate.go 

generate-spec:
	go run cmd/spec/main.go

validate:
	go run cmd/validate/*

infra:
	docker-compose -f local/db-compose.yaml up

stop-infra:
	docker-compose -f local/db-compose.yaml down

audit:
	docker build . -t quickstarts:audit
	grype quickstarts:audit --fail-on medium --only-fixed

run_local:
	@if [ ! -f env.local ]; then \
	printf "Setting up a local env for you as env.local"; \
	cp env.example env.local; \
	fi
	. env.local
	docker-compose -f local/db-compose.yaml down
	docker-compose -f local/db-compose.yaml up &
	go run cmd/migrate/migrate.go
	go run main.go

create-resource:
	./make_item.sh
