# vuhive-cloud Helm Chart

Official Helm chart for deploying the **`vuhive-cloud`** control plane on Kubernetes.

`vuhive-cloud` is a cloud-native, distributed load-testing control plane that orchestrates ephemeral test compilation build jobs, manages reusable runner profiles, schedules native Kubernetes CronJobs, coordinates multi-pod start barriers, and ingests execution telemetry into PostgreSQL and S3/MinIO.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Architecture in Kubernetes](#architecture-in-kubernetes)
- [Deployment Modes](#deployment-modes)
  - [1. Local / Evaluation Deployment (with vuhive-cloud-infra)](#1-local--evaluation-deployment-with-vuhive-cloud-infra)
  - [2. Production Deployment (with External PostgreSQL & S3)](#2-production-deployment-with-external-postgresql--s3)
- [Configuration Reference](#configuration-reference)
- [Production Best Practices & Caveats](#production-best-practices--caveats)
  - [Database Migrations & Lifecycle](#database-migrations--lifecycle)
  - [API Callback URL & DNS Resolution](#api-callback-url--dns-resolution)
  - [RBAC Scoping & Multi-Namespace Isolation](#rbac-scoping--multi-namespace-isolation)
  - [External Secrets Management](#external-secrets-management)
  - [Pod Security Standards (PSS) & Hardening](#pod-security-standards-pss--hardening)
- [Operations & Troubleshooting](#operations--troubleshooting)
  - [Verifying Health & Status](#verifying-health--status)
  - [Accessing the Control Plane API & Querying Runs](#accessing-the-control-plane-api--querying-runs)
  - [Upgrading the Chart](#upgrading-the-chart)
  - [Uninstalling the Chart](#uninstalling-the-chart)

---

## Prerequisites

- **Kubernetes**: Version `1.28+`
- **Helm**: Version `3.10+` or `Helm 4+`
- **Backing Services**:
  - **PostgreSQL**: Version `14+` (Version `16+` recommended)
  - **Object Storage**: AWS S3 or MinIO (S3-compatible)

---

## Architecture in Kubernetes

The `vuhive-cloud` Helm chart deploys the following Kubernetes primitives:

```text
┌────────────────────────────────────────────────────────────────────────┐
│ Namespace: vuhive-system                                               │
│                                                                        │
│   ┌─────────────────────────────┐      ┌───────────────────────────┐   │
│   │ Deployment: vuhive-cloud    │◄────►│ Service: vuhive-cloud     │   │
│   │ (Control Plane, REST API)   │      │ (ClusterIP, Port 8080)    │   │
│   └──────────────┬──────────────┘      └───────────────────────────┘   │
│                  │                                                     │
│                  │ (Orchestrates Ephemeral Compilation Jobs)           │
│                  ▼                                                     │
│   ┌─────────────────────────────┐                                      │
│   │ batch/v1 Jobs (Builder)     │ ──► Writes Binaries to S3            │
│   │ (golang:1.26-alpine)        │                                      │
│   └─────────────────────────────┘                                      │
└──────────────────┬─────────────────────────────────────────────────────┘
                   │
                   │ (Orchestrates Runner Jobs & CronJobs)
                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│ Namespace: vuhive-runners                                              │
│                                                                        │
│   ┌─────────────────────────────┐      ┌───────────────────────────┐   │
│   │ batch/v1 CronJobs           │ ──►  │ batch/v1 Jobs (Runners)   │   │
│   │ (Scheduled Workloads)       │      │ - runner-init (downloads) │   │
│   └─────────────────────────────┘      │ - runner (executes test)  │   │
│                                        │ - Posts back to API       │   │
│                                        └───────────────────────────┘   │
└────────────────────────────────────────────────────────────────────────┘
```

---

## Deployment Modes

### 1. Local / Evaluation Deployment (with `vuhive-cloud-infra`)

For local development (e.g. Rancher Desktop, Kind, Minikube) or evaluation clusters, deploy backing services using the companion `vuhive-cloud-infra` chart, then deploy the control plane:

```bash
# 1. Add required repositories and build infra dependencies
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update
helm dependency build deploy/helm/vuhive-cloud-infra

# 2. Install backing infrastructure (PostgreSQL + MinIO)
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s

# 3. Deploy vuhive-cloud control plane
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --set database.host=vuhive-infra-postgresql \
  --set s3.endpoint=http://vuhive-infra-minio:9000 \
  --wait --timeout=120s
```

### 2. Production Deployment (with External PostgreSQL & S3)

In production, backing services should be provisioned via managed cloud infrastructure (e.g., AWS Aurora PostgreSQL and AWS S3).

#### Step 1: Create External Secrets

```bash
# Create database secret
kubectl create secret generic vuhive-db-secret -n vuhive-system \
  --from-literal=DATABASE_URL="postgres://vuhive_user:StrongPassword@postgres.prod.internal:5432/vuhive?sslmode=require"

# Create S3 credentials secret
kubectl create secret generic vuhive-s3-secret -n vuhive-system \
  --from-literal=AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE" \
  --from-literal=AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

#### Step 2: Deploy with Production Overrides

Create `prod-values.yaml`:

```yaml
replicaCount: 2

image:
  repository: ghcr.io/morphy76/vuhive-cloud/server
  tag: "0.1.0"
  pullPolicy: IfNotPresent

resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi

rbac:
  create: true
  clusterScoped: false

database:
  existingSecret: "vuhive-db-secret"
  existingSecretKey: "DATABASE_URL"
  autoMigrate: true

s3:
  region: "us-east-1"
  bucket: "prod-vuhive-artifacts"
  usePathStyle: false
  existingSecret: "vuhive-s3-secret"

runner:
  namespace: "vuhive-runners"
  initImage: "ghcr.io/morphy76/vuhive-cloud/runner-init:0.1.0"

builder:
  namespace: "vuhive-system"

apiCallbackUrl: "http://vuhive-vuhive-cloud.vuhive-system.svc.cluster.local:8080/api/v1/runs/complete"

ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: vuhive.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: vuhive-tls-cert
      hosts:
        - vuhive.example.com
```

Deploy the chart:

```bash
helm install vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  --create-namespace \
  -f prod-values.yaml \
  --wait --timeout=180s
```

---

## Configuration Reference

The following table lists the configurable parameters of the `vuhive-cloud` chart and their default values.

| Parameter | Description | Default |
|---|---|---|
| `replicaCount` | Number of control plane pod replicas | `1` |
| `image.repository` | Control plane container image repository | `ghcr.io/morphy76/vuhive-cloud/server` |
| `image.tag` | Control plane container image tag (overrides `appVersion`) | `""` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Secrets for pulling images from private registries | `[]` |
| `nameOverride` | Override chart release name | `""` |
| `fullnameOverride` | Override full chart release name | `""` |
| `serviceAccount.create` | Create a dedicated ServiceAccount | `true` |
| `serviceAccount.name` | Custom ServiceAccount name (auto-generated if empty) | `""` |
| `serviceAccount.annotations` | Annotations for ServiceAccount (e.g. AWS IRSA) | `{}` |
| `serviceAccount.automountServiceAccountToken` | Mount ServiceAccount token into pod | `true` |
| `rbac.create` | Generate RBAC Roles & RoleBindings | `true` |
| `rbac.clusterScoped` | Create ClusterRoles instead of namespace-scoped Roles | `false` |
| `podAnnotations` | Custom annotations attached to control plane pods | `{}` |
| `podLabels` | Custom labels attached to control plane pods | `{}` |
| `podSecurityContext.runAsNonRoot` | Enforce non-root execution | `true` |
| `podSecurityContext.runAsUser` | Non-root user ID | `10001` |
| `podSecurityContext.runAsGroup` | Non-root group ID | `10001` |
| `podSecurityContext.fsGroup` | Storage filesystem group ID | `10001` |
| `securityContext.allowPrivilegeEscalation` | Disallow container privilege escalation | `false` |
| `securityContext.readOnlyRootFilesystem` | Mount container root filesystem read-only | `true` |
| `securityContext.capabilities.drop` | Linux capabilities dropped from container | `["ALL"]` |
| `service.type` | Kubernetes Service type (`ClusterIP`, `NodePort`, `LoadBalancer`) | `ClusterIP` |
| `service.port` | External service port exposed by cluster service | `8080` |
| `service.targetPort` | Internal container port | `8080` |
| `service.annotations` | Annotations for the Service | `{}` |
| `ingress.enabled` | Enable Kubernetes Ingress resource | `false` |
| `ingress.className` | Ingress controller class name (e.g., `nginx`, `traefik`) | `""` |
| `ingress.annotations` | Annotations for Ingress | `{}` |
| `ingress.hosts` | List of host rules for Ingress | `[{"host":"vuhive-cloud.local","paths":[{"path":"/","pathType":"Prefix"}]}]` |
| `ingress.tls` | TLS configuration for Ingress hosts | `[]` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `livenessProbe.httpGet.path` | Liveness probe HTTP path | `/healthz` |
| `readinessProbe.httpGet.path` | Readiness probe HTTP path | `/healthz` |
| `database.host` | PostgreSQL host | `vuhive-infra-postgresql` |
| `database.port` | PostgreSQL port | `5432` |
| `database.name` | Database name | `vuhive` |
| `database.user` | Database username | `vuhive` |
| `database.password` | Database password | `vuhive-dev` |
| `database.sslmode` | SSL connection mode (`disable`, `require`, `verify-full`) | `disable` |
| `database.url` | Full database connection URL string (overrides host/user/pass) | `""` |
| `database.existingSecret` | Name of Secret containing database connection string | `""` |
| `database.existingSecretKey` | Key within `database.existingSecret` containing the URL | `DATABASE_URL` |
| `database.autoMigrate` | Run automated database schema migrations via Helm pre-install hook | `true` |
| `s3.endpoint` | S3 / MinIO endpoint URL (leave empty for AWS S3) | `http://vuhive-infra-minio:9000` |
| `s3.region` | S3 region | `us-east-1` |
| `s3.bucket` | S3 bucket name for artifacts, binaries, and logs | `vuhive-artifacts` |
| `s3.accessKeyId` | S3 access key ID | `vuhive-dev` |
| `s3.secretAccessKey` | S3 secret access key | `vuhive-dev-secret` |
| `s3.usePathStyle` | Use path-style addressing (`true` for MinIO, `false` for AWS S3) | `true` |
| `s3.existingSecret` | Name of Secret containing AWS credentials | `""` |
| `s3.existingSecretAccessKey`| Key within `s3.existingSecret` for access key | `AWS_ACCESS_KEY_ID` |
| `s3.existingSecretSecretKey`| Key within `s3.existingSecret` for secret key | `AWS_SECRET_ACCESS_KEY` |
| `runner.namespace` | Target namespace where runner Jobs and CronJobs are spawned | `vuhive-runners` |
| `runner.initImage` | Init container image fetching binaries from S3 | `ghcr.io/morphy76/vuhive-cloud/runner-init:latest` |
| `runner.defaultImage` | Default runner base image | `alpine:3.20` |
| `builder.namespace` | Target namespace where ephemeral Go build jobs run | `vuhive-system` |
| `builder.image` | Ephemeral builder container image | `golang:1.26-alpine` |
| `apiCallbackUrl` | Explicit API callback URL used by runner jobs | Auto-computed |
| `nodeSelector` | Node selector labels for control plane pods | `{}` |
| `tolerations` | Node tolerations for control plane pods | `[]` |
| `affinity` | Node affinity rules for control plane pods | `{}` |
| `extraEnv` | Additional environment variables injected into control plane | `[]` |
| `extraEnvFrom` | Additional configmaps or secrets injected via `envFrom` | `[]` |

---

## Production Best Practices & Caveats

### Database Migrations & Lifecycle

`vuhive-cloud` manages its database schema via Goose migrations (`000001_init_schema.sql`):

1. **Helm Pre-Install Hook (`database.autoMigrate: true`)**:
   A dedicated pre-install and pre-upgrade Kubernetes Job runs `vuhive-cloud --migrate-only` before the application pods roll out. This guarantees that schema changes are in place before traffic shifts.
2. **Startup Migrations**:
   The control plane also runs automatic migration validation upon initialization if `AUTO_MIGRATE=true`.
3. **Manual CLI Migrations**:
   To execute migrations independently without Helm:
   ```bash
   vuhive-cloud --migrate-only
   ```

### API Callback URL & DNS Resolution

Runner pods execute in the `runner.namespace` (e.g. `vuhive-runners`) while the control plane lives in `vuhive-system`.

> [!WARNING]
> If `apiCallbackUrl` is left blank, the Helm chart automatically templates `http://<release>-vuhive-cloud.<release-namespace>.svc.cluster.local:8080/api/v1/runs/complete`.
> When overriding `apiCallbackUrl`, **always use the fully qualified domain name (FQDN)**. Using short names (e.g. `http://vuhive-vuhive-cloud:8080`) can cause `ndots:5` search domain path traversal leaks in pods running in separate namespaces.

### RBAC Scoping & Multi-Namespace Isolation

- **Scoped RBAC (`rbac.clusterScoped: false`)**:
  Creates isolated Kubernetes `Role` and `RoleBinding` resources strictly bound to the release namespace (`vuhive-system`) and the runner namespace (`vuhive-runners`). This follows the principle of least privilege.
- **Cluster-Scoped RBAC (`rbac.clusterScoped: true`)**:
  Creates a `ClusterRole` and `ClusterRoleBinding`. Use this mode when testing workloads span dynamic or tenant-specific runner namespaces.

### External Secrets Management

Avoid committing plain-text database URLs or AWS access keys to Git repositories. Supply them using `existingSecret`:

```yaml
database:
  existingSecret: "my-db-credentials"
  existingSecretKey: "DATABASE_URL"

s3:
  existingSecret: "my-s3-credentials"
  existingSecretAccessKey: "AWS_ACCESS_KEY_ID"
  existingSecretSecretKey: "AWS_SECRET_ACCESS_KEY"
```

### Pod Security Standards (PSS) & Hardening

The control plane and runner pods are engineered to comply with Kubernetes **Restricted** Pod Security Standards:
- Executed under non-root UID `10001` (`runAsNonRoot: true`).
- Container root filesystems are mounted read-only (`readOnlyRootFilesystem: true`).
- All Linux capabilities are stripped (`capabilities.drop: ["ALL"]`).
- Privilege escalation is disabled (`allowPrivilegeEscalation: false`).

---

## Operations & Troubleshooting

### Verifying Health & Status

```bash
# Check pod status and rollout
kubectl get pods -n vuhive-system -l app.kubernetes.io/name=vuhive-cloud

# Check liveness and health endpoints
kubectl exec -n vuhive-system deploy/vuhive-vuhive-cloud -- \
  wget -qO- http://localhost:8080/healthz
```

### Accessing the Control Plane API & Querying Runs

Port-forward the control plane service to query runs, reports, and logs:

```bash
# Port-forward API service locally
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080

# Query test runs (supports ?suite_id=, ?status=, ?limit=, ?offset=, etc.)
curl -s http://localhost:8080/api/v1/runs?status=COMPLETED | jq .

# Inspect run metadata, duration, exit code and SLA status
curl -s http://localhost:8080/api/v1/runs/<run-id> | jq .

# Retrieve full execution report summary.json (direct JSON or ?presign=true URL)
curl -s http://localhost:8080/api/v1/runs/<run-id>/report | jq .

# Retrieve full execution logs (direct text or ?presign=true URL)
curl -s http://localhost:8080/api/v1/runs/<run-id>/logs
```

For complete curl workflows and API examples, see the **[Adoption Cookbook (`docs/cookbook.md`)](../../docs/cookbook.md)** and the **[OpenAPI 3.0.3 Specification (`api/openapi.yaml`)](../../api/openapi.yaml)**.

### Upgrading the Chart

```bash
helm upgrade vuhive deploy/helm/vuhive-cloud \
  --namespace vuhive-system \
  -f prod-values.yaml \
  --wait
```

### Uninstalling the Chart

```bash
helm uninstall vuhive --namespace vuhive-system
```

> [!NOTE]
> Uninstalling the control plane chart does not delete the database tables or S3 objects, preserving your historical test run reports.
