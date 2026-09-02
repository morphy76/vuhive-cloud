# vuhive-cloud: Roadmap & GitHub Issues Breakdown

This document provides a structured, issue-ready breakdown for `github.com/morphy76/vuhive-cloud`. Each issue includes a title, labels, user story / objective, key requirements, and acceptance criteria to enable direct import into GitHub Issues and GitHub Projects.

---

## Milestone 1: Core Foundation & Single-Runner Cloud Engine

**Target Goal:** End-to-end functionality for uploading Go source test suites, compiling static binaries via ephemeral K8s Jobs into S3, managing reusable Runner Profiles, dispatching single-runner K8s Jobs and native CronJobs with node affinities/tolerations, and indexing reports in PostgreSQL.

---

### Epic 1.1: Hexagonal Architecture, Domain Models & Data Layer

#### Issue 1.1.1: Project Scaffolding & Hexagonal Domain Models
- **Labels:** `area/domain`, `milestone-1`, `epic/foundation`
- **Objective:** Establish the Hexagonal / DDD package structure and define pure domain models without external framework dependencies.
- **Tasks:**
  - Initialize directory structure: `internal/domain/{model,event}`, `internal/application/{ports,service}`, `internal/adapters/{inbound,outbound}`.
  - Define `TestSuite` aggregate with lifecycle states.
  - Define `Artifact` entity and `Platform` value object (`linux/amd64`, `linux/arm64`).
  - Define `RunnerProfile` value object (tolerations, affinity, CPU/RAM).
  - Define `TestRun` aggregate with status state machine (`QUEUED`, `RUNNING`, `COMPLETED`, `FAILED`, `ABORTED`).
  - Define `Schedule` aggregate with Cron expression validation.
  - Add compile-time interface verification assertions across domain boundaries.
- **Acceptance Criteria:**
  - Pure Go domain models with 100% unit test coverage on state transitions.
  - Zero imports of Gin, pgx, or client-go in `internal/domain`.

#### Issue 1.1.2: PostgreSQL Database Schema Migrations & pgx Repositories
- **Labels:** `area/database`, `milestone-1`, `epic/foundation`
- **Objective:** Implement database schema migrations and outbound repository adapters using `pgxpool`.
- **Tasks:**
  - Create Golang migrate scripts for `test_suites`, `artifacts`, `configurations`, `runner_profiles`, `schedules`, and `test_runs` tables with indexes.
  - Implement repository outbound ports for CRUD operations.
  - Add domain error translation (e.g. `pgx.ErrNoRows` -> `domain.ErrNotFound`).
- **Acceptance Criteria:**
  - Migrations run idempotently up and down.
  - Integration tests against PostgreSQL via `testcontainers-go` pass.

#### Issue 1.1.3: S3 / MinIO Object Storage Adapter
- **Labels:** `area/storage`, `milestone-1`, `epic/foundation`
- **Objective:** Implement outbound `StoragePort` using AWS SDK v2 for S3 / MinIO compatibility.
- **Tasks:**
  - Implement methods for storing/retrieving source tarballs, static binaries, configuration YAMLs, execution logs, and `summary.json`.
  - Add bucket initialization and presigned URL generation (optional).
- **Acceptance Criteria:**
  - Storage adapter satisfies `StoragePort` interface.
  - Integration tests with MinIO via `testcontainers-go` pass.

---

### Epic 1.2: Source-to-Binary Compilation Subsystem

#### Issue 1.2.1: Ephemeral Kubernetes Build Job Generator
- **Labels:** `area/build`, `milestone-1`, `epic/compilation`
- **Objective:** Build the orchestrator that generates and dispatches isolated K8s compilation Jobs.
- **Tasks:**
  - Construct `batch/v1` `Job` manifest using image `golang:1.26-alpine`.
  - Configure compilation commands (`CGO_ENABLED=0 GOOS=linux GOARCH=<target> go build -trimpath -ldflags="-s -w"`).
  - Stream build stdout/stderr to S3 and update artifact status in DB.
- **Acceptance Criteria:**
  - Source tarball is successfully compiled into a static binary targeting both `amd64` and `arm64`.
  - Binary is pushed to S3 and tagged as `READY` in `artifacts` table.

