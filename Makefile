# Pinned tool versions.
GOLANGCI_LINT_VERSION := v2.12.2
PROTOC_GEN_GO_VERSION := v1.36.11

GOBIN := $(shell go env GOPATH)/bin

.PHONY: tools generate generate-proto lint lint-go lint-go-fix test build fmt help

## tools: Install pinned development tools (golangci-lint, protoc-gen-go)
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

## generate: Regenerate all generated code
generate: generate-proto
	go generate ./...

## generate-proto: Regenerate Go code from proto definitions (requires protoc)
generate-proto:
	PATH="$(GOBIN):$$PATH" protoc \
		-I internal/testproto \
		--go_out=paths=source_relative:internal/testproto \
		internal/testproto/book.proto

## lint: Run all linters
lint: lint-go

## lint-go: Run Go linters
lint-go:
	@echo "Running golang linter..."
	go vet ./...
	@if [ -f .golangci.yml ] || [ -f .golangci.yaml ]; then golangci-lint run ./...; \
	else echo "no .golangci.yml yet; skipping golangci-lint"; fi

## lint-go-fix: Run Go linter and auto-fix issues when possible
lint-go-fix:
	@echo "Running golang linter with auto-fix..."
	golangci-lint run --fix ./...

## test: Run Go unit tests
test:
	@echo "Running unit tests..."
	go test ./... -coverprofile=coverage.out -covermode=atomic

## build: Build all Go packages
build:
	@echo "Building Go packages..."
	go build ./...

## fmt: Format Go code
fmt:
	@echo "Running formatter..."
	golangci-lint fmt ./...

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
