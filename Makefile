help:
	@echo "Availabe commands:"
	@echo "------------------"
	@echo "test	- run tests"
	@echo "migrate	- run database migration"
	@echo "generate-spec	- run openAPI3 generator"

	
test:
	go test ./... -coverprofile fmtcoverage.html fmt

migrate:
	go run cmd/migrate/migrate.go 

generate-spec:
	go run cmd/spec/main.go
