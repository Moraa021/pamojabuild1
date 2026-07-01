.PHONY: run build test clean

run:
	cd backend && go run cmd/app/main.go

build:
	cd backend && go build -o bin/pamoja cmd/app/main.go

test:
	cd backend && go test ./...

clean:
	cd backend && rm -rf bin/

migrate:
	cd backend && go run cmd/app/main.go --migrate

db-setup:
	createdb pamoja
	cd backend && go run cmd/app/main.go --migrate

deps:
	cd backend && go mod tidy
	cd backend && go mod download