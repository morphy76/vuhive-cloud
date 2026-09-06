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
- **Backend-For-Frontend (`cmd/bff`)**: Lightweight gateway serving the embedded React 19 SPA web dashboard and PWA assets directly via Go `embed.FS`, managing client sessions, and aggregating API calls.

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

### 2. Deploy vuhive-cloud Control Plane

With default values, Helm automatically creates the `vuhive-runners` and `vuhive-system` namespaces
(see [Namespace Management](#namespace-management) below):

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set database.host=vuhive-infra-postgresql \
  --set s3.endpoint=http://vuhive-infra-minio:9000
```

> [!NOTE]
> When `s3.endpoint` is non-empty (as in the MinIO case above), `s3.usePathStyle` is automatically treated as `true` by the control plane server, runner-init, and runner-wrapper. Path-style addressing (`http://<endpoint>/<bucket>/`) is required for MinIO because virtual-hosted-style URLs (`http://<bucket>.<service>/`) depend on DNS wildcards unavailable for Kubernetes Service names.

### 3. Local Cluster Testing with Locally Built Images

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

For full details on local container building and troubleshooting `ImagePullBackOff` issues, see [`CONTRIBUTING.md`](../../CONTRIBUTING.md).

### 4. Production Deployment (with External PostgreSQL & S3)

In production, backing services should be provisioned via managed cloud infrastructure (e.g., AWS Aurora PostgreSQL and AWS S3).

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

### CronJob Run Correlation

When a `CronJob` fires a `batch/v1` Job:

1. The runner pod's `VUHIVE_RUN_ID` is populated from `metadata.labels['batch.kubernetes.io/job-name']` (the **Job name**, e.g. `vuhive-sched-abc-28492020`), not from `metadata.name` (the pod name).
2. The `RunnerJobWatcher` informer auto-creates a `TestRun` record linked to the schedule, and records both `k8s_job_name` and the **actual `k8s_namespace`** from `runner.namespace` (not a hard-coded default).
3. When the runner-wrapper POSTs the completion callback with `run_id = <job-name>`, the control plane first attempts UUID lookup, then falls back to `k8s_job_name` correlation, ensuring the `TestRun` is correctly finalized with summary KPIs regardless of whether the run was dispatched ad-hoc or via a CronJob.


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