#### Issue 1.2.2: Artifact Registry Inbound REST API
- **Labels:** `area/api`, `milestone-1`, `epic/compilation`
- **Objective:** Expose REST endpoints to upload Go test source code and list compiled binaries.
- **Tasks:**
  - `POST /api/v1/suites/{id}/builds` (multipart form upload for tar.gz + target arch).
  - `GET /api/v1/suites/{id}/artifacts` (list available binaries with checksums).
- **Acceptance Criteria:**
  - Source archives are accepted, staged in S3, and trigger build Jobs asynchronously.

---

### Epic 1.3: Runner Pod Orchestration & Profiles

#### Issue 1.3.1: Reusable Runner Profiles Management
- **Labels:** `area/orchestration`, `milestone-1`, `epic/runners`
- **Objective:** Provide CRUD API and domain logic for reusable Runner Profiles.
- **Tasks:**
  - Define profile structure (runner image, CPU/RAM requests/limits, nodeSelector, nodeAffinity, tolerations).
  - Expose `POST /api/v1/profiles` and `GET /api/v1/profiles`.
- **Acceptance Criteria:**
  - Profiles can be created, updated, and validated against Kubernetes resource schemas.

#### Issue 1.3.2: Runner Pod Init Container & Injected Entrypoint
- **Labels:** `area/runner`, `milestone-1`, `epic/runners`
- **Objective:** Develop the init container and runner wrapper that manages test execution inside K8s pods.
- **Tasks:**
  - Build `runner-init` image: downloads the target binary and `vuhive.yaml` from S3 into an `emptyDir` volume (`/shared`).
  - Create entrypoint script/wrapper: executes `/shared/runner --summary-export=/shared/summary.json`, traps exit signals, writes `/shared/run.log`, and uploads artifacts to S3 upon termination.
- **Acceptance Criteria:**
  - Pod initializes seamlessly, runs test binary, and guarantees report upload even on non-zero exit codes.

#### Issue 1.3.3: Ad-Hoc K8s Job Dispatcher & Informer Watcher
- **Labels:** `area/orchestration`, `milestone-1`, `epic/runners`
- **Objective:** Implement use case service that creates K8s Jobs and watches pod lifecycle.
- **Tasks:**
  - Implement `RunService.TriggerRun(ctx, cmd)` that manifests the K8s Job applying profile affinities and tolerations.
  - Implement client-go Informer to watch Job status changes (`Running`, `Succeeded`, `Failed`) and update `test_runs` table.
- **Acceptance Criteria:**
  - Triggering a run launches a K8s Job; state transitions from `QUEUED` -> `RUNNING` -> `COMPLETED`/`FAILED`.

---

### Epic 1.4: Scheduling, Reporting & CLI

#### Issue 1.4.1: Native Kubernetes CronJob Manager
- **Labels:** `area/scheduling`, `milestone-1`, `epic/scheduling`
- **Objective:** Manage native `batch/v1` `CronJob` resources in Kubernetes.
- **Tasks:**
  - Implement `ScheduleService` creating, updating, and deleting K8s CronJobs matching user cron expressions.
  - Implement Job watcher that detects CronJob-spawned pods and records them as linked `TestRun` records in PostgreSQL.
- **Acceptance Criteria:**
  - Deleting a schedule cleans up the underlying K8s CronJob.
  - Scheduled firings automatically create tracked `TestRun` entries.

#### Issue 1.4.2: Report Ingestion & Performance KPI Indexing
- **Labels:** `area/reporting`, `milestone-1`, `epic/reporting`
- **Objective:** Ingest deterministic `vuhive` summary reports and extract metrics.
- **Tasks:**
  - Implement endpoint `POST /api/v1/runs/{id}/complete`.
  - Parse `summary.json` to extract SLA pass/fail, throughput TPS, error rate %, and latency percentiles (`p50`, `p90`, `p95`, `p99`).
  - Update `test_runs` record with indexed values and S3 keys.
- **Acceptance Criteria:**
  - Historical query endpoints return parsed performance metrics without re-reading S3 JSON files.

