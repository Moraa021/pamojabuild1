BINARY=cmd/app
OUT=bin/app

.PHONY: build
build:
	cd backend && go build -o $(OUT) ./$(BINARY)

.PHONY: run
run:
	cd backend && go run ./$(BINARY)

.PHONY: test
test:
	cd backend && go test ./...
