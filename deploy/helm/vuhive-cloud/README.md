# vuhive-cloud

Official Helm chart for the `vuhive-cloud` control plane.

## Overview

`vuhive-cloud` provides the Kubernetes-native control plane for orchestrating distributed load testing suites and runner jobs.

> **Documentation Navigation**:
> - **System Architecture & Overview**: [`README.md`](../../README.md) and [`ARCHITECTURE_SPEC.md`](../../ARCHITECTURE_SPEC.md)
> - **Developer & Contributor Guide**: [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
> - **Infrastructure Chart (PostgreSQL + MinIO + OpenAPI Viewer)**: [`deploy/helm/vuhive-cloud-infra/README.md`](../vuhive-cloud-infra/README.md)
> - **Adoption Guide & API Recipes**: [`docs/cookbook.md`](../../docs/cookbook.md)
> - **REST API Reference**: [OpenAPI 3.1 Specification (`api/openapi.yaml`)](../../api/openapi.yaml) (served live at `GET /openapi.yaml` and `GET /openapi.json`)
> - **Engineering Philosophy**: [`AI_DISCLOSURE.md`](../../AI_DISCLOSURE.md)

### Architecture Components

- **Control Plane (`cmd/server`)**: Core engine orchestrating ephemeral compilation jobs, runner profiles, `batch/v1` Jobs, native CronJobs, and KPI indexing.
- **Backend-For-Frontend (`cmd/bff`)**: High-throughput composite gateway serving the embedded React 19 SPA web dashboard and PWA assets directly via Go `embed.FS`, managing client sessions, providing sub-50ms parallel aggregation (`GET /api/bff/v1/dashboard`), unified run detail endpoints with presigned S3 links (`GET /api/bff/v1/runs/{id}`), and transparent reverse proxying for entity CRUD operations (`/api/bff/v1/suites`, `/api/bff/v1/profiles`, `/api/bff/v1/schedules`, `/api/bff/v1/runs`) to the upstream control plane. The embedded frontend includes full offline-capable domain concept micro-guidance and accessible help tooltips (`<HelpTooltip />`, `<InfoBadge />`) with zero external CDN dependencies.

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+ / Helm 4+
- Backing services:
  - PostgreSQL database (can be deployed via `vuhive-cloud-infra`)
  - MinIO or AWS S3 compatible object storage (can be deployed via `vuhive-cloud-infra`)

## Quickstart

### 1. Deploy Infrastructure

```bash
# Add dependencies repositories
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

# Install infrastructure backing services
helm dependency build deploy/helm/vuhive-cloud-infra
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace
```

> [!TIP]
> If deploying the optional OpenAPI viewer (Swagger UI) in `vuhive-cloud-infra`, refer to the [Infrastructure Helm Guide](../vuhive-cloud-infra/README.md#enabling-the-openapi-viewer-swagger-ui) for configuring browser-accessible `specUrl` when using `kubectl port-forward`.

### 2. Deploy vuhive-cloud Control Plane

With default values, Helm automatically creates the `vuhive-runners` and `vuhive-system` namespaces
(see [Namespace Management](#namespace-management) below):

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set database.host=vuhive-infra-postgresql \
  --set s3.endpoint=http://vuhive-infra-minio:9000
```

#### Enabling OIDC Authentication & RBAC (with Keycloak)

To secure the control plane REST API with Keycloak OpenID Connect and Role-Based Access Control:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set database.host=vuhive-infra-postgresql \
  --set s3.endpoint=http://vuhive-infra-minio:9000 \
  --set auth.enabled=true \
  --set auth.issuerUrl=http://vuhive-infra-vuhive-cloud-infra-keycloak:8080/realms/vuhive \
  --set auth.jwksUrl=http://vuhive-infra-vuhive-cloud-infra-keycloak:8080/realms/vuhive/protocol/openid-connect/certs \
  --set auth.runner.clientId=vuhive-runner \
  --set auth.runner.clientSecret=vuhive-runner-secret
```

> [!NOTE]
> When `s3.endpoint` is non-empty (as in the MinIO case above), `s3.usePathStyle` is automatically treated as `true` by the control plane server, runner-init, and runner-wrapper. Path-style addressing (`http://<endpoint>/<bucket>/`) is required for MinIO because virtual-hosted-style URLs (`http://<bucket>.<service>/`) depend on DNS wildcards unavailable for Kubernetes Service names.

### 3. Verify Health & Version Metadata

Verify that the control plane deployment is healthy and inspect runtime compile-time metadata (`version`, `commit`, `build_time`):

```bash
# Port-forward the control plane service to localhost
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080

# Check service health
curl -i http://localhost:8080/healthz

# Inspect runtime version and build metadata
curl -i http://localhost:8080/version
```

Example `/version` response:
```json
{
  "version": "0.1.0",
  "commit": "aca4153",
  "build_time": "2026-09-06T12:00:00Z"
}
```

### 4. Local Cluster Testing with Locally Built Images

When testing code changes against a local cluster (e.g. Rancher Desktop, Kind, Minikube), build container images using `make docker-build` (which applies `--load --provenance=false` so images are loaded directly into the local CRI store):

```bash
# In repository root:
make docker-build

# Deploy with local image overrides:
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set image.repository=vuhive/server \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent \
  --set runner.initImage=vuhive/runner-init:local
```

For full details on local container building, BuildKit cache pruning (`make docker-prune`), and troubleshooting `ImagePullBackOff` / Kubelet ImageGC eviction issues, see [`CONTRIBUTING.md`](../../CONTRIBUTING.md).

### 5. Production Deployment (with External PostgreSQL, S3 & OIDC)

In production environments, backing services should be provisioned via managed cloud infrastructure (e.g. AWS Aurora / RDS, GCP Cloud SQL, CloudNativePG), dedicated object storage (AWS S3, Cloudflare R2, Ceph), and enterprise Identity Providers (production Keycloak, Okta, Auth0, Microsoft Entra ID).

> [!TIP]
> For high-level architectural comparison and provider options (AWS RDS, CloudNativePG, Cloudflare R2, Ceph, Okta, Keycloak), see the [External Infrastructure & Third-Party Deployment Scenarios section in root README.md](../../README.md#external-infrastructure--third-party-deployment-scenarios).

#### Step 1: Create Kubernetes Secrets for External Credentials

Do not commit plaintext passwords or access keys into values files or version control. Pre-create Kubernetes Secrets in the release namespace (`vuhive-system`):

```bash
# 1. PostgreSQL Database URL Secret
kubectl create secret generic vuhive-db-secret \
  --namespace vuhive-system \
  --from-literal=DATABASE_URL="postgres://vuhive_user:SecurePassword123@postgres.production.internal:5432/vuhive?sslmode=require"

# 2. S3 Object Storage Credentials Secret (omit if using IRSA / Workload Identity)
kubectl create secret generic vuhive-s3-secret \
  --namespace vuhive-system \
  --from-literal=AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE" \
  --from-literal=AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# 3. Runner M2M OAuth2 Client Secret (when auth.enabled=true)
kubectl create secret generic vuhive-runner-auth \
  --namespace vuhive-system \
  --from-literal=RUNNER_CLIENT_SECRET="SecretRunnerKey987"
```

#### Step 2: Configure Cloud IAM / IRSA (Optional for AWS S3)

When running on Amazon EKS, authenticate against AWS S3 without static access keys by using **IAM Roles for Service Accounts (IRSA)** or **EKS Pod Identity**:

1. Create an AWS IAM Policy granting read/write access to your S3 bucket:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:ListBucket"],
         "Resource": [
           "arn:aws:s3:::vuhive-production-artifacts",
           "arn:aws:s3:::vuhive-production-artifacts/*"
         ]
       }
     ]
   }
   ```
2. Annotate the control plane ServiceAccount in Helm values:
   ```yaml
   serviceAccount:
     create: true
     annotations:
       eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/vuhive-control-plane-s3-role"
   ```

#### Step 3: Configure Production Values (`values-production.yaml`)

Use the template provided at [`values-production.yaml`](./values-production.yaml) or customize your values:

```yaml
replicaCount: 2

# PostgreSQL Database Binding
database:
  host: "postgres.production.internal"
  port: 5432
  name: "vuhive"
  user: "vuhive_user"
  sslmode: "require"
  existingSecret: "vuhive-db-secret"
  existingSecretKey: "DATABASE_URL"
  autoMigrate: true

# S3 Object Storage Binding
s3:
  # Leave empty for AWS S3 regional endpoints; set URL for Cloudflare R2 / Ceph / MinIO
  endpoint: ""
  region: "eu-west-1"
  bucket: "vuhive-production-artifacts"
  # Set false for AWS S3 virtual-hosted style; set true for Cloudflare R2, Ceph, MinIO
  usePathStyle: false
  existingSecret: "vuhive-s3-secret"
  existingSecretAccessKey: "AWS_ACCESS_KEY_ID"
  existingSecretSecretKey: "AWS_SECRET_ACCESS_KEY"

# Identity & Access Management (OIDC)
auth:
  enabled: true
  issuerUrl: "https://auth.production.internal/realms/vuhive"
  jwksUrl: "https://auth.production.internal/realms/vuhive/protocol/openid-connect/certs"
  runner:
    clientId: "vuhive-runner"
    tokenUrl: "https://auth.production.internal/realms/vuhive/protocol/openid-connect/token"
    existingSecret: "vuhive-runner-auth"
    existingSecretKey: "RUNNER_CLIENT_SECRET"

# Production Sizing & High Availability
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: vuhive-cloud
          topologyKey: kubernetes.io/hostname
```

#### Step 4: Install or Upgrade with Helm

Execute the Helm installation:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --create-namespace \
  -f deploy/helm/vuhive-cloud/values-production.yaml \
  --wait --timeout=180s
```

During deployment:
1. **Automated Schema Migrations (`database.autoMigrate`)**: Helm runs the `vuhive-cloud-migration` pre-install/pre-upgrade hook Job, applying Goose database schema migrations before updating the Deployment pods.
2. **Readiness Probing**: The deployment awaits database connectivity and S3 storage bucket verification before marking pods ready.

#### Step 5: Post-Install Verification

Verify deployment health and migration job completion:

```bash
# Check migration job completion status
kubectl get jobs -n vuhive-system -l app.kubernetes.io/component=migration

# Check control plane pods
kubectl get pods -n vuhive-system -l app.kubernetes.io/name=vuhive-cloud

# Verify service liveness
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080 &
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/version
```



## Namespace Management

When `rbac.clusterScoped` is `false` (the default), the chart creates scoped `Role` and `RoleBinding`
objects in both the runner namespace and the builder namespace. Kubernetes requires these namespaces to
exist before the resources are applied.

### Automatic Namespace Creation (default)

By default, `runner.createNamespace: true` and `builder.createNamespace: true` instruct Helm to create
those namespaces automatically. The created namespaces carry the annotation:

```yaml
annotations:
  "helm.sh/resource-policy": keep
```

This means the namespaces are **intentionally preserved** on `helm uninstall` to protect any live runner
pods or build jobs that may still be running.

### Manual Pre-Creation

If you prefer to manage namespaces outside of Helm (e.g., via GitOps or a cluster bootstrap process),
set `createNamespace: false` and create the namespaces manually before installing:

```bash
kubectl create namespace vuhive-runners
kubectl create namespace vuhive-system

helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set runner.createNamespace=false \
  --set builder.createNamespace=false
```

### Cluster-Scoped RBAC

To avoid namespace management entirely, enable cluster-scoped RBAC (single `ClusterRole` /
`ClusterRoleBinding`). This is suitable for single-tenant clusters:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set rbac.clusterScoped=true
```

### Deploying Runner and Builder in the Same Namespace

For development or minimal setups, point runner and builder to the same namespace as the control plane:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set runner.namespace=vuhive-system \
  --set builder.namespace=vuhive-system
```

In this case no extra namespaces are created and no cross-namespace RBAC is needed. Furthermore, `apiCallbackUrl` automatically resolves to the unqualified service name `http://<fullname>:<port>/api/v1/runs/complete`.

### Runner Completion Callbacks and DNS Search Domain Mitigation

Runner containers upload execution artifacts (`run.log` and `summary.json`) to S3 and post completion telemetry to the control plane callback endpoint (`POST /api/v1/runs/complete`).

To ensure callback requests succeed across different Kubernetes network setups:
- **Same Namespace (`runner.namespace == Release.Namespace`)**: The chart automatically configures `apiCallbackUrl` as `http://<fullname>:<port>/api/v1/runs/complete`. With 0 dots, standard Kubernetes pods resolve the service name directly via the local namespace search domain without traversing upstream search lists.
- **Cross-Namespace (`runner.namespace != Release.Namespace`)**: Standard Kubernetes pods have `ndots:5` in `/etc/resolv.conf`. Because `<service>.<namespace>.svc.cluster.local` has 4 dots, standard resolvers query host/DHCP upstream search domains first, which can cause connection failures if upstream wildcard DNS returns `127.0.0.1`. The chart mitigates this by generating a fully qualified domain name with a **trailing dot** (`http://<fullname>.<namespace>.svc.cluster.local.:<port>/api/v1/runs/complete`), bypassing search lists and directing the query straight to CoreDNS.
- **Custom Override**: You can override `apiCallbackUrl` explicitly with `--set apiCallbackUrl=...` if you route runner callbacks through custom gateways or ingresses.

### Ad-Hoc Test Run Dispatching

In addition to scheduled runs, the control plane allows developers and CI/CD pipelines to dispatch ad-hoc test executions on demand via `POST /api/v1/runs`. The control plane manifests an ephemeral `batch/v1` `Job` directly in `runner.namespace` adhering to the specified `RunnerProfile` and `TestSuite` artifact. Runner pods execute within the target namespace under the Restricted Pod Security Standard and invoke the callback URL upon completion.

### CronJob Run Correlation

When a `CronJob` fires a `batch/v1` Job:

1. The runner pod's `VUHIVE_RUN_ID` is populated from `metadata.labels['batch.kubernetes.io/job-name']` (the **Job name**, e.g. `vuhive-sched-abc-28492020`), not from `metadata.name` (the pod name).
2. The `RunnerJobWatcher` informer auto-creates a `TestRun` record linked to the schedule, and records both `k8s_job_name` and the **actual `k8s_namespace`** from `runner.namespace` (not a hard-coded default).
3. When the runner-wrapper POSTs the completion callback with `run_id = <job-name>`, the control plane first attempts UUID lookup, then falls back to `k8s_job_name` correlation, ensuring the `TestRun` is correctly finalized with summary KPIs regardless of whether the run was dispatched ad-hoc or via a CronJob.

### Pre-Build AST Static Analysis & Framework Enforcement

The control plane implements an automated pre-build static verification gate for all uploaded test archives (`POST /api/v1/suites/{id}/builds`):
- **`go.mod` Verification**: Validates that `go.mod` declares `github.com/morphy76/vuhive` as a required direct dependency.
- **Inverted Control (`package scenario`)**: Uploaded Go files must belong to `package scenario` (defining `NewScenario()`, `Scenario()`, `InitScenario()`, or `Register(*vuhive.Engine)`). Defining `package main` or `func main()` is prohibited. The platform automatically injects an immutable `main.go` driver into the compilation workspace.
- **Import Blocklist**: Prohibits dangerous libraries (`os/exec`, `syscall`, `unsafe`, `plugin`, `runtime/cgo`, `golang.org/x/sys`) by default.
- **Cluster Deployment Overrides**:
  - `ALLOW_INSECURE_IMPORTS` (`true`/`false`): Controls whether users can request an import blocklist override (`allow_insecure_imports=true`). When enabled, overridden artifacts are flagged as dangerous.
  - `ALLOWED_IMPORT_PACKAGES` (comma-separated): Configures cluster-wide package exemptions.

### Backend-For-Frontend (BFF) Gateway & Dashboard Routing

The Backend-For-Frontend service (`cmd/bff`) acts as the presentation gateway and SPA host for the control plane:

- **Composite Aggregation (`/api/bff/v1/dashboard`)**: Concurrently aggregates system status, active run counts, recent suites, and runner profiles within a single sub-50ms HTTP request.
- **Unified Run Detail (`/api/bff/v1/runs/{id}`)**: Enriches test run execution records with parsed summary KPIs and dynamically generated pre-signed S3 download URLs for `summary.json` and `run.log`.
- **Live SSE Telemetry Stream (`/api/bff/v1/events`)**: Streams real-time Server-Sent Events (`text/event-stream`) for run state transitions (`run_status_changed`), compilation changes (`build_status_changed`), and periodic system heartbeats (`system_heartbeat`) without high-frequency browser polling.
- **Transparent Reverse Proxying**: Routes under `/api/bff/v1/suites`, `/api/bff/v1/profiles`, `/api/bff/v1/schedules`, and `/api/bff/v1/runs` transparently proxy requests to the upstream control plane (`/api/v1/*`), handling HTTP header propagation (Bearer tokens, API keys) and connection pooling automatically.
- **Unified Ingress Smart Routing**:
  When `ingress.enabled=true` and `bff.enabled=true` (default), the chart's Ingress resource automatically partitions traffic:
  - `/api/v1/bff/auth` (and `/backchannel-logout`) → routed to the BFF service (port 8081) for OIDC session initiation, callbacks, token renewal, and backchannel logout.
  - `/api/bff/v1` → routed to the BFF service (port 8081) for sub-50ms dashboard aggregations, unified run details, and live SSE event streams.
  - `/api/v1/bff` → routed to the BFF service (port 8081) for legacy/backward-compatible endpoints.
  - `/api/v1` → routed directly to the control plane server (port 8080) for core runner callbacks, barrier rendezvous, developer CLI, and REST API access.
  - `/` → routed to the BFF service (port 8081) serving the embedded React 19 PWA dashboard and static web assets.
  - If `bff.enabled=false`, all routes fallback to the core control plane server.
- **Service Configuration**: Configured via CLI flags or environment variables:
  - `--control-plane-url` / `CONTROL_PLANE_URL`: Upstream control plane address (defaults to `http://<fullname>:8080`).
  - `--control-plane-token` / `CONTROL_PLANE_TOKEN`: Bearer token or API key forwarded in upstream requests.
  - `--control-plane-retries` / `CONTROL_PLANE_RETRIES`: Number of retry attempts on transient 5xx errors (default `2`).
  - `--sse-poll-interval` / `SSE_POLL_INTERVAL`: Frequency for polling upstream control plane state transitions (default `2s`).
  - `--sse-heartbeat-interval` / `SSE_HEARTBEAT_INTERVAL`: Keep-alive heartbeat interval for active SSE connections (default `15s`).
  - `--port` / `PORT`: Listening HTTP port (default `8081`).
- **OIDC & Token Handler Security**:
  - `KEYCLOAK_ISSUER_URL`: Realm endpoint (`http://<keycloak-svc>:8080/realms/vuhive`).
  - `KEYCLOAK_CLIENT_ID`: Confidential client ID (`vuhive-cloud-bff`).
  - `KEYCLOAK_CLIENT_SECRET`: Client secret mounted from existing Secret or generated by chart.
  - `SESSION_COOKIE_SECRET`: Encryption key for securing `HttpOnly` session cookies in browser clients.

#### Progressive Web App (PWA) & HTTPS / Ingress Requirements

The embedded dashboard delivered by the BFF operates as an installable Progressive Web App (PWA) with offline resilience:
- **Embedded Production Bundle**: Pre-compiled static assets (`manifest.webmanifest`, `sw.js`, icons, vendor chunks) are packaged directly into the Go BFF binary via `embed.FS`, requiring no external static file server.
- **Service Worker & Workbox Caching**: Workbox caches static JS/CSS/font assets (**Cache-First**) and API read responses (**Network-First** with 24-hour cache fallback).
- **Offline Query Persistence**: TanStack Query persists query caches to IndexedDB (`idb-keyval`), displaying an **Offline Mode** banner and **Offline Preview** tags when network connectivity drops.
- **HTTPS / Secure Context Requirement**: Web browsers strictly require a secure origin (`https://` or `http://localhost`) to register the service worker, activate offline caching, and trigger the native `beforeinstallprompt` installation button. When deploying in production via Kubernetes Ingress, ensure TLS termination is enabled (e.g., using `cert-manager` with an Ingress controller such as NGINX, Traefik, or AWS ALB).

### Execution Artifact Housekeeping & Retention Lifecycle Engine

The control plane includes an automated retention lifecycle worker and housekeeping subsystem:
- **Background Periodic Cleanup**: A dedicated background worker executes retention passes at configurable intervals (`housekeeping.interval`, default `1h`).
- **Independent Retention Windows**: System-wide TTLs govern execution stdout/stderr logs (`housekeeping.logsTtlDays`, default `7d`), deterministic summary reports (`housekeeping.reportsTtlDays`, default `30d`), compiled scenario binaries (`housekeeping.artifactsTtlDays`, default `180d`), and historical database test run records (`housekeeping.runsTtlDays`, default `90d`).
- **Per-Suite Policy Overrides**: Test suites can declare fine-grained retention policies stored as JSONB metadata, overriding system defaults.
- **Safe Run Archiving vs. Pruning**: Expired runs can be transitioned to `ARCHIVED` status (`housekeeping.archiveOnly: true`), stripping bulky raw execution JSON while preserving indexed latency percentiles ($p_{50}\dots p_{99}$), throughput, and error KPIs for longitudinal telemetry.
- **Orphaned & Expired Artifact Cleanup**: Abandoned or failed builds with no referring test runs are automatically purged, and expired binaries are dereferenced safely without violating foreign key constraints.
- **Native S3 Bucket Lifecycle Synchronization**: When `housekeeping.applyS3Lifecycle: true`, the control plane configures native S3 bucket lifecycle rules on startup and during housekeeping cycles, delegating automated object expiration directly to the storage subsystem (AWS S3 or MinIO).
- **On-Demand API Triggers**: Operators can trigger immediate ad-hoc housekeeping sweeps or test dry-run simulations via `POST /api/v1/system/housekeeping` and inspect active policies via `GET /api/v1/system/housekeeping/policy`.

## Configuration Parameters

| Parameter | Description | Default |
|---|---|---|
| `replicaCount` | Number of control plane replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/morphy76/vuhive-cloud/server` |
| `image.tag` | Image tag | Chart `appVersion` (`0.0.1`) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `serviceAccount.create` | Create ServiceAccount | `true` |
| `serviceAccount.automountServiceAccountToken` | Automount service account token | `true` |
| `rbac.create` | Create RBAC permissions | `true` |
| `rbac.clusterScoped` | Scope RBAC at cluster level instead of namespace level | `false` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `ingress.enabled` | Enable Ingress | `false` |
| `database.host` | PostgreSQL host | `vuhive-infra-postgresql` |
| `database.port` | PostgreSQL port | `5432` |
| `database.name` | PostgreSQL database name | `vuhive` |
| `database.user` | PostgreSQL user | `vuhive` |
| `database.password` | PostgreSQL password | `vuhive-dev` |
| `database.sslmode` | PostgreSQL SSL mode | `disable` |
| `database.existingSecret` | Existing Secret name for `DATABASE_URL` | `""` |
| `database.autoMigrate` | Run database migrations via Helm pre-install / pre-upgrade hook job | `true` |
| `s3.endpoint` | S3 endpoint URL | `http://vuhive-infra-minio:9000` |
| `s3.region` | S3 region | `us-east-1` |
| `s3.bucket` | S3 bucket name | `vuhive-artifacts` |
| `s3.accessKeyId` | S3 access key ID | `vuhive-dev` |
| `s3.secretAccessKey` | S3 secret access key | `vuhive-dev-secret` |
| `s3.usePathStyle` | Force S3 path-style addressing (`http://<endpoint>/<bucket>/`). Required for MinIO and any in-cluster S3-compatible endpoint. Automatically enabled when `s3.endpoint` is non-empty. | `true` |
| `s3.existingSecret` | Name of Secret containing AWS credentials | `""` |
| `s3.existingSecretAccessKey`| Key within `s3.existingSecret` for access key | `AWS_ACCESS_KEY_ID` |
| `s3.existingSecretSecretKey`| Key within `s3.existingSecret` for secret key | `AWS_SECRET_ACCESS_KEY` |
| `runner.namespace` | Target namespace where runner Jobs and CronJobs are spawned | `vuhive-runners` |
| `runner.createNamespace` | Automatically create `runner.namespace` if it does not exist (ignored when `rbac.clusterScoped=true` or namespace equals release namespace) | `true` |
| `runner.initImage` | Init container image fetching binaries from S3 | `ghcr.io/morphy76/vuhive-cloud/runner-init:latest` |
| `runner.defaultImage` | Default runner base image | `alpine:3.20` |
| `builder.namespace` | Namespace where test builder jobs run | `vuhive-system` |
| `builder.createNamespace` | Automatically create `builder.namespace` if it does not exist (ignored when `rbac.clusterScoped=true` or namespace equals release/runner namespace) | `true` |
| `builder.image` | Builder container image | `golang:1.26-alpine` |
| `apiCallbackUrl` | Callback URL for runner jobs. Auto-computed with path `/api/v1/runs/complete`: unqualified service name in same namespace, or trailing-dot FQDN in cross-namespace mode to prevent `ndots:5` search leaks. | Auto-computed |
| `cors.allowedOrigins` | Allowed cross-origin domains for browser clients and Swagger UI (comma-separated origins or `*`) | `*` |
| `housekeeping.enabled` | Enable background housekeeping worker and retention lifecycle engine | `true` |
| `housekeeping.interval` | Periodic interval between background housekeeping passes | `1h` |
| `housekeeping.dryRun` | Simulate cleanup without deleting storage objects or database records | `false` |
| `housekeeping.applyS3Lifecycle` | Automatically configure native S3 bucket lifecycle rules for prefix-based object expiration | `true` |
| `housekeeping.logsTtlDays` | Retention window (days) for test run execution stdout/stderr logs (`0` disables log purging) | `7` |
| `housekeeping.reportsTtlDays` | Retention window (days) for execution summary reports (`0` disables report purging) | `30` |
| `housekeeping.runsTtlDays` | Retention window (days) for historical test run database records (`0` disables run pruning) | `90` |
| `housekeeping.artifactsTtlDays` | Retention window (days) for compiled scenario binaries in S3 (`0` disables binary pruning) | `180` |
| `housekeeping.archiveOnly` | When true, transitions expired runs to `ARCHIVED` status instead of permanently deleting rows | `false` |
| `auth.enabled` | Enable OpenID Connect (OIDC) JWT authentication and RBAC for all REST endpoints | `false` |
| `auth.issuerUrl` | Keycloak realm OIDC issuer URL (e.g. `http://<keycloak-svc>:8080/realms/vuhive`) | `""` |
| `auth.jwksUrl` | Keycloak realm JWKS public keyset URL | `""` |
| `auth.runner.clientId` | Machine-to-machine OAuth2 client identifier injected into runner jobs | `vuhive-runner` |
| `auth.runner.clientSecret` | Machine-to-machine OAuth2 client secret injected into runner jobs | `vuhive-runner-secret` |
| `auth.runner.tokenUrl` | Explicit token endpoint URL for runner jobs (defaults to `<issuerUrl>/protocol/openid-connect/token`) | `""` |
| `auth.runner.existingSecret` | Name of existing Secret containing runner client secret | `""` |
| `auth.runner.existingSecretKey` | Key within `auth.runner.existingSecret` containing secret | `RUNNER_CLIENT_SECRET` |
| `bff.enabled` | Enable Backend-For-Frontend (BFF) sub-deployment and web UI | `true` |
| `bff.replicaCount` | Number of BFF replicas | `1` |
| `bff.image.repository` | BFF container image repository | `ghcr.io/morphy76/vuhive-cloud/bff` |
| `bff.image.tag` | BFF image tag | Chart `appVersion` (`0.0.1`) |
| `bff.image.pullPolicy` | BFF image pull policy | `IfNotPresent` |
| `bff.service.type` | BFF Service type | `ClusterIP` |
| `bff.service.port` | BFF Service port | `8081` |
| `bff.controlPlaneUrl` | Upstream control plane URL override (defaults to `http://<fullname>:8080`) | `""` |
| `bff.controlPlaneToken` | Bearer token for control plane if required | `""` |
| `bff.ssePollInterval` | Polling frequency for upstream run/build status transitions | `2s` |
| `bff.sseHeartbeatInterval` | Keep-alive heartbeat interval for SSE client streams | `15s` |
| `bff.keycloak.issuerUrl` | Keycloak realm endpoint for BFF Token Handler | `""` |
| `bff.keycloak.clientId` | Keycloak confidential client ID for BFF | `vuhive-cloud-bff` |
| `bff.keycloak.clientSecret` | Keycloak confidential client secret for BFF | `""` |
| `bff.keycloak.clientSecretExistingSecret` | Name of existing Secret containing BFF client secret | `""` |
| `bff.keycloak.clientSecretKey` | Key within `clientSecretExistingSecret` containing secret | `client-secret` |
| `bff.keycloak.sessionCookieSecret` | Session cookie encryption secret | `""` |
| `bff.keycloak.sessionCookieSecretRef` | Name of existing Secret containing session cookie encryption secret | `""` |
| `bff.keycloak.sessionCookieSecretKey` | Key within `sessionCookieSecretRef` containing secret | `session-cookie-secret` |
| `bff.keycloak.sessionCookieName` | Cookie name for encrypted browser session | `vuhive_session` |
| `bff.database.url` | PostgreSQL connection URL for persistent HTTP session store (`bff_sessions`) | `""` |
| `bff.database.existingSecret` | Name of existing Secret containing PostgreSQL connection URL | `""` |
| `bff.database.existingSecretKey` | Key within `existingSecret` containing the database connection URL | `database-url` |
| `bff.session.encryptionKey` | 32-byte AES-256-GCM encryption key for stored OAuth tokens at rest | `""` |
| `bff.session.encryptionKeyExistingSecret` | Name of existing Secret containing 32-byte session token encryption key | `""` |
| `bff.session.encryptionKeyKey` | Key within `encryptionKeyExistingSecret` containing the encryption key | `session-encryption-key` |


