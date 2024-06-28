help:
	@echo "Availabe commands:"
	@echo "------------------"
	@echo "test		- run tests"
	@echo "coverage	- open browser with detailed test coverage report"
	@echo "migrate		- run database migration"
	@echo "generate-spec	- run openAPI3 generator"
	@echo	"validate-topics - run help topics validator"
	@echo  "infra           - start required infrastructure"
	@echo "stop-infra      - stop required infrastructure"
	@echo "audit 		- run grype audit on the docker image"
	@echo "create-resource	- a cli tool to bootstrap a new learning resource"

	
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
	docker compose -f local/db-compose.yaml up

stop-infra:
	docker compose -f local/db-compose.yaml down

audit:
	docker build . -t quickstarts:audit
	grype quickstarts:audit --fail-on medium --only-fixed

create-resource:
	./make_item.sh
