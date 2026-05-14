OUT_DIR := build
OUT_BIN := api
CMD 	:= .

GOFLAGS := -trimpath
LDFLAGS := -ldflags="-s -w"

.PHONY: build run fmt lint vuln test check swagger install-tools

## build: compile the binary to ./build/api
build:
	@mkdir -p $(OUT_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(OUT_DIR)/$(OUT_BIN) $(CMD)

## run: run the app without compiling a binary
run:
	go run $(CMD)

## fmt: checks if all files are compliant with goimports' formatting
fmt:
	golangci-lint fmt --diff ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## vuln: run govulncheck
vuln:
	govulncheck ./...

## test: run integration and unit tests (requires Docker)
test:
	go test -race -shuffle=on -timeout=5m ./...

## check: run fmt, lint, vuln, and tests
check: fmt lint vuln test

## swagger: generate swagger docs from annotations
swagger:
	swag init

## install-tools: install required tools for fmt, lint, etc targets
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/swaggo/swag/cmd/swag@latest

## help: list available targets
help:
	@grep -E "^##" Makefile | sed "s/## //"