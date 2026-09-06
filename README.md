# vuhive-cloud Project Documentation & Setup Package

This repository contains the implementation and architecture documentation for **`vuhive-cloud`** (`github.com/morphy76/vuhive-cloud`) — a Kubernetes-native control plane for orchestrating distributed load testing suites and runner jobs.

Project roadmaps, epics, and implementation tasks are tracked directly via the [GitHub Issues Tracker](https://github.com/morphy76/vuhive-cloud/issues) and [GitHub Milestones](https://github.com/morphy76/vuhive-cloud/milestones).

---

## Key Features

- 🛠 **Ephemeral Source-to-Binary Compilation**: Upload raw Go source archives (`go.mod` + scenario code); `vuhive-cloud` dynamically spins up isolated Kubernetes build jobs (`golang:1.26-alpine`) to cross-compile static binaries targeting `linux/amd64` or `linux/arm64`. Failed compilations can be retried immediately by re-uploading corrected sources — the control plane automatically resets the artifact state and prunes the stale Kubernetes Job.
- 🛡️ **Pre-Build AST Static Analysis & Framework Enforcement**: Mandatory adoption of `github.com/morphy76/vuhive` enforced at the pre-build stage. Uploaded source archives undergo automated static inspection (`go/parser` and `go/ast`) to verify direct `vuhive` dependency in `go.mod` and validate implementation of the `vuhive.Scenario` contract. Inverted control enforces `package scenario`, strictly prohibiting user-defined `package main` or `func main()` to prevent arbitrary code execution, crypto-mining, and rogue background daemons. A package import blocklist rejects forbidden libraries (`os/exec`, `syscall`, `unsafe`, `plugin`, `runtime/cgo`, `golang.org/x/sys`) before compilation, with configurable cluster deployment overrides and "dangerous" risk tagging. Validated scenarios have a trusted, platform-managed `main.go` execution driver injected dynamically, guaranteeing that runtime CLI flags (`--summary-export`, `--config`) and OS signal traps (`SIGINT`, `SIGTERM`) cannot be bypassed.
- 🔒 **Hardened Execution Isolation**: Test runner pods comply with the Kubernetes **Restricted** Pod Security Standards (non-root UID `10001`, read-only root filesystems, all Linux capabilities dropped, zero host privilege).
- 🧩 **Reusable Runner Profiles**: Decouple test scenario code from infrastructure scheduling. Define reusable profiles specifying CPU/memory requests and limits, node selectors, tolerations, and node affinities for targeted execution.
- 🚀 **Ad-Hoc Test Run Dispatching via REST API**: Trigger immediate on-demand test executions (`POST /api/v1/runs`) allowing developers and CI/CD pipelines to validate scenarios instantly. The control plane validates active suites, compiled artifacts, and runner profiles, creates a persistent `TestRun` in `QUEUED` status, and manifests an ephemeral Kubernetes `batch/v1` Job.
- ⏰ **Native Kubernetes CronJob Scheduling & Run Correlation**: Declarative scheduling mapped 1-to-1 to native Kubernetes `batch/v1` `CronJob`s with standard cron syntax (`0 2 * * *`), eliminating external scheduler dependencies. Each CronJob-spawned Job is automatically correlated to a `TestRun` record via `k8s_job_name`: the informer watcher auto-creates the record and stores the actual `k8s_namespace` (not a hard-coded default). Completion callbacks resolve `run_id` first by UUID, then by `k8s_job_name` as a fallback for CronJob-spawned pods.
- 📊 **Automated KPI Indexing & SLA Verification**: Automatically parses deterministic execution reports (`summary.json`), extracting and indexing latency percentiles ($p_{50}$, $p_{90}$, $p_{95}$, $p_{99}$), throughput (TPS), error rates, and SLA pass/fail status into PostgreSQL.
- 📈 **Execution Reports, Logs & Metrics Query API**: Query and filter historical runs by suite, schedule, status, and date range. Fetch indexed performance KPIs, full deterministic execution reports (`summary.json`), and runner stdout/stderr logs directly or as presigned S3 download URLs. Fully documented via [OpenAPI 3.1](./api/openapi.yaml), served directly by the control plane (`GET /openapi.yaml` and `GET /openapi.json`), and interactively testable via the optional Swagger UI viewer in the infrastructure chart.
- 📦 **Pluggable Object Storage**: Integrates seamlessly with AWS S3 or MinIO for long-term retention of source packages, compiled binaries, full execution logs, and detailed performance summaries.
- ⏱ **Distributed Start Barrier Synchronization**: Built-in rendezvous coordinator guarantees multi-pod distributed load generators synchronize and fire simultaneously without clock skew.
- 🛑 **Execution Lifecycle Control & Graceful Abort**: Monitor active runs in real time and abort executions on demand (`POST /api/v1/runs/{id}/abort`), instantly tearing down Kubernetes workloads while propagating SIGTERM for partial log flush, updating state to `ABORTED` with audited cancellation metadata, and reclaiming cluster resources.
- 🔄 **Resilient Runner Callbacks & DNS Mitigation**: Injected runner wrappers reliably post completion telemetry via `POST /api/v1/runs/complete`. The `run_id` field accepts either a domain UUID (ad-hoc runs) or a Kubernetes Job name (CronJob-spawned runs); the control plane resolves either transparently. Helm orchestration automatically mitigates Kubernetes `ndots:5` search domain hijacking by provisioning unqualified service endpoints within shared namespaces or trailing-dot absolute FQDNs (`.svc.cluster.local.`) across namespaces.
- 🌐 **Modern React 19 Web Interface & PWA**: The official frontend (`web/`) built with React 19, Vite 6, TypeScript, and zero-runtime Tailwind CSS v4. Features an adaptive responsive shell with a collapsible sidebar for desktop ($\ge 1024\text{px}$), a touch-gesture slide-out drawer for tablets ($768\text{px}-1023\text{px}$), and a fixed bottom navigation bar for mobile ($<768\text{px}$). Ships an accessible design system built on **Radix UI** headless primitives (Dialog, Tooltip, Popover, DropdownMenu, Tabs, Select, Toast, Switch) conforming to **WCAG 2.1 AA** with full keyboard navigation, visible focus rings, semantic ARIA patterns, screen reader landmarks, and status badges using icon+text (not color alone per WCAG 1.4.1). Includes dark/light theme switching, vendor-split asset chunking (`vendor-react`, `vendor-ui`, `vendor-radix`), automated **axe-core** accessibility auditing in CI, and zero-overhead delivery directly via the Go BFF's embedded `embed.FS` file server with immutable HTTP caching.
- 💡 **Inline Tooltip Documentation & Domain Concept Help Badges**: Comprehensive accessible micro-guidance across all forms, technical input fields, and metric badges eliminate configuration guesswork and accelerate onboarding. Features reusable `<HelpTooltip />` hover/focus popover triggers and `<InfoBadge />` callout banners for critical domain invariants:
  - **Build Upload & AST Enforcement**: Explains required `go.mod` dependency (`github.com/morphy76/vuhive`), inverted control (`package scenario`), strictly prohibited `main()` drivers, and forbidden system packages (`os/exec`, `syscall`, `unsafe`, `plugin`, `runtime/cgo`).
  - **Runner Profiles & Kubernetes Sizing**: Explains CPU millicores (`1000m`), binary memory SI units (`1Gi`), node tolerations syntax (`key=value:NoSchedule`), and Guaranteed QoS resource allocation.
  - **Cron Scheduling**: Standard 5-field CRON expression builder with quick presets (Hourly, Nightly, Weekly) and cluster UTC clock timezone alignment.
  - **Performance KPIs & SLAs**: Clear definitions for latency percentiles ($p_{50}$, $p_{90}$, $p_{95}$, $p_{99}$), throughput (TPS), error rate thresholds, and distributed rendezvous barrier synchronization.
- ⚡ **Backend-For-Frontend (BFF) Composite Aggregation & Reverse Proxy**: High-throughput composite gateway (`cmd/bff`) tailored for web and operational dashboards. Features concurrent fan-out aggregation (`GET /api/bff/v1/dashboard`) executing sub-50ms parallel queries across system health, active test runs count, recent suites, and runner profiles; unified run detail endpoints (`GET /api/bff/v1/runs/{id}`) combining run execution status, indexed performance KPIs (TPS, latency percentiles $p_{50}\dots p_{99}$, error rates), and pre-signed S3 download URLs for reports and logs; and transparent reverse proxying for entity CRUD operations (`/api/bff/v1/suites`, `/api/bff/v1/profiles`, `/api/bff/v1/schedules`, `/api/bff/v1/runs`) with header propagation and connection pooling.
- 📡 **Real-Time Server-Sent Events (SSE) Telemetry Stream**: The BFF exposes a persistent event stream (`GET /api/bff/v1/events` and backwards-compatible `/api/v1/bff/events`) delivering real-time execution state transitions without high-frequency browser polling. Streams typed event frames including `run_status_changed` (live progression through `QUEUED`, `RUNNING`, `COMPLETED`, `FAILED`, `ABORTED` with latency percentiles and error rates), `build_status_changed` (`QUEUED`, `BUILDING`, `READY`, `FAILED`), and periodic `system_heartbeat` keeping firewalls and proxies alive. Implemented with a non-blocking fan-out hub, per-client buffered channels, and proactive context cancellation monitoring to prevent goroutine leaks.
- 🌍 **Cross-Origin API Support (CORS) & Preflight Handling**: Built-in configurable CORS middleware and HTTP `OPTIONS` preflight handling (`204 No Content`) across all REST and OpenAPI endpoints, enabling seamless cross-origin API consumption from browser applications, third-party dashboards, and the bundled Swagger UI viewer.
- 🧹 **Execution Artifact Housekeeping & Retention Lifecycle Engine**: Automated background maintenance and retention enforcement for multi-tenant storage optimization and compliance. Configurable system-wide or per-suite time-to-live (TTL) policies govern independent retention windows for raw execution stdout/stderr logs (`runs/*/run.log`), deterministic summary reports (`runs/*/summary.json`), compiled scenario binaries, and historical database test run records. Features safe archiving (`RunStatusArchived` stripping raw summary JSON while preserving indexed KPI percentiles for longitudinal trend analysis), orphaned build artifact pruning, declarative synchronization of native S3 bucket lifecycle rules for automated storage-tier object expiration, safe dry-run simulation mode, and on-demand REST API triggers (`POST /api/v1/system/housekeeping` and `GET /api/v1/system/housekeeping/policy`).

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

> For details on database, storage parameters, and enabling the optional OpenAPI viewer (Swagger UI, including browser-accessible `specUrl` configuration for `kubectl port-forward`), see the [Infrastructure Helm Installation Guide (`deploy/helm/vuhive-cloud-infra/README.md`)](./deploy/helm/vuhive-cloud-infra/README.md).

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

# Inspect runtime version & compile-time metadata (injected via ldflags)
curl -i http://localhost:8080/version
# Response: {"version":"0.1.0","commit":"aca4153","build_time":"2026-09-06T12:00:00Z"}

# Fetch machine-readable OpenAPI 3.1 specification
curl -i http://localhost:8080/openapi.json
curl -i http://localhost:8080/openapi.yaml
```

To create your first runner profile, upload test suites, and trigger ad-hoc runs (`POST /api/v1/runs`), follow the **[Adoption Cookbook (`docs/cookbook.md`)](./docs/cookbook.md)**. For full REST API endpoint specifications, refer to the **[OpenAPI Reference (`api/openapi.yaml`)](./api/openapi.yaml)** or fetch it live at `/openapi.yaml` / `/openapi.json`.

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
