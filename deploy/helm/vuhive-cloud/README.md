# vuhive-cloud

Official Helm chart for the `vuhive-cloud` control plane.

## Overview

`vuhive-cloud` provides the Kubernetes-native control plane for orchestrating distributed load testing suites and runner jobs.

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

In this case no extra namespaces are created and no cross-namespace RBAC is needed.

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
| `s3.usePathStyle` | S3 path style addressing | `true` |
| `s3.existingSecret` | Existing Secret name for AWS credentials | `""` |
| `runner.namespace` | Namespace where runner jobs are spawned | `vuhive-runners` |
| `runner.createNamespace` | Automatically create `runner.namespace` if it does not exist (ignored when `rbac.clusterScoped=true` or namespace equals release namespace) | `true` |
| `runner.initImage` | Runner init container image | `ghcr.io/morphy76/vuhive-cloud/runner-init:latest` |
| `runner.defaultImage` | Default runner base image | `alpine:3.20` |
| `builder.namespace` | Namespace where test builder jobs run | `vuhive-system` |
| `builder.createNamespace` | Automatically create `builder.namespace` if it does not exist (ignored when `rbac.clusterScoped=true` or namespace equals release/runner namespace) | `true` |
| `builder.image` | Builder container image | `golang:1.26-alpine` |
| `apiCallbackUrl` | Callback URL for runner jobs | Auto-computed |
