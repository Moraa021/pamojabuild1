
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
test: test-unit test-integration

.PHONY: test-unit
test-unit:
	go test ./... -count=1

.PHONY: test-integration
test-integration:
	go test ./tests/integration -count=1

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

.PHONY: migrate
migrate:
	go run ./cmd/migrate

.PHONY: seed
seed:
	go run ./cmd/seed

.PHONY: clean
clean:
	rm -rf $(OUT)

