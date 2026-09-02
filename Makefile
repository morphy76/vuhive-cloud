COMPONENT := vuhive
VERSION ?= $(shell cat VERSION.$(COMPONENT) 2>/dev/null || echo "0.0.0")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
MODULE := github.com/morphy76/vuhive-cloud
LDFLAGS := -s -w \
  -X '$(MODULE)/internal/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
  -X '$(MODULE)/internal/version.BuildTime=$(BUILD_TIME)'

.PHONY: all
all: build test ## Build and run tests

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: build-server build-runner-wrapper ## Build all binaries

.PHONY: build-server
build-server: ## Build control plane server binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: build-runner-wrapper
build-runner-wrapper: ## Build runner wrapper binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/runner-wrapper ./cmd/runner-wrapper

.PHONY: test
test: ## Run unit tests
	go test -v ./...

.PHONY: test-race
test-race: ## Run unit tests with race detector
	go test -v -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v -tags=integration ./...

.PHONY: test-bench
test-bench: ## Run benchmarks
	go test -bench=. -run=^$$ ./...

.PHONY: test-examples
test-examples: ## Build and verify examples
	@echo "No examples to build yet"

.PHONY: lint
lint: ## Run golangci-lint
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed"

.PHONY: generate
generate: ## Run go generate
	go generate ./...

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
