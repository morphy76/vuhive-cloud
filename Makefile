COMPONENT := vuhive
VERSION ?= $(shell cat VERSION.$(COMPONENT) 2>/dev/null || cat VERSION 2>/dev/null || echo "0.0.0")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
MODULE := github.com/morphy76/vuhive-cloud
LDFLAGS := -s -w \
  -X '$(MODULE)/internal/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
  -X '$(MODULE)/internal/version.BuildTime=$(BUILD_TIME)'

DOCKER ?= docker
SERVER_IMAGE ?= vuhive/server:local
BFF_IMAGE ?= vuhive/bff:local
RUNNER_INIT_IMAGE ?= vuhive/runner-init:local

.PHONY: all
all: build test ## Build and run tests

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: web-build build-server build-runner-wrapper build-runner-init build-bff build-cli ## Build all binaries and web assets

.PHONY: web-install
web-install: ## Install frontend dependencies via pnpm in web/
	pnpm --dir web install

.PHONY: web-build
web-build: ## Build frontend production assets with pnpm in web/
	pnpm --dir web build

.PHONY: web-test
web-test: ## Run frontend Vitest component tests in web/
	pnpm --dir web test

.PHONY: web-lint
web-lint: ## Run frontend linter and type checking in web/
	pnpm --dir web lint

.PHONY: build-web
build-web: web-build ## Alias for web-build

.PHONY: test-web
test-web: web-test ## Alias for web-test

.PHONY: lint-web
lint-web: web-lint ## Alias for web-lint

.PHONY: build-server
build-server: ## Build control plane server binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/server ./cmd/server

.PHONY: build-cli
build-cli: ## Build developer CLI binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/vuhive ./cmd/cli

.PHONY: build-bff
build-bff: ## Build Backend-For-Frontend (BFF) service binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/bff ./cmd/bff

.PHONY: build-runner-wrapper
build-runner-wrapper: ## Build runner wrapper binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/runner-wrapper ./cmd/runner-wrapper

.PHONY: build-runner-init
build-runner-init: ## Build runner init binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/runner-init ./cmd/runner-init

.PHONY: docker-build
docker-build: docker-build-server docker-build-bff docker-build-runner-init ## Build all container images with --load for local cluster testing

.PHONY: docker-build-server
docker-build-server: ## Build control plane server container image with --load
	$(DOCKER) build --load --provenance=false \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(SERVER_IMAGE) \
		-f deploy/docker/server.Dockerfile .

.PHONY: docker-build-bff
docker-build-bff: ## Build BFF service container image with --load
	$(DOCKER) build --load --provenance=false \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(BFF_IMAGE) \
		-f deploy/docker/bff.Dockerfile .

.PHONY: docker-build-runner-init
docker-build-runner-init: ## Build runner-init container image with --load
	$(DOCKER) build --load --provenance=false \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(RUNNER_INIT_IMAGE) \
		-f deploy/docker/runner-init.Dockerfile .

.PHONY: docker-prune
docker-prune: ## Prune Docker build cache to prevent Kubelet ImageGC eviction on local clusters
	$(DOCKER) builder prune -f


.PHONY: test
test: ## Run unit tests
	go test -v ./...

.PHONY: test-race
test-race: ## Run unit tests with race detector
	go test -v -race ./...

.PHONY: test-bff
test-bff: ## Run Go BFF unit tests with race detector
	go test -v -race ./cmd/bff/... ./internal/bff/...

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

.PHONY: lint-bff
lint-bff: ## Run golangci-lint on BFF code
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./cmd/bff/... ./internal/bff/... || echo "golangci-lint not installed"

.PHONY: generate
generate: ## Run go generate
	go generate ./...

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
