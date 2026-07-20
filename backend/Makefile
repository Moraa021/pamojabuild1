
BINARY=cmd/app
OUT=bin/app

.PHONY: all
all: build

.PHONY: build
build:
	go build -o $(OUT) ./cmd/app

.PHONY: run
run:
	go run ./cmd/app

.PHONY: test
test:
	go test ./... -count=1

.PHONY: test-unit
test-unit:
	go test ./... -count=1

.PHONY: test-postgres
test-postgres:
	@test -n "$(TEST_DATABASE_URL)" || (echo "TEST_DATABASE_URL is required" && exit 1)
	go test ./cmd/app ./internal/authorization ./internal/lightning/repository ./internal/task/repository -count=1

.PHONY: test-e2e
test-e2e:
	@echo "No e2e tests defined"

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: lint
lint:
	@echo "Running go vet..."
	go vet ./...

.PHONY: swagger
swagger:
	@echo "Run 'swag init' in project root if swag is installed"

.PHONY: migrate-up
migrate-up:
	go run ./cmd/migrate up

.PHONY: migrate-down
migrate-down:
	go run ./cmd/migrate down 1

.PHONY: migrate-version
migrate-version:
	go run ./cmd/migrate version

.PHONY: migration-create
migration-create:
	@test -n "$(NAME)" || (echo "NAME is required, for example: make migration-create NAME=add_task_states" && exit 1)
	migrate create -ext sql -dir db/migrations $(NAME)

.PHONY: seed
seed:
	go run ./cmd/seed

.PHONY: clean
clean:
	rm -rf $(OUT)
