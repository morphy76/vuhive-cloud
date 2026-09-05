# vuhive-cloud-infra

Helm chart to deploy backing infrastructure services (PostgreSQL and MinIO) for `vuhive-cloud`.

## Overview

This chart bundles the two external dependencies required by the `vuhive-cloud` control plane:

The `vuhive-cloud-infra` umbrella chart bundles community-proven Helm subcharts:
- **`groundhog2k/postgres`**: Lightweight, secure PostgreSQL 16+ instance.
- **`minio/minio`**: S3-compatible standalone object storage server with pre-created buckets.

> [!WARNING]
> This chart is intended for **local development and evaluation** only (e.g., Rancher Desktop, Kind, Minikube). For production deployments, provision PostgreSQL and S3/MinIO via managed cloud services and reference them from the [`vuhive-cloud`](../vuhive-cloud/README.md) chart using `existingSecret`.

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+ / Helm 4+
- Helm repositories:
  - `groundhog2k` — `https://groundhog2k.github.io/helm-charts/`
  - `minio` — `https://charts.min.io/`

## Quickstart

```bash
# Add dependency chart repositories
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

# Build chart dependencies
helm dependency build deploy/helm/vuhive-cloud-infra

# Install infrastructure chart
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s
```

## Configuration Parameters

| Parameter | Description | Default |
|---|---|---|
| `postgresql.settings.superuserPassword` | PostgreSQL superuser password | `vuhive-dev-root` |
| `postgresql.userDatabase.name` | Application database name | `vuhive` |
| `postgresql.userDatabase.user` | Application database user | `vuhive` |
| `postgresql.userDatabase.password` | Application database password | `vuhive-dev` |
| `minio.rootUser` | MinIO root user | `vuhive-dev` |
| `minio.rootPassword` | MinIO root password | `vuhive-dev-secret` |
| `minio.buckets[0].name` | Default artifact bucket name | `vuhive-artifacts` |
| `minio.buckets[0].policy` | Default artifact bucket policy | `none` |

> **Note:** Default credentials are intended for local development only.
> Always override secrets in production using `existingSecret` references.