#### Issue 1.4.3: REST API Security & vuhive-cloud CLI
- **Labels:** `area/api`, `area/cli`, `milestone-1`, `epic/cli`
- **Objective:** Secure the API with Bearer/API Key tokens and provide a developer CLI.
- **Tasks:**
  - Implement authentication middleware for Gin.
  - Build `vuhive-cloud` CLI (`upload`, `run`, `status`, `logs`, `report`).
- **Acceptance Criteria:**
  - Developers can trigger and monitor tests directly from their local terminal.

---

### Epic 1.5: Helm Deployment & Infrastructure Packaging

#### Issue 1.5.1: Helm Chart for vuhive-cloud Control Plane & Infrastructure
- **Labels:** `area/deploy`, `milestone-1`, `epic/deployment`
- **Objective:** Provide an official Helm chart (`deploy/helm/vuhive-cloud`) to package and deploy the `vuhive-cloud` control plane, RBAC policies, service accounts, and backing services.
- **Tasks:**
  - Create Helm chart structure under `deploy/helm/vuhive-cloud` (`Chart.yaml`, `values.yaml`, templates).
  - Define templates for `vuhive-cloud` Deployment, Service, Ingress, and ServiceAccount.
  - Define RBAC templates (`ClusterRole`/`Role` and bindings) granting control plane permissions to manage `batch/v1` `Job`s, `CronJob`s, pods, and logs in runner namespaces.
  - Add configuration for connecting to PostgreSQL and S3/MinIO (support external instances and optional subchart dependencies for quickstart/dev).
  - Provide customizable values for resource requests/limits, environment variables, tolerations, and node selectors for the control plane.
- **Acceptance Criteria:**
  - `helm lint` and `helm template` render clean, valid Kubernetes manifests without errors.
  - A clean deployment on a Kubernetes cluster brings up the control plane and allows it to orchestrate runner jobs.

---

## Milestone 2: Distributed Multi-Pod Coordination & Live Streaming

**Target Goal:** Scale load generation across multiple synchronized worker pods and stream live telemetry during test execution.

---

### Epic 2.1: Distributed Multi-Pod Coordination
- **Issue 2.1.1: Workload Partitioning Engine**
  - Divide VU target concurrency and arrival rates evenly across $N$ worker pods in `vuhive.yaml` overlays.
- **Issue 2.1.2: Distributed Start Barrier (Rendezvous)**
  - Implement synchronized pod startup coordination via Redis lock or K8s coordinator leader to guarantee all pods generate load simultaneously.
- **Issue 2.1.3: Multi-Report Merging & HDR Aggregation**
  - Combine multiple `summary.json` files and merge sharded HDR histograms into a unified suite report.

---

### Epic 2.2: Live Telemetry Streaming
- **Issue 2.2.1: Runner Live Telemetry Emitter**
  - Add periodic gRPC / WebSocket streaming adapter to `vuhive` runner to push 1-second metric snapshots.
- **Issue 2.2.2: Live Monitoring Web Dashboard**
  - Provide real-time UI showing live TPS, latency percentiles, active VUs, and streaming logs.

---

## Milestone 3: Multi-Namespace, Multi-Cluster & Enterprise SSO

**Target Goal:** Enable multi-tenant enterprise operation across isolated namespaces, multiple cloud Kubernetes clusters, and corporate identity providers.

---

### Epic 3.1: Multi-Namespace & Multi-Cluster Dispatcher
- **Issue 3.1.1: Dynamic Runner Namespaces**
  - Allow test suites to target user-selected namespaces with auto-provisioned ServiceAccounts and RBAC roles.
- **Issue 3.1.2: Remote Multi-Cluster Kubeconfig Orchestration**
  - Connect external EKS, GKE, and on-prem K8s clusters to dispatch runners in geographical regions closest to target services.

---

### Epic 3.2: Enterprise Authentication & RBAC
- **Issue 3.2.1: OIDC / OAuth2 Identity Integration**
  - Plug in OIDC SSO for GitHub, Keycloak, and Google Identity.
- **Issue 3.2.2: Team Spaces & RBAC Permissions**
  - Enforce role-based access control (Admin, Tester, Viewer) scoped to test suites and runner profiles.
