# vuhive-cloud-infra

Helm chart to deploy backing infrastructure services (PostgreSQL and MinIO) for `vuhive-cloud`.

## Overview

This chart bundles backing infrastructure dependencies for the `vuhive-cloud` control plane:
- **`groundhog2k/postgres`**: Lightweight, secure PostgreSQL 16+ instance.
- **`minio/minio`**: S3-compatible standalone object storage server with pre-created buckets.
- **`keycloak`**: Containerized OIDC Identity Provider and authorization server with declarative realm import.
- **OpenAPI Viewer (`swaggerapi/swagger-ui`)**: Optional third-party interactive UI viewer for testing and exploring control plane APIs without bundling UI assets into the core Go control plane binary.

> [!WARNING]
> This chart is intended for **local development and evaluation** only (e.g., Rancher Desktop, Kind, Minikube). For production deployments, provision PostgreSQL, S3/Object Storage, and OIDC Identity Providers via managed cloud services or enterprise clusters, and reference them from the [`vuhive-cloud`](../vuhive-cloud/README.md) chart using `existingSecret` and `values-production.yaml`. See [External Infrastructure & Third-Party Deployment Scenarios](../../README.md#external-infrastructure--third-party-deployment-scenarios) for architecture and deployment guidance.

> **Documentation Navigation**:
> - **System Architecture & Overview**: [`README.md`](../../README.md) and [`ARCHITECTURE_SPEC.md`](../../ARCHITECTURE_SPEC.md)
> - **Developer & Contributor Guide**: [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
> - **Control Plane Helm Chart**: [`deploy/helm/vuhive-cloud/README.md`](../vuhive-cloud/README.md)
> - **Adoption Guide & API Recipes**: [`docs/cookbook.md`](../../docs/cookbook.md)
> - **REST API Reference**: [OpenAPI 3.1 Specification (`api/openapi.yaml`)](../../api/openapi.yaml) (served live at `GET /openapi.yaml` and `GET /openapi.json`)
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
  --set openapiViewer.specUrl="http://localhost:8080/openapi.json" \
  --wait --timeout=180s
```

> [!IMPORTANT]
> **Browser-Accessible `specUrl` Requirement**:
> Swagger UI is a client-side Single Page Application (SPA) executed directly inside the operator's desktop browser (not a server-side proxy in the cluster). When loaded, the browser directly resolves and fetches the specification URL (`specUrl`).
>
> - **In-Cluster Default (`http://vuhive-vuhive-cloud:8080/openapi.json`)**: Resolves only inside the Kubernetes pod network. Desktop browsers accessing Swagger UI externally cannot resolve Kubernetes internal DNS names (`vuhive-vuhive-cloud`), causing `ERR_NAME_NOT_RESOLVED`.
> - **Local Development via `kubectl port-forward`**: Configure `specUrl: "http://localhost:8080/openapi.json"` (as shown above) and port-forward both the viewer and the control plane to your local machine.
> - **Ingress / Shared Domain**: When exposing Swagger UI and the control plane under the same ingress hostname, configure a relative path (e.g., `--set openapiViewer.specUrl="/openapi.json"`).

#### Accessing Swagger UI via Port-Forwarding

When running locally, port-forward both the OpenAPI viewer and the `vuhive-cloud` control plane:

```bash
# 1. Port-forward the OpenAPI Swagger UI viewer (port 8081)
kubectl port-forward -n vuhive-system svc/vuhive-infra-vuhive-cloud-infra-openapi-viewer 8081:8080

# 2. In a separate terminal, port-forward the control plane service (port 8080)
kubectl port-forward -n vuhive-system svc/vuhive-vuhive-cloud 8080:8080
```

Then navigate to `http://localhost:8081` in your desktop browser. Swagger UI initiates a browser `fetch()` to `http://localhost:8080/openapi.json`.

Because modern web browsers enforce the Same-Origin Policy when fetching resources across different ports or hostnames, the `vuhive-cloud` control plane includes built-in Cross-Origin Resource Sharing (CORS) middleware and responds to HTTP `OPTIONS` preflight requests with `204 No Content` and standard CORS headers (`Access-Control-Allow-Origin: *`), ensuring seamless API exploration and ad-hoc request testing without browser blocks.

### Keycloak (OIDC Identity Provider & Authorization Server)

Keycloak provides OIDC authentication and token issuance for the control plane and developer CLI:
- **Image**: `quay.io/keycloak/keycloak:26.1.0`
- **Database Backend**: Automatically connects to the in-chart PostgreSQL instance (`vuhive-infra-postgresql`).
- **Declarative Realm Import**: Imports `files/vuhive-realm.json` defining the `vuhive` realm with **zero pre-created users**, standard roles (`vuhive-admin`, `vuhive-deployer`, `vuhive-developer`, `vuhive-viewer`, `vuhive-runner`), groups (`/administrators`, `/deployers`, `/developers`, `/viewers`), and clients (`vuhive-cloud-api`, `vuhive-cloud-cli`, `vuhive-runner`, `vuhive-cloud-bff`).
- **Accessing Keycloak Admin Console via Port-Forwarding**:
  ```bash
  kubectl port-forward -n vuhive-system svc/vuhive-infra-vuhive-cloud-infra-keycloak 8082:8080
  ```
  Navigate to `http://localhost:8082` and sign in with admin credentials (`admin` / `admin`).

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
| `keycloak.enabled` | Deploy Keycloak OIDC identity provider | `true` |
| `keycloak.image.repository` | Container image repository for Keycloak | `quay.io/keycloak/keycloak` |
| `keycloak.image.tag` | Container image tag | `26.1.0` |
| `keycloak.adminUser` | Keycloak bootstrap administrator username | `admin` |
| `keycloak.adminPassword` | Keycloak bootstrap administrator password | `admin` |
| `keycloak.service.port` | Keycloak service port | `8080` |
| `openapiViewer.enabled` | Deploy optional third-party OpenAPI viewer (Swagger UI) | `false` |
| `openapiViewer.image.repository` | Container image repository for OpenAPI viewer | `swaggerapi/swagger-ui` |
| `openapiViewer.image.tag` | Container image tag | `v5.18.2` |
| `openapiViewer.image.pullPolicy` | Container image pull policy | `IfNotPresent` |
| `openapiViewer.specUrl` | Target URL to OpenAPI spec (fetched client-side by browser). Use `http://localhost:8080/openapi.json` for `kubectl port-forward` or `/openapi.json` for shared Ingress | `http://vuhive-vuhive-cloud:8080/openapi.json` |
| `openapiViewer.service.type` | Kubernetes service type | `ClusterIP` |
| `openapiViewer.service.port` | Kubernetes service port | `8080` |
| `openapiViewer.ingress.enabled` | Enable Kubernetes Ingress for OpenAPI viewer | `false` |
| `openapiViewer.ingress.className` | Ingress class name | `""` |
| `openapiViewer.ingress.hosts[0].host` | Ingress host | `openapi.local` |
| `openapiViewer.ingress.hosts[0].paths[0].path` | Ingress path | `/` |

> **Note:** Default credentials are intended for local development only.
> Always override secrets in production using `existingSecret` references.
