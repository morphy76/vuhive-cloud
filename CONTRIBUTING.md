# Contributing to vuhive-cloud

Thank you for your interest in contributing to **`vuhive-cloud`**!

This guide outlines our development philosophy, repository structure, local development and container build workflow, testing standards, and pull request procedures.

---

## Documentation Ecosystem & Architecture Boundaries

To keep documentation clean, modular, and discoverable, documentation in this repository is strictly segregated by audience and lifecycle responsibility:

| Document | Role & Audience |
|---|---|
| **[`README.md`](./README.md)** | **Introduction & Overview**: System capabilities, architectural topology, roadmap, and quickstart overview. |
| **[`CONTRIBUTING.md`](./CONTRIBUTING.md)** | **Developer & Contributor Guide**: Local environment setup, container builds with `--load`, local cluster validation, coding standards, and PR guidelines. |
| **[`AI_DISCLOSURE.md`](./AI_DISCLOSURE.md)** | **Development Philosophy & AI Transparency**: Spec-Driven Development (SDD) paradigm, human vs. agent responsibility boundaries, and verification gates. |
| **[`deploy/helm/vuhive-cloud/README.md`](./deploy/helm/vuhive-cloud/README.md)** | **Control Plane Installation**: Production Helm deployment guide, configuration values reference, external secrets, RBAC, and security hardening. |
| **[`deploy/helm/vuhive-cloud-infra/README.md`](./deploy/helm/vuhive-cloud-infra/README.md)** | **Infrastructure Installation**: Quickstart backing services setup for evaluation (PostgreSQL + MinIO). |
| **[`docs/cookbook.md`](./docs/cookbook.md)** | **Adoption Guide & API Recipes**: End-to-end recipes for test engineers and platform teams packaging test suites, configuring runner profiles, scheduling CronJobs, dispatching runs, and querying KPIs. |
| **[`api/openapi.yaml`](./api/openapi.yaml)** | **REST API Reference**: Full OpenAPI 3.0.3 specification covering all control plane endpoints and data schemas. |
| **[`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md)** | **Architectural Specification**: Bounded contexts, DDD domain aggregates, database schema (DDL), and security postures. |

---

## Development Philosophy & AI Disclosure

`vuhive-cloud` follows **Spec-Driven Development (SDD)** and actively embraces autonomous coding agents for implementation:

- **Humans** define the architectural boundaries, security invariants, data models, and acceptance criteria.
- **Agents** execute TDD cycles (Red-Green-Refactor), mechanical plumbing, and code generation within rigorous constraints.
- All contributions—whether authored by human engineers or AI agents—must satisfy the same strict quality gates: zero compilation warnings, static compile-time interface assertions, zero race conditions, lint passing, and automated test coverage.

See [`AI_DISCLOSURE.md`](./AI_DISCLOSURE.md) for full details.

---

## Local Development Environment

### Prerequisites

- **Go**: Version `1.26`
- **Docker / Rancher Desktop**: Docker 29+ (or nerdctl) with BuildKit enabled
- **Kubernetes**: 1.28+ local cluster (Rancher Desktop, Kind, or Minikube)
- **Helm**: 3.10+ or 4+
- **kubectl**: Configured for your local Kubernetes cluster
- **golangci-lint**: Recommended for static code analysis

### Building Local Binaries

All binaries are compiled via the root `Makefile` with compile-time version metadata injected via `ldflags`:

```bash
# Build all local binaries (server, bff, runner-wrapper, runner-init)
make build

# Build individual binaries
make build-server
make build-bff
make build-runner-wrapper
make build-runner-init

# View all available targets and descriptions
make help
```

---

## Container Image Management & Local Cluster Testing

When testing changes against a local Kubernetes cluster (such as Rancher Desktop, Kind, or Minikube), standard `docker build` commands often cause `ImagePullBackOff` errors.

### Why `--load --provenance=false` is Required

Modern container engines (such as Rancher Desktop using Docker 29+ with the containerd snapshotter `io.containerd.snapshotter.v1`) use BuildKit by default. 

Running a standard `docker build -t image:local .` stores the image index in Buildx cache without exporting it into the CRI daemon image store. Consequently, Kubernetes cannot find the image locally and attempts to pull it from a remote registry, causing `ImagePullBackOff` (`pull access denied`).

To make container images immediately available to the local Kubernetes node without pushing to a remote registry:
1. `--load` instructs Buildx to export the built image to the Docker image store.
2. `--provenance=false` disables BuildKit multi-platform attestation manifests that can confuse older or single-platform CRI runtimes.

### Dedicated Makefile Targets

The root `Makefile` provides dedicated targets pre-configured with `--load --provenance=false` and build argument injection (`VERSION`, `COMMIT`, `BUILD_TIME`):

```bash
# Build all container images for local testing
make docker-build

# Build only the control plane server image (vuhive/server:local)
make docker-build-server

# Build only the runner init/wrapper image (vuhive/runner-init:local)
make docker-build-runner-init
```

You can customize the target tags via environment variables:

```bash
SERVER_IMAGE=myrepo/server:dev RUNNER_INIT_IMAGE=myrepo/runner-init:dev make docker-build
```

### Verifying Images with Rancher Desktop / Kubernetes

Once built with `make docker-build`, verify the images exist in your local image store:

```bash
docker images | grep vuhive
```

Expected output:
```text
vuhive/server        local    ...
vuhive/runner-init   local    ...
```

### Deploying Local Images with Helm

To deploy `vuhive-cloud` using your locally built images:

```bash
# 1. Install infrastructure dependencies (PostgreSQL + MinIO)
helm dependency build deploy/helm/vuhive-cloud-infra
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s

# 2. Deploy control plane with local image overrides
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set image.repository=vuhive/server \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent \
  --set runner.initImage=vuhive/runner-init:local \
  --wait --timeout=120s
```

> [!IMPORTANT]
> Ensure `image.pullPolicy=IfNotPresent` (or `Never`) so Kubernetes does not attempt to pull the `:local` tag from an external registry.

---

## Testing & Quality Assurance

We maintain strict Test-Driven Development (TDD) discipline:

1. **Red**: Write a failing test specifying the expected behavior and API contract first.
2. **Green**: Implement the minimal code required to pass the test.
3. **Refactor**: Clean up and optimize while ensuring all tests stay green.

### Running Tests

```bash
# Run unit tests
make test

# Run tests with the Go race detector
make test-race

# Run integration tests (requires Docker for testcontainers)
make test-integration

# Run benchmarks
make test-bench

# Run linter
make lint
```

---

## Coding Standards & Architectural Guidelines

### 1. Hexagonal Architecture & DDD Boundaries
- **Domain Layer (`domain/`)**: Pure business logic (aggregates, entities, value objects, domain errors). Must NEVER import application or adapter packages.
- **Application Layer (`application/`)**: Use case orchestration. Defines driving (`ports/inbound/`) and driven (`ports/outbound/`) interfaces. Must NEVER import adapter packages.
- **Adapters Layer (`adapters/`)**: Infrastructure implementations (PostgreSQL, MinIO, Kubernetes, Gin HTTP handlers). Converts between external DTOs and pure domain models.

### 2. Static Interface Verification
Every concrete struct implementing an inbound or outbound port interface MUST include a compile-time static type assertion:
```go
var _ outbound.RunRepository = (*PgxRunRepository)(nil)
```

### 3. Structured Logging (`zerolog`)
- Loggers must be context-aware: `zerolog.Ctx(ctx)`.
- Public functions follow the start/done pattern:
  - Enter: `log.Debug().Msg("starting <operation>")`
  - Exit: `log.Info().Dur("duration_ms", time.Since(start)).Msg("completed <operation>")`
- Never log credentials, API keys, or raw authentication tokens.

---

## Git & Pull Request Workflow

### Worktrees for Parallel Development
When working across multiple issues, use Git worktrees under `.worktrees/issue-<number>` to avoid branch collisions:

```bash
git worktree add -b <branch-name> .worktrees/issue-<number> origin/<branch-name>
```

### Commit Conventions
Follow the Conventional Commits specification:
- `feat(<scope>): description (closes #<issue>)`
- `fix(<scope>): description (closes #<issue>)`
- `docs(<scope>): description (closes #<issue>)`
- `refactor(<scope>): description`

### Pull Request Checklist
Before opening a pull request, verify:
- [ ] `make help` lists all targets cleanly.
- [ ] `make test` and `make test-race` pass with zero failures.
- [ ] `make lint` reports zero issues.
- [ ] All new structs satisfying interfaces have static compile-time assertions.
- [ ] Relevant documentation has been updated according to the documentation ecosystem boundaries.
