# vuhive-cloud-infra

Helm chart to deploy backing infrastructure services (PostgreSQL and MinIO) for `vuhive-cloud`.

## Overview

This chart bundles backing infrastructure dependencies for the `vuhive-cloud` control plane:
- **`groundhog2k/postgres`**: Lightweight, secure PostgreSQL 16+ instance.
- **`minio/minio`**: S3-compatible standalone object storage server with pre-created buckets.
- **OpenAPI Viewer (`swaggerapi/swagger-ui`)**: Optional third-party interactive UI viewer for testing and exploring control plane APIs without bundling UI assets into the core Go control plane binary.

> [!WARNING]
> This chart is intended for **local development and evaluation** only (e.g., Rancher Desktop, Kind, Minikube). For production deployments, provision PostgreSQL and S3/MinIO via managed cloud services and reference them from the [`vuhive-cloud`](../vuhive-cloud/README.md) chart using `existingSecret`.

> **Documentation Navigation**:
> - **System Architecture & Overview**: [`README.md`](../../README.md) and [`ARCHITECTURE_SPEC.md`](../../ARCHITECTURE_SPEC.md)
> - **Developer & Contributor Guide**: [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
> - **Control Plane Helm Chart**: [`deploy/helm/vuhive-cloud/README.md`](../vuhive-cloud/README.md)
> - **Adoption Guide & API Recipes**: [`docs/cookbook.md`](../../docs/cookbook.md)
> - **REST API Reference**: [`api/openapi.yaml`](../../api/openapi.yaml)
> - **Engineering Philosophy**: [`AI_DISCLOSURE.md`](../../AI_DISCLOSURE.md)

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

### Enabling the OpenAPI Viewer (Swagger UI)

To deploy the optional interactive OpenAPI viewer alongside PostgreSQL and MinIO:

```bash
helm install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace vuhive-system \
  --create-namespace \
  --set openapiViewer.enabled=true \
  --wait --timeout=180s
```

Once deployed, access the OpenAPI viewer locally via port-forwarding:

```bash
kubectl port-forward -n vuhive-system svc/vuhive-infra-vuhive-cloud-infra-openapi-viewer 8081:8080
```

Then navigate to `http://localhost:8081` in your browser. The viewer fetches the control plane's machine-readable specification from `http://vuhive-vuhive-cloud:8080/openapi.json` (configurable via `openapiViewer.specUrl`).

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
| `openapiViewer.enabled` | Deploy optional third-party OpenAPI viewer (Swagger UI) | `false` |
| `openapiViewer.image.repository` | Container image repository for OpenAPI viewer | `swaggerapi/swagger-ui` |
| `openapiViewer.image.tag` | Container image tag | `v5.18.2` |
| `openapiViewer.image.pullPolicy` | Container image pull policy | `IfNotPresent` |
| `openapiViewer.specUrl` | Target URL to control plane OpenAPI specification | `http://vuhive-vuhive-cloud:8080/openapi.json` |
| `openapiViewer.service.type` | Kubernetes service type | `ClusterIP` |
| `openapiViewer.service.port` | Kubernetes service port | `8080` |
| `openapiViewer.ingress.enabled` | Enable Kubernetes Ingress for OpenAPI viewer | `false` |
| `openapiViewer.ingress.className` | Ingress class name | `""` |
| `openapiViewer.ingress.hosts[0].host` | Ingress host | `openapi.local` |
| `openapiViewer.ingress.hosts[0].paths[0].path` | Ingress path | `/` |

> **Note:** Default credentials are intended for local development only.
> Always override secrets in production using `existingSecret` references.
