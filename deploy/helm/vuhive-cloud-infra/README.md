# vuhive-cloud-infra Helm Chart

Helm chart to deploy backing infrastructure services (PostgreSQL and MinIO) for local evaluation, development, and testing of **`vuhive-cloud`**.

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quickstart Installation](#quickstart-installation)
- [Backing Services & Connection Details](#backing-services--connection-details)
  - [PostgreSQL Database](#postgresql-database)
  - [MinIO Object Storage](#minio-object-storage)
- [Configuration Reference](#configuration-reference)
- [Persistent Storage & Volumes](#persistent-storage--volumes)
- [Teardown & Cleanup](#teardown--cleanup)

---

## Overview

The `vuhive-cloud` control plane depends on two primary infrastructure primitives:
1. **Relational Database (PostgreSQL)**: Stores test suites, compiled artifact metadata, runner profiles, CronJob schedules, and test run KPI metrics.
2. **Object Storage (S3 / MinIO)**: Stores uploaded source archives, compiled Linux executables, execution logs, and detailed `summary.json` performance reports.

The `vuhive-cloud-infra` umbrella chart bundles community-proven Helm subcharts:
- **`groundhog2k/postgres`**: Lightweight, secure PostgreSQL 16+ instance.
- **`minio/minio`**: S3-compatible standalone object storage server with pre-created buckets.

---

## Prerequisites

- **Kubernetes**: Version `1.28+`
- **Helm**: Version `3.10+` or `Helm 4+`
- **Kubernetes StorageClass**: Default StorageClass supporting dynamic `PersistentVolumeClaim` provisioning (e.g. `local-path`, `standard`, or `gp3`).

---

## Quickstart Installation

```bash
# 1. Add dependency chart repositories
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

# 2. Build chart dependencies
helm dependency build deploy/helm/vuhive-cloud-infra

# 3. Install infrastructure into vuhive-system namespace
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --wait --timeout=180s
```

---

## Backing Services & Connection Details

When installed with default release name `vuhive-infra` in namespace `vuhive-system`, the services expose the following endpoints:

### PostgreSQL Database

- **Service Endpoint**: `vuhive-infra-postgresql.vuhive-system.svc.cluster.local:5432`
- **Default Database**: `vuhive`
- **Default Username**: `vuhive`
- **Default Password**: `vuhive-dev`
- **Connection URL**:
  ```text
  postgres://vuhive:vuhive-dev@vuhive-infra-postgresql:5432/vuhive?sslmode=disable
  ```

### MinIO Object Storage

- **API Endpoint**: `http://vuhive-infra-minio.vuhive-system.svc.cluster.local:9000`
- **Console Endpoint**: `http://vuhive-infra-minio.vuhive-system.svc.cluster.local:9001`
- **Root Access Key**: `vuhive-dev`
- **Root Secret Key**: `vuhive-dev-secret`
- **Default Bucket**: `vuhive-artifacts` (automatically provisioned upon startup)

---

## Configuration Reference

The following table summarizes parameters configured in `values.yaml`:

| Parameter | Description | Default |
|---|---|---|
| `postgresql.enabled` | Enable PostgreSQL subchart | `true` |
| `postgresql.settings.superuser` | PostgreSQL superuser username | `postgres` |
| `postgresql.settings.superuserPassword` | PostgreSQL superuser password | `vuhive-dev` |
| `postgresql.userDatabase.name` | Application database name | `vuhive` |
| `postgresql.userDatabase.user` | Application database user | `vuhive` |
| `postgresql.userDatabase.password` | Application database password | `vuhive-dev` |
| `postgresql.storage.requestedSize` | Storage volume size for PostgreSQL | `1Gi` |
| `minio.enabled` | Enable MinIO subchart | `true` |
| `minio.mode` | MinIO operating mode (`standalone` or `distributed`) | `standalone` |
| `minio.rootUser` | Root admin access key | `vuhive-dev` |
| `minio.rootPassword` | Root admin secret key | `vuhive-dev-secret` |
| `minio.buckets[0].name` | Pre-created default bucket name | `vuhive-artifacts` |
| `minio.buckets[0].policy` | Access policy for default bucket | `none` |
| `minio.buckets[0].purge` | Purge bucket on install/upgrade | `false` |
| `minio.persistence.size` | Storage volume size for MinIO | `2Gi` |
| `minio.resources.requests.memory` | Memory request for MinIO pod | `256Mi` |

---

## Persistent Storage & Volumes

Both PostgreSQL and MinIO create `PersistentVolumeClaim` (PVC) resources to preserve database state and artifacts across pod restarts:

```bash
kubectl get pvc -n vuhive-system -l app.kubernetes.io/managed-by=Helm
```

> [!TIP]
> In lightweight local development environments (like Rancher Desktop or Kind), the default StorageClass automatically fulfills these claims on local disk.

---

## Teardown & Cleanup

To uninstall the infrastructure release:

```bash
helm uninstall vuhive-infra --namespace vuhive-system
```

To purge persistent data volumes completely:

```bash
kubectl delete pvc -n vuhive-system \
  -l release=vuhive-infra
```
