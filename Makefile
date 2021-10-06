help:
	@echo "Availabe commands:"
	@echo "------------------"
	@echo "test		- run tests"
	@echo "coverage	- open browser with detailed test coverage report"
	@echo "migrate		- run database migration"
	@echo "generate-spec	- run openAPI3 generator"

	
test:
	go test ./... -coverprofile=c.out

coverage:
	go tool cover -html=c.out

migrate:
	go run cmd/migrate/migrate.go 

generate-spec:
	go run cmd/spec/main.go
