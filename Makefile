OUT_DIR := build
OUT_BIN := api
CMD 	:= .

GOFLAGS := -trimpath
LDFLAGS := -ldflags="-s -w"

.PHONY: build run fmt vet lint security test check swagger install-tools

## build: compile the binary to ./build/api
build:
	@mkdir -p $(OUT_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(OUT_DIR)/$(OUT_BIN) $(CMD)

## run: run the app without compiling a binary
run:
	go run $(CMD)

## fmt: checks if all files are compliant with goimports' formatting
fmt:
	goimports -l .

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## security: run govulncheck
security:
	govulncheck ./...

## test: run integration and unit tests (requires Docker)
test:
	go test -race ./...

## check: run fmt, vet, lint, security, and tests
check: fmt vet lint security test

## swagger: generate swagger docs from annotations
swagger:
	swag init

## install-tools: install required tools for fmt, lint, etc targets
install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/swaggo/swag/cmd/swag@latest

## help: list available targets
help:
	@grep -E "^##" Makefile | sed "s/## //"