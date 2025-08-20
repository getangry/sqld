.PHONY: help test test-coverage test-race lint fmt vet build clean deps check-deps

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
deps: ## Download dependencies
	go mod download
	go mod verify

test: ## Run tests
	go test -v ./...

test-race: ## Run tests with race detector
	go test -v -race ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

build: ## Build the project
	go build -v ./...

clean: ## Clean build artifacts
	go clean ./...
	rm -f coverage.txt coverage.html

check: deps vet lint test ## Run all checks (deps, vet, lint, test)

ci: check test-race test-coverage ## Run CI pipeline locally

# Git hooks
install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	@echo '#!/bin/sh\nmake check' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed successfully"

# Release helpers
tag: ## Create a new git tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Usage: make tag VERSION=v1.0.0"; exit 1; fi
	git tag $(VERSION)
	git push origin $(VERSION)

# Documentation
docs: ## Generate and serve documentation
	godoc -http=:6060

# Performance
profile-cpu: ## Run CPU profiling
	go test -cpuprofile=cpu.prof -bench=. ./...
	go tool pprof cpu.prof

profile-mem: ## Run memory profiling
	go test -memprofile=mem.prof -bench=. ./...
	go tool pprof mem.prof