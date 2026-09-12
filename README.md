# vuhive-cloud Project Documentation & Setup Package

This repository contains the implementation and architecture documentation for **`vuhive-cloud`** (`github.com/morphy76/vuhive-cloud`) — a Kubernetes-native control plane for orchestrating distributed load testing suites and runner jobs.

Project roadmaps, epics, and implementation tasks are tracked directly via the [GitHub Issues Tracker](https://github.com/morphy76/vuhive-cloud/issues) and [GitHub Milestones](https://github.com/morphy76/vuhive-cloud/milestones).

---

## Key Features

### 1. Test Suite & Scenario Lifecycle Management
- 🧪 **Declarative Test Suites & Execution Profiles**: Full lifecycle CRUD management for test suites (`/api/v1/suites`) and attached scenario configurations (`vuhive.yaml`), supporting dynamic activation, versioning, and archiving via REST API without direct database intervention.
- 🧹 **Automated Retention & Housekeeping Engine**: Background maintenance enforcing system-wide or per-suite TTL policies across raw execution logs, deterministic summary reports, compiled binaries, and historical database records—including safe KPI-preserving archiving and native S3 lifecycle synchronization.

### 2. Ephemeral Compilation & Pre-Build AST Security
- 🛠 **Isolated Kubernetes Build Subsystem**: Ingests Go source archives (`go.mod` + scenario code) and dynamically spins up isolated, ephemeral Kubernetes compilation jobs (`golang:1.26-alpine`) to produce static, cross-compiled binaries targeting `linux/amd64` or `linux/arm64`.
- 🛡️ **Pre-Build AST Static Analysis & Contract Enforcement**: Uploaded archives undergo automated Go AST inspection (`go/parser` and `go/ast`) to verify direct `github.com/morphy76/vuhive` dependencies, enforce inverted control (`package scenario` implementing `vuhive.Scenario`), and block unauthorized system libraries (`os/exec`, `syscall`, `unsafe`, `runtime/cgo`).
- 🔒 **Platform-Managed Driver Injection**: Dynamically injects an immutable, platform-managed `main.go` execution driver, guaranteeing that runtime CLI flags (`--summary-export`, `--config`) and OS signal traps (`SIGINT`, `SIGTERM`) cannot be bypassed.

### 3. Kubernetes-Native Workload Orchestration & Scheduling
- 🚀 **Ad-Hoc Job Dispatching**: On-demand test execution (`POST /api/v1/runs`) creating a persistent `TestRun` in `QUEUED` state and manifesting an ephemeral `batch/v1` Job in the runner namespace with configurable deadlines (`activeDeadlineSeconds`).
- ⏰ **Native CronJob Scheduling & Correlation**: Declarative recurring test schedules mapped 1-to-1 to native Kubernetes `batch/v1` `CronJob`s (`0 2 * * *`). The informer watcher automatically correlates CronJob-spawned pods to `TestRun` records via `k8s_job_name` and real `k8s_namespace`.
- 🛑 **Execution Lifecycle Control & Graceful Abort**: Real-time run monitoring and on-demand abort (`POST /api/v1/runs/{id}/abort`) that instantly tears down Kubernetes workloads, propagates SIGTERM for partial log flushes, records audited cancellation metadata, and reclaims cluster resources.
- ⏱ **Distributed Start Barrier Synchronization**: Built-in rendezvous coordinator that synchronizes distributed multi-pod load generators to fire concurrently without clock skew.

### 4. Hardened Execution Isolation & Security Posture
- 🔒 **Restricted Pod Security Standards (PSS)**: Test runner pods strictly enforce non-root UID `10001`, read-only root filesystems with isolated `/tmp` and `/shared` `emptyDir` scratch spaces, `allowPrivilegeEscalation: false`, dropped Linux capabilities `["ALL"]`, zero host privileges, and `seccompProfile: RuntimeDefault`.
- 🌐 **Egress NetworkPolicy Isolation**: Enforces cluster-level network isolation denying cloud instance metadata endpoints (`169.254.169.254/32`) and internal Kubernetes API CIDRs, while permitting DNS, S3/MinIO, control plane telemetry callbacks, and designated target test CIDRs.
- 🧩 **Reusable Runner Profiles & RuntimeClasses**: Decouples scenario code from scheduling constraints with reusable profiles defining CPU/RAM requests and limits (Guaranteed QoS), node selectors, tolerations, node affinities, and kernel-isolated container sandboxes (e.g. gVisor `runsc`, Kata Containers).

### 5. Automated KPI Indexing, Analytics & Retention
- 📊 **Deterministic KPI Parsing & PostgreSQL Indexing**: Automatically digests execution reports (`summary.json`), indexing latency percentiles ($p_{50}$, $p_{90}$, $p_{95}$, $p_{99}$), throughput (TPS), error rates, and SLA pass/fail compliance directly into PostgreSQL for historical regression analysis.
- 📦 **Pluggable Object Storage Retention**: Integrates seamlessly with AWS S3 or MinIO for durable storage and presigned download URL generation for raw execution logs (`run.log`), full summary reports (`summary.json`), and compiled scenario binaries.
- 🔄 **Resilient Runner Telemetry & DNS Mitigation**: Injected runner wrapper reliably posts completion telemetry to `/api/v1/runs/complete` (resolving ad-hoc UUIDs or CronJob names) with built-in mitigation against Kubernetes `ndots:5` search domain lookup latency.

### 6. Developer Experience: Web Dashboard, Go BFF & CLI
- 🌐 **Modern React 19 PWA Web Dashboard**: Progressive Web App frontend (`web/`) with dark/light themes, offline query cache persistence via IndexedDB, and live status updates, packaged with zero overhead into the Go BFF via Go `embed.FS`.
- 📜 **Interactive Telemetry & Configuration Tooling**: Rich operator tools including virtualized ANSI log streaming (`run.log`), interactive tabular and collapsible JSON tree summary report inspectors (`summary.json`), YAML configuration editor with visual diffing, and visual runner profile builder.
- ⚡ **Go Backend-For-Frontend (cmd/bff) & Real-Time SSE Hub**: High-throughput gateway providing sub-50ms composite dashboard aggregation, persistent Server-Sent Events (SSE) telemetry fan-out (`run_status_changed`, `build_status_changed`), and transparent reverse proxying.
- 🛡️ **Confidential OAuth2 Token Handler & Enterprise OIDC**: Shields access/refresh tokens from the browser using secure `HttpOnly` session cookies, encrypted session persistence in PostgreSQL (AES-256-GCM), automatic token rotation, OIDC Backchannel Logout (RS256 JWKS verification), Keycloak RBAC personas (`vuhive-admin`, `vuhive-deployer`, `vuhive-developer`, `vuhive-viewer`, `vuhive-runner`), and cross-platform Developer CLI (`cmd/cli`).

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

The official Helm chart ([`deploy/helm/vuhive-cloud`](./deploy/helm/vuhive-cloud/README.md)) deploys both the core control plane server (`cmd/server`) and the Go BFF (`cmd/bff`, enabled by default) with unified Ingress routing. The Ingress automatically routes web dashboard traffic, BFF aggregations, and Keycloak OIDC Token Handler sessions (`/`, `/api/bff/v1`, `/api/v1/bff/auth`) to the BFF service, while routing core APIs (`/api/v1`) directly to the control plane server.

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
| **[`README.md`](./README.md)** | **Introduction & Overview** | **Introduces** `vuhive-cloud`: system capabilities, high-level architecture, core domain models, and quick evaluation guide. |
| **[`deploy/helm/vuhive-cloud/README.md`](./deploy/helm/vuhive-cloud/README.md)** | **Control Plane Installation & Deployment** | **Installs** the control plane: complete Helm parameters, production deployment, procedures for binding external infrastructure (PostgreSQL, S3/R2/Ceph), and Keycloak realm prerequisites (clients, roles, groups). |
| **[`deploy/helm/vuhive-cloud-infra/README.md`](./deploy/helm/vuhive-cloud-infra/README.md)** | **Infrastructure Installation (Evaluation)** | **Installs** backing evaluation services (PostgreSQL, MinIO, Keycloak IAM, and optional Swagger UI viewer). |
| **[`docs/cookbook.md`](./docs/cookbook.md)** | **Adoption Guide & Recipes** | **Drives adoption**: end-to-end recipes for authoring load tests with `vuhive`, packaging test suites, REST API recipes, CLI workflows, and diagnostics. |
| **[`api/openapi.yaml`](./api/openapi.yaml)** | **REST API Reference** | **Documents the APIs**: complete OpenAPI 3.1 specification for all endpoints, schemas, and OIDC bearer authentication contracts. |
| **[`CONTRIBUTING.md`](./CONTRIBUTING.md)** | **Developer & Contributor Guide** | Local environment setup, container builds with `--load`, local cluster validation, coding standards, and PR guidelines. |
| **[`AI_DISCLOSURE.md`](./AI_DISCLOSURE.md)** | **Engineering Philosophy & AI Disclosure** | Spec-Driven Development (SDD) paradigm, human vs. agent responsibility division, and quality gates. |
| **[`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md)** | **Architectural Specification** | Bounded contexts, DDD domain aggregates, database schema (DDL), and security postures. |

---

## Quickstart

Get up and running locally on Rancher Desktop, Kind, or Minikube in three steps:

### 1. Deploy Backing Infrastructure (PostgreSQL + MinIO + Keycloak)
Deploy backing services via the local infrastructure chart:

```bash
# 1. Add chart repositories
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

# 2. Deploy infrastructure (PostgreSQL + MinIO + Keycloak + optional Swagger UI)
helm dependency build deploy/helm/vuhive-cloud-infra
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s
```

> For details on database and storage parameters, Keycloak IAM database isolation, and enabling the optional OpenAPI viewer (Swagger UI, including browser-accessible `specUrl` configuration for `kubectl port-forward`), see the [Infrastructure Helm Installation Guide (`deploy/helm/vuhive-cloud-infra/README.md`)](./deploy/helm/vuhive-cloud-infra/README.md).

### 2. Deploy vuhive-cloud Control Plane
Deploy the control plane connected to the local infrastructure:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --wait --timeout=120s
```

> **MinIO & Local Development Note**: Setting `s3.endpoint` automatically enables path-style S3 addressing (`port 9000`). When deploying locally built images (`make docker-build` with `--load`), pass `-f deploy/helm/vuhive-cloud/values-dev.yaml`. See the [Control Plane Helm Installation Guide (`deploy/helm/vuhive-cloud/README.md`)](./deploy/helm/vuhive-cloud/README.md) for full configuration reference and production deployment options.

### 3. Verify Health, Version & OpenAPI Endpoints
 
Port-forward the control plane service:

```bash
# Port-forward the control plane REST API (port 8080)
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080

# In a separate terminal, port-forward the Web UI dashboard (port 8081)
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud-bff 8081:8081
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

## Production Deployments & External Infrastructure Binding

While the [Quickstart](#quickstart) uses the bundled evaluation chart (`deploy/helm/vuhive-cloud-infra`) to provision ephemeral, single-pod instances of PostgreSQL, MinIO, and Keycloak for local testing, **production deployments must bind to dedicated external infrastructure** provisioned via managed cloud providers, enterprise Kubernetes operators, or existing corporate clusters.

> [!IMPORTANT]
> **Documentation Boundary**: Operational procedures for binding external third-party dependencies are **not** a topic of this main project README, but are comprehensively documented in the **[Control Plane Helm Chart Guide (`deploy/helm/vuhive-cloud/README.md`)](./deploy/helm/vuhive-cloud/README.md#5-external-infrastructure--production-deployment-scenarios)**:
> - **PostgreSQL Binding**: Managed cloud databases (AWS RDS / Aurora, GCP Cloud SQL) and Kubernetes operators (CloudNativePG, Crunchy Data PGO), credential management via `database.existingSecret`, SSL enforcement (`sslmode=require`), and automated pre-install/pre-upgrade schema migrations (`database.autoMigrate`).
> - **S3 & Object Storage**: Native AWS S3 with IAM Roles for Service Accounts (IRSA / EKS Pod Identity) vs. third-party S3-compatible providers (Cloudflare R2, Ceph, MinIO) with path-style addressing (`s3.usePathStyle=true`).
> - **External IAM & Keycloak Realm Prerequisites**: Detailed specifications for pre-configuring the target Keycloak realm, including all four required clients (`vuhive-cloud-api`, `vuhive-cloud-cli`, `vuhive-runner`, `vuhive-cloud-bff`), RBAC realm roles (`vuhive-admin`, `vuhive-deployer`, `vuhive-developer`, `vuhive-viewer`, `vuhive-runner`), group mappings, token claim mappers, and the starter realm template [`deploy/helm/vuhive-cloud-infra/files/vuhive-realm.json`](./deploy/helm/vuhive-cloud-infra/files/vuhive-realm.json).
> - **Ready-to-Use Production Template**: [`deploy/helm/vuhive-cloud/values-production.yaml`](./deploy/helm/vuhive-cloud/values-production.yaml).

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
make docker-build-bff
make docker-build-runner-init

# Prune Docker build cache to prevent Kubelet ImageGC eviction on local clusters
make docker-prune


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

# Run BFF with persistent PostgreSQL sessions and automatic startup migrations
./bin/bff --port=8081 --control-plane-url=http://localhost:8080 \
  --database-url="postgres://vuhive:vuhive-dev@localhost:5432/vuhive?sslmode=disable" \
  --session-encryption-key="your-32-byte-aes256-secret-key"

# Run BFF with Keycloak OIDC Token Handler authentication enabled
./bin/bff --port=8081 --control-plane-url=http://localhost:8080 \
  --database-url="postgres://vuhive:vuhive-dev@localhost:5432/vuhive?sslmode=disable" \
  --keycloak-issuer-url="http://localhost:8082/realms/vuhive" \
  --keycloak-client-id="vuhive-cloud-bff" \
  --keycloak-client-secret="vuhive-bff-secret"

# Run BFF during frontend development with Vite HMR reverse proxying
./bin/bff --port=8081 --control-plane-url=http://localhost:8080 --dev-proxy-url=http://localhost:5173

# Run BFF with a local static build directory override
./bin/bff --port=8081 --control-plane-url=http://localhost:8080 --static-dir=./web/dist
```

---

## Authentication, Authorization & Developer CLI

`vuhive-cloud` integrates enterprise-grade OpenID Connect (OIDC) authentication and Role-Based Access Control (RBAC) backed by Keycloak.

### Role-Based Access Control (RBAC) Matrix

| Persona Role | Keycloak Group | Permitted Operations |
|---|---|---|
| **`vuhive-admin`** | `/administrators` | Full administrative control: manage all resources, configurations, and housekeeping policies. |
| **`vuhive-deployer`** | `/deployers` | Workload execution: dispatch ad-hoc runs (`POST /api/v1/runs`), abort runs (`POST /api/v1/runs/{id}/abort`), create and manage CronJob schedules (`POST /api/v1/schedules`), inspect all entities. |
| **`vuhive-developer`** | `/developers` | Authoring & Compilation: register test suites (`POST /api/v1/suites`), upload Go scenario source packages (`POST /api/v1/suites/{id}/builds`), inspect builds, view runs and reports. |
| **`vuhive-viewer`** | `/viewers` | Read-only inspection: list and view test suites, runner profiles, schedules, runs, logs, and summary reports. |
| **`vuhive-runner`** | M2M Service Account | Runner execution: post start barrier sync signals (`POST /api/v1/barrier/rendezvous`), abort barrier sessions, report run completion (`POST /api/v1/runs/complete`). |

### Developer CLI (`vuhive`)

The `vuhive` CLI (`cmd/cli`, compiled via `make build-cli` into `bin/vuhive`) provides a first-class command-line interface for scenario developers and release engineers.

```bash
# Build the developer CLI
make build-cli

# Authenticate via browser PKCE flow (with Device Authorization fallback)
./bin/vuhive auth login --issuer http://localhost:8080/realms/vuhive --server http://localhost:8080

# Inspect active credentials, token expiration, and granted roles
./bin/vuhive auth status

# Upload scenario source package and trigger automated compilation
./bin/vuhive suite upload <source-path> --suite-id <suite-uuid>

# Dispatch an ad-hoc load test run
./bin/vuhive run start <suite-uuid> --profile-id <profile-uuid> --artifact-id <artifact-uuid>

# Inspect execution status, KPIs, stream logs, or fetch report
./bin/vuhive run status <run-uuid>
./bin/vuhive run logs <run-uuid>
./bin/vuhive run report <run-uuid>

# Manage native Kubernetes CronJob schedules
./bin/vuhive schedule create --name nightly-test --suite-id <suite-uuid> --profile-id <profile-uuid> --artifact-id <artifact-uuid> --cron "0 2 * * *"
./bin/vuhive schedule list

# Log out and revoke credentials
./bin/vuhive auth logout
```

---

The control plane exposes its REST API at port `8080`. See [`api/openapi.yaml`](./api/openapi.yaml) for the full API reference.
For detailed Helm configuration options, see the chart READMEs:
- [`deploy/helm/vuhive-cloud-infra/README.md`](./deploy/helm/vuhive-cloud-infra/README.md) — infrastructure (PostgreSQL + MinIO + Keycloak + optional Swagger UI viewer)
- [`deploy/helm/vuhive-cloud/README.md`](./deploy/helm/vuhive-cloud/README.md) — control plane (namespace management, RBAC modes, OIDC authentication, all parameters)

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
