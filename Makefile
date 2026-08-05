.PHONY: help test lint lint-full build coverage clean fmt vet contract-test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run tests
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

coverage: test ## Show test coverage
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run incremental linters (set LINT_BASE in CI)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	@if [ -n "$(LINT_BASE)" ] && git rev-parse --verify "$(LINT_BASE)" >/dev/null 2>&1; then \
		golangci-lint run --new-from-rev="$(LINT_BASE)" ./...; \
	else \
		golangci-lint run ./...; \
	fi

lint-full: ## Run the full linter against the whole repository
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

contract-test: ## Run Go/Python contract comparison tests
	go test ./internal/session/contract -run Test

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

build: ## Build example binaries
	go build -o bin/simple_agent ./cmd/examples/simple_agent
	@echo "Binaries built in bin/"

clean: ## Clean build artifacts
	rm -rf bin/ coverage.txt coverage.html

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

.DEFAULT_GOAL := help
