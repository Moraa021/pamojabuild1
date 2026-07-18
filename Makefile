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
	cd backend && go run ./cmd/migrate up

migrate-down:
	cd backend && go run ./cmd/migrate down 1

migration-create:
	@test -n "$(NAME)" || (echo "NAME is required, for example: make migration-create NAME=add_task_states" && exit 1)
	cd backend && migrate create -ext sql -dir db/migrations $(NAME)

db-setup:
	createdb pamoja
	cd backend && go run ./cmd/migrate up

deps:
	cd backend && go mod tidy
	cd backend && go mod download
