# vuhive-cloud

[![CI Tests](https://github.com/morphy76/vuhive-cloud/actions/workflows/test-main.yml/badge.svg)](https://github.com/morphy76/vuhive-cloud/actions/workflows/test-main.yml)
[![Lint](https://github.com/morphy76/vuhive-cloud/actions/workflows/lint.yml/badge.svg)](https://github.com/morphy76/vuhive-cloud/actions/workflows/lint.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg)](https://go.dev/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28%2B-326ce5.svg)](https://kubernetes.io/)

**`vuhive-cloud`** is an open-source, cloud-native control plane for orchestrating distributed load testing workloads on Kubernetes. It manages, distributes, and scales load testing scenarios implemented with the [`vuhive`](https://github.com/morphy76/vuhive) Go library (current release `v1.1.5`).

It transforms Go-based load testing suites into compiled, self-contained Linux binaries via ephemeral Kubernetes build jobs, executes them in security-hardened runner pods, schedules recurring executions with native Kubernetes `CronJob`s, synchronizes multi-pod workers via distributed rendezvous barriers, and indexes performance KPIs in real time.

---

## Key Features

- 🛠 **Ephemeral Source-to-Binary Compilation**: Upload raw Go source archives (`go.mod` + scenario code); `vuhive-cloud` dynamically spins up isolated Kubernetes build jobs (`golang:1.26-alpine`) to cross-compile static binaries targeting `linux/amd64` or `linux/arm64`.
- 🔒 **Hardened Execution Isolation**: Test runner pods comply with the Kubernetes **Restricted** Pod Security Standards (non-root UID `10001`, read-only root filesystems, all Linux capabilities dropped, zero host privilege).
- 🧩 **Reusable Runner Profiles**: Decouple test scenario code from infrastructure scheduling. Define reusable profiles specifying CPU/memory requests and limits, node selectors, tolerations, and node affinities for targeted execution.
- ⏰ **Native Kubernetes CronJob Scheduling**: Declarative scheduling mapped 1-to-1 to native Kubernetes `batch/v1` `CronJob`s with standard cron syntax (`0 2 * * *`), eliminating external scheduler dependencies.
- 📊 **Automated KPI Indexing & SLA Verification**: Automatically parses deterministic execution reports (`summary.json`), extracting and indexing latency percentiles ($p_{50}$, $p_{90}$, $p_{95}$, $p_{99}$), throughput (TPS), error rates, and SLA pass/fail status into PostgreSQL.
- 📦 **Pluggable Object Storage**: Integrates seamlessly with AWS S3 or MinIO for long-term retention of source packages, compiled binaries, full execution logs, and detailed performance summaries.
- ⏱ **Distributed Start Barrier Synchronization**: Built-in rendezvous coordinator guarantees multi-pod distributed load generators synchronize and fire simultaneously without clock skew.

---

## Architecture at a Glance

`vuhive-cloud` follows Hexagonal Architecture (Ports & Adapters) and Domain-Driven Design (DDD) principles:

```text
                                 ┌─────────────────────────────────┐
                                 │     API Client / CI/CD Pipeline │
                                 └────────────────┬────────────────┘
                                                  │ HTTP REST (Gin)
                                                  ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ vuhive-cloud Control Plane                                                             │
│                                                                                        │
│   ┌────────────────────────┐   ┌──────────────────────────┐   ┌────────────────────┐   │
│   │ Build Service          │   │ Profile & Run Service    │   │ Schedule Service   │   │
│   └───────────┬────────────┘   └────────────┬─────────────┘   └─────────┬──────────┘   │
│               │                             │                           │              │
│               │ (Ephemeral Builds)          │ (Runner Jobs)             │ (CronJobs)   │
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

## Documentation Navigation

| Document | Purpose & Audience |
|---|---|
| **[`README.md`](./README.md)** | System overview, core capabilities, architecture, and quickstart. |
| **[`docs/cookbook.md`](./docs/cookbook.md)** | **Adoption Guide & API Recipes**: Step-by-step `curl` walkthroughs for building suites, runner profiles, scheduling, triggering executions, ingesting KPIs, and barrier coordination. |
| **[`deploy/helm/vuhive-cloud/README.md`](./deploy/helm/vuhive-cloud/README.md)** | **Control Plane Helm Chart**: Production deployment guide, comprehensive configuration values reference, external secrets, RBAC, and security hardening. |
| **[`deploy/helm/vuhive-cloud-infra/README.md`](./deploy/helm/vuhive-cloud-infra/README.md)** | **Infrastructure Helm Chart**: Quickstart backing services setup for local evaluation (PostgreSQL + MinIO). |
| **[`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md)** | Detailed engineering specification, domain models, database DDL, and multi-milestone roadmap. |

---

## Quickstart

Get up and running locally on Rancher Desktop, Kind, or Minikube in three steps:

### 1. Deploy Backing Infrastructure (PostgreSQL + MinIO)

```bash
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

helm dependency build deploy/helm/vuhive-cloud-infra
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s
```

### 2. Deploy vuhive-cloud Control Plane

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set database.host=vuhive-infra-postgresql \
  --set s3.endpoint=http://vuhive-infra-minio:9000 \
  --wait --timeout=120s
```

### 3. Verify Health & Explore API Recipes

Port-forward the control plane service:

```bash
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080
```

Verify service liveness:

```bash
curl -i http://localhost:8080/healthz
```

To create your first runner profile, upload test suites, and trigger runs, follow the **[Adoption Cookbook (`docs/cookbook.md`)](./docs/cookbook.md)**.

---

## Repository Structure & Development Guide

```text
.
├── cmd/
│   ├── server/                 # Control plane REST server & migration entrypoint
│   ├── runner-init/            # Runner pod init container (downloads binary & config from S3)
│   └── runner-wrapper/         # Runner entrypoint wrapper (executes workload, captures KPIs)
├── internal/
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
# Build all local binaries
make build

# Run unit tests
make test

# Run tests with race detection
make test-race

# Run linter
make lint

# View all available targets
make help
```

---

## Project Tracking & Milestones

Development is tracked via GitHub Milestones and Issues:

- **[Milestone 1: Core Foundation & Single-Runner Cloud Engine](https://github.com/morphy76/vuhive-cloud/milestone/1)** (Completed)
  - Core domain models, PostgreSQL migrations & pgx repositories (#1, #2, #41)
  - S3/MinIO object storage adapter (#3)
  - Ephemeral Kubernetes build job generator & artifact API (#4, #5)
  - Runner profiles & init-container wrapper orchestration (#6, #7, #8)
  - Native Kubernetes CronJob scheduling (#9)
  - Performance KPI indexing & report ingestion (#20)
  - Helm packaging & CI/CD automation (#21, #27, #28, #53)
- **[Milestone 2: Distributed Multi-Pod Coordination & Live Streaming](https://github.com/morphy76/vuhive-cloud/milestone/2)** (In Progress)
  - Workload partitioning engine (#11)
  - Distributed start barrier rendezvous (#12)
  - Multi-report merging & HDR aggregation (#13)
  - Live runner telemetry streaming & web dashboard (#14, #15)
- **[Milestone 3: Multi-Namespace, Multi-Cluster & Enterprise SSO](https://github.com/morphy76/vuhive-cloud/milestone/3)** (Planned)
  - Dynamic runner namespaces & multi-cluster dispatching (#16, #17)
  - OIDC / OAuth2 authentication & team RBAC (#18, #19)

---

## License

This project is licensed under the [MIT License](./LICENSE).
