# vuhive-cloud Project Documentation & Setup Package

This repository contains the implementation and architecture documentation for **`vuhive-cloud`** (`github.com/morphy76/vuhive-cloud`) — a Kubernetes-native control plane for orchestrating distributed load testing suites and runner jobs.

Project roadmaps, epics, and implementation tasks are tracked directly via the [GitHub Issues Tracker](https://github.com/morphy76/vuhive-cloud/issues) and [GitHub Milestones](https://github.com/morphy76/vuhive-cloud/milestones).

---

## Key Features

- 🛠 **Ephemeral Source-to-Binary Compilation**: Upload raw Go source archives (`go.mod` + scenario code); `vuhive-cloud` dynamically spins up isolated Kubernetes build jobs (`golang:1.26-alpine`) to cross-compile static binaries targeting `linux/amd64` or `linux/arm64`. Failed compilations can be retried immediately by re-uploading corrected sources — the control plane automatically resets the artifact state and prunes the stale Kubernetes Job.
- 🔒 **Hardened Execution Isolation**: Test runner pods comply with the Kubernetes **Restricted** Pod Security Standards (non-root UID `10001`, read-only root filesystems, all Linux capabilities dropped, zero host privilege).
- 🧩 **Reusable Runner Profiles**: Decouple test scenario code from infrastructure scheduling. Define reusable profiles specifying CPU/memory requests and limits, node selectors, tolerations, and node affinities for targeted execution.
- ⏰ **Native Kubernetes CronJob Scheduling**: Declarative scheduling mapped 1-to-1 to native Kubernetes `batch/v1` `CronJob`s with standard cron syntax (`0 2 * * *`), eliminating external scheduler dependencies.
- 📊 **Automated KPI Indexing & SLA Verification**: Automatically parses deterministic execution reports (`summary.json`), extracting and indexing latency percentiles ($p_{50}$, $p_{90}$, $p_{95}$, $p_{99}$), throughput (TPS), error rates, and SLA pass/fail status into PostgreSQL.
- 📈 **Execution Reports, Logs & Metrics Query API**: Query and filter historical runs by suite, schedule, status, and date range. Fetch indexed performance KPIs, full deterministic execution reports (`summary.json`), and runner stdout/stderr logs directly or as presigned S3 download URLs. Fully documented via [OpenAPI 3.1](./api/openapi.yaml), served directly by the control plane (`GET /openapi.yaml` and `GET /openapi.json`), and interactively testable via the optional Swagger UI viewer in the infrastructure chart.
- 📦 **Pluggable Object Storage**: Integrates seamlessly with AWS S3 or MinIO for long-term retention of source packages, compiled binaries, full execution logs, and detailed performance summaries.
- ⏱ **Distributed Start Barrier Synchronization**: Built-in rendezvous coordinator guarantees multi-pod distributed load generators synchronize and fire simultaneously without clock skew.
- 🛑 **Execution Lifecycle Control & Graceful Abort**: Monitor active runs in real time and abort executions on demand (`POST /api/v1/runs/{id}/abort`), instantly tearing down Kubernetes workloads while propagating SIGTERM for partial log flush, updating state to `ABORTED` with audited cancellation metadata, and reclaiming cluster resources.
- 🌐 **Embedded React 19 SPA & PWA File Server**: The Go Backend-For-Frontend (`cmd/bff`) serves the compiled React 19 web dashboard and PWA assets directly using Go `embed.FS`, eliminating separate web server containers (e.g. Nginx). Includes SPA fallback routing to `index.html`, immutable HTTP caching headers for hashed assets (`/assets/*`), service worker / manifest delivery, and live-reload reverse proxying (`--dev-proxy-url`) for local Vite frontend development.

---

## Architecture at a Glance

`vuhive-cloud` follows Hexagonal Architecture (Ports & Adapters) and Domain-Driven Design (DDD) principles:

```text
┌─────────────────────────────────┐   ┌─────────────────────────────────┐
│   React 19 PWA Web Interface    │   │     API Client / CI/CD Pipeline │
└────────────────┬────────────────┘   └────────────────┬────────────────┘
                 │ HTTP REST / SSE                     │ HTTP REST (Gin)
                 ▼                                     │
┌─────────────────────────────────┐                    │
│ Backend-For-Frontend (cmd/bff)  │                    │
└────────────────┬────────────────┘                    │
                 │ HTTP (Aggregation & Gateway)        │
                 └─────────────────►┌──────────────────▼─────────────────────────────────────────────────────────────────────┐
                                    │ vuhive-cloud Control Plane (cmd/server)                                                │
                                    │                                                                                        │
                                    │   ┌────────────────────────┐   ┌──────────────────────────┐   ┌────────────────────┐   │
                                    │   │ Build Service          │   │ Profile & Run Service    │   │ Schedule Service   │   │
                                    │   └───────────┬────────────┘   └────────────┬─────────────┘   └─────────┬──────────┘   │
                                    │               │                             │                           │              │
                                    │               │ (Ephemeral Builds)          │ (Runner Jobs / Abort)     │ (CronJobs)   │
                                    │               ▼                             ▼                           ▼              │
                                    │   ┌────────────────────────────────────────────────────────────────────────────────┐   │
                                    │   │ Kubernetes Client Orchestrator (batch/v1 Jobs, CronJobs, Informer Watcher)     │   │
                                    │   └────────────────────────────────────────────────────────────────────────────────┘   │
                                    │               │                             │                           │              │
                                    │               ▼                             ▼                           ▼              │
                                    │      PostgreSQL (pgx)                 S3 / MinIO Storage          Barrier Coordinator  │
                                    │      - Test Suites                    - Source Archives           - Worker Rendezvous  │
                                    │      - Runner Profiles                - Compiled Binaries         - Sync Start Delay   │
                                    │      - Cron Schedules                 - Run Logs                                       │
                                    │      - Test Runs & KPIs               - summary.json Reports                           │
                                    └────────────────────────────────────────────────────────────────────────────────────────┘
```

For complete architectural specifications, DDD aggregate boundaries, and database schemas, see **[`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md)**.

---

## Development Philosophy & AI Disclosure

`vuhive-cloud` is engineered using **Spec-Driven Development (SDD)** and actively fosters the adoption of autonomous coding agents.

- **The Human "What"**: Human engineers define the specifications, domain boundaries, security invariants, architectural contracts, and acceptance criteria.
- **The Agent "How"**: Autonomous coding agents execute implementation details, strict TDD cycles (Red-Green-Refactor), plumbing, and mechanical refactoring.
- **Strict Quality Enforcement**: AI generation is never unchecked—every change is subject to static compile-time interface assertions, race condition detection, automated test suites, linters, and mandatory human review.

For complete details on our development methodology, human oversight model, and guidelines for contributing with AI tools, see **[`AI_DISCLOSURE.md`](./AI_DISCLOSURE.md)**.

---

## Documentation Ecosystem & Navigation

| Document | Role & Audience | Focus & Boundary |
|---|---|---|
| **[`README.md`](./README.md)** | **Introduction & Overview** | Introduces system capabilities, architectural topology, and project roadmap. |
| **[`CONTRIBUTING.md`](./CONTRIBUTING.md)** | **Developer & Contributor Guide** | Local environment setup, container builds with `--load`, local cluster validation, coding standards, and PR guidelines. |
| **[`AI_DISCLOSURE.md`](./AI_DISCLOSURE.md)** | **Engineering Philosophy & AI Disclosure** | Spec-Driven Development (SDD) paradigm, human vs. agent responsibility division, and quality gates. |
| **[`deploy/helm/vuhive-cloud/README.md`](./deploy/helm/vuhive-cloud/README.md)** | **Control Plane Installation** | Installs control plane on Kubernetes; configuration values, external secrets, RBAC, and security hardening. |
| **[`deploy/helm/vuhive-cloud-infra/README.md`](./deploy/helm/vuhive-cloud-infra/README.md)** | **Infrastructure Installation** | Installs backing evaluation services (PostgreSQL + MinIO + optional OpenAPI Swagger UI viewer). |
| **[`docs/cookbook.md`](./docs/cookbook.md)** | **Adoption Guide & Recipes** | Dedicated to `vuhive-cloud` adoption: end-to-end recipes for packaging test suites, profiles, schedules, and runs. |
| **[`api/openapi.yaml`](./api/openapi.yaml)** | **REST API Reference** | Documents the APIs: complete OpenAPI 3.1 contract served live by control plane (`GET /openapi.yaml`, `GET /openapi.json`). |
| **[`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md)** | **Architectural Specification** | Bounded contexts, DDD domain aggregates, database schema (DDL), and security postures. |

---

## Quickstart

Get up and running locally on Rancher Desktop, Kind, or Minikube in three steps:

### 1. Deploy Backing Infrastructure (PostgreSQL + MinIO)
Deploy backing services via the local infrastructure chart:

```bash
# 1. Add chart repositories
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

# 2. Deploy infrastructure (PostgreSQL + MinIO + optional Swagger UI)
helm dependency build deploy/helm/vuhive-cloud-infra
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s
```

> For details on database, storage parameters, and enabling the optional OpenAPI viewer (Swagger UI), see the [Infrastructure Helm Installation Guide (`deploy/helm/vuhive-cloud-infra/README.md`)](./deploy/helm/vuhive-cloud-infra/README.md).

### 2. Deploy vuhive-cloud Control Plane
Deploy the control plane connected to the local infrastructure:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --wait --timeout=120s
```

> **MinIO Note**: Setting `s3.endpoint` automatically enables path-style S3 addressing in the control plane server, runner-init, and runner-wrapper — no extra flag needed. See the [Control Plane Helm Installation Guide (`deploy/helm/vuhive-cloud/README.md`)](./deploy/helm/vuhive-cloud/README.md) for full configuration reference and production deployment options.

### 3. Verify Health, Version & OpenAPI Endpoints
 
Port-forward the control plane service:

```bash
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080
```

Verify service liveness, version, and OpenAPI specifications:

```bash
# Check service health
curl -i http://localhost:8080/healthz

# Inspect runtime version
curl -i http://localhost:8080/version

# Fetch machine-readable OpenAPI 3.1 specification
curl -i http://localhost:8080/openapi.json
curl -i http://localhost:8080/openapi.yaml
```

To create your first runner profile, upload test suites, and trigger runs, follow the **[Adoption Cookbook (`docs/cookbook.md`)](./docs/cookbook.md)**. For full REST API endpoint specifications, refer to the **[OpenAPI Reference (`api/openapi.yaml`)](./api/openapi.yaml)** or fetch it live at `/openapi.yaml` / `/openapi.json`.

---

## Repository Structure & Development Guide

```text
.
├── api/
│   ├── openapi.yaml            # OpenAPI 3.1 REST API specification (YAML)
│   ├── openapi.json            # OpenAPI 3.1 REST API specification (JSON)
│   └── spec.go                 # Embedded specification assets (embed.FS)
├── cmd/
│   ├── bff/                    # Backend-For-Frontend service (React 19 PWA aggregation & API gateway)
│   ├── server/                 # Control plane REST server & migration entrypoint
│   ├── runner-init/            # Runner pod init container (downloads binary & config from S3)
│   └── runner-wrapper/         # Runner entrypoint wrapper (executes workload, captures KPIs)
├── internal/
│   ├── bff/                    # BFF service hexagonal hierarchy (domain models, ports, adapters)
│   ├── domain/                 # Pure domain layer (models, value objects, events, errors)
│   ├── application/            # Use case orchestration layer (inbound/outbound ports & services)
│   ├── adapters/               # Infrastructure adapters (PostgreSQL pgx, S3 MinIO, K8s, REST)
│   └── version/                # Compile-time version metadata
├── deploy/
│   ├── docker/                 # Production Dockerfiles (server.Dockerfile, runner-init.Dockerfile)
│   └── helm/
│       ├── vuhive-cloud/       # Control plane Helm chart
│       └── vuhive-cloud-infra/ # Backing infrastructure Helm chart (PostgreSQL + MinIO)
└── docs/
    └── cookbook.md             # End-to-end adoption cookbook & API recipes
```

### Development & Make Targets

Build and test commands are driven by the root `Makefile`:

```bash
# Build all local binaries (server, bff, runner-wrapper, runner-init)
make build

# Build standalone BFF service binary
make build-bff

# Build container images with --load for local cluster testing (Rancher Desktop)
make docker-build

# Or build individual images
make docker-build-server
make docker-build-runner-init

# Run unit tests
make test

# Run tests with race detection
make test-race

# Run linter
make lint

# View all available targets
make help
```

For detailed instructions on local cluster validation, BuildKit containerd image loading, and coding standards, see **[`CONTRIBUTING.md`](./CONTRIBUTING.md)**.

### Running the Go BFF & Web Dashboard Locally

```bash
# Run BFF with embedded SPA assets
./bin/bff --port=8081 --control-plane-url=http://localhost:8080

# Run BFF during frontend development with Vite HMR reverse proxying
./bin/bff --port=8081 --control-plane-url=http://localhost:8080 --dev-proxy-url=http://localhost:5173

# Run BFF with a local static build directory override
./bin/bff --port=8081 --control-plane-url=http://localhost:8080 --static-dir=./web/dist
```

---

## License

This project is licensed under the [MIT License](./LICENSE).
The control plane exposes its REST API at port `8080`. See [`api/openapi.yaml`](./api/openapi.yaml) for the full API reference.
For detailed Helm configuration options, see the chart READMEs:
- [`deploy/helm/vuhive-cloud-infra/README.md`](./deploy/helm/vuhive-cloud-infra/README.md) — infrastructure (PostgreSQL + MinIO + optional Swagger UI viewer)
- [`deploy/helm/vuhive-cloud/README.md`](./deploy/helm/vuhive-cloud/README.md) — control plane (namespace management, RBAC modes, all parameters)

## Documents in this Package

1. **[ARCHITECTURE_SPEC.md](./ARCHITECTURE_SPEC.md)**
   - **Executive Summary & System Vision**
   - **System Architecture & Topology Diagram** (Control Plane, PostgreSQL, S3/MinIO, Ephemeral Build Jobs, Runner Pods, Node Affinities/Tolerations)
   - **Hexagonal Architecture & Package Layout** (DDD Boundaries, Domain Aggregates, Inbound/Outbound Ports)
   - **Detailed Execution Workflows** (Source Upload -> Pre-Build AST Static Analysis & Framework Enforcement -> Ephemeral Compilation -> S3 Storage -> K8s Runner Job / Native CronJob -> Ingestion)
   - **PostgreSQL Database Schema (DDL)** (`test_suites`, `artifacts`, `configurations`, `runner_profiles`, `schedules`, `test_runs`)
   - **REST API Contract & Endpoints**
   - **Kubernetes Runner Pod Specification & Hardening** (Pod Security Standards restricted profile, Egress NetworkPolicies, Init-container artifact fetch, emptyDir mount, execution wrapper)
   - **Roadmap & Epic Breakdown** (Direct references to GitHub Milestones and Issues)

2. **[api/openapi.yaml](./api/openapi.yaml)**
   - Full OpenAPI 3.1 specification for all REST API endpoints exposed by the control plane (served live at `/openapi.yaml` and `/openapi.json`).

## Project Tracking & Roadmap

All work is organized across three primary milestones on GitHub:

- **[Milestone 1: Core Foundation & Single-Runner Cloud Engine](https://github.com/morphy76/vuhive-cloud/milestone/1)**
  - Epic 1.1: Core Foundation, Domain Models & Data Layer (#1, #2, #3, #26)
  - Epic 1.2: Source-to-Binary Compilation & Framework Enforcement (#4, #5, #22)
  - Epic 1.3: Runner Pod Orchestration, Profiles & Security Hardening (#6, #7, #8, #23, #25)
  - Epic 1.4: Scheduling, Reporting & CLI (#9, #10, #20, #24, #80)
  - Epic 1.5: Deployment, CI/CD & Infrastructure Packaging (#21, #27, #28, #53, #81)
- **[Milestone 1.5: Control Plane Web Interface & Go BFF (React 19 PWA)](https://github.com/morphy76/vuhive-cloud/milestone/4)**
  - Epic 1.5.1: Go BFF Architecture, Embedded Assets & API Gateway (#57, #58, #59, #60)
  - Epic 1.5.2: React 19 Application Scaffolding, Design System & PWA Shell (#61, #62, #63)
  - Epic 1.5.3: Inline Documentation & Contextual Recipe Guidance (#64, #65)
  - Epic 1.5.4: Test Suite & Artifact Build Management Views (#66, #67, #68)
  - Epic 1.5.5: Runner Profiles & Execution Orchestration (#69, #70, #71)
  - Epic 1.5.6: Performance Analytics, Telemetry & Log Inspection (#72, #73, #74, #75)
  - Epic 1.5.7: Packaging, CI/CD, Containerization & Helm Deployment (#76, #77, #78, #79)
- **[Milestone 2: Distributed Multi-Pod Coordination & Live Streaming](https://github.com/morphy76/vuhive-cloud/milestone/2)**
  - Epic 2.1: Distributed Multi-Pod Coordination (#11, #12, #13)
  - Epic 2.2: Live Telemetry Streaming (#14, #15)
- **[Milestone 3: Multi-Namespace, Multi-Cluster & Enterprise SSO](https://github.com/morphy76/vuhive-cloud/milestone/3)**
  - Epic 3.1: Multi-Namespace & Multi-Cluster Dispatcher (#16, #17)
  - Epic 3.2: Enterprise Authentication & RBAC (#18, #19)

---

## License

This project is licensed under the [MIT License](./LICENSE).
