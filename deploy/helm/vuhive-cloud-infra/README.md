# vuhive-cloud-infra

Helm chart to deploy backing infrastructure services (PostgreSQL and MinIO) for `vuhive-cloud`.

## Overview

This chart bundles backing infrastructure dependencies for the `vuhive-cloud` control plane:
- **`groundhog2k/postgres`**: Lightweight, secure PostgreSQL 16+ instance.
- **`minio/minio`**: S3-compatible standalone object storage server with pre-created buckets.
- **`keycloak`**: Containerized OIDC Identity Provider and authorization server with declarative realm import.
- **OpenAPI Viewer (`swaggerapi/swagger-ui`)**: Optional third-party interactive UI viewer for testing and exploring control plane APIs without bundling UI assets into the core Go control plane binary.

> [!WARNING]
> This chart is intended for **local development and evaluation** only (e.g., Rancher Desktop, Kind, Minikube). For production deployments, provision PostgreSQL, S3/Object Storage, and OIDC Identity Providers via managed cloud services or enterprise clusters, and reference them from the [`vuhive-cloud`](../vuhive-cloud/README.md) chart using `existingSecret` and `values-production.yaml`. See [External Infrastructure & Production Deployment Scenarios](../vuhive-cloud/README.md#5-external-infrastructure--production-deployment-scenarios) for architecture and deployment guidance.

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

The OpenAPI viewer runs with its own dedicated context root defaulting to `/docs` (`openapiViewer.contextPath: "/docs"`), passing `BASE_URL: "/docs"` to the Swagger UI container.

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

Then navigate to `http://localhost:8081/docs` in your desktop browser. Swagger UI serves under context root `/docs` and initiates a browser `fetch()` to `http://localhost:8080/openapi.json`.

Because modern web browsers enforce the Same-Origin Policy when fetching resources across different ports or hostnames, the `vuhive-cloud` control plane includes built-in Cross-Origin Resource Sharing (CORS) middleware and responds to HTTP `OPTIONS` preflight requests with `204 No Content` and standard CORS headers (`Access-Control-Allow-Origin: *`), ensuring seamless API exploration and ad-hoc request testing without browser blocks.

### MinIO (S3-Compatible Object Storage)

MinIO provides local S3-compatible object storage for test scenario archives, compiled binaries, and execution logs:
- **Dedicated Context Root**: MinIO Console is pre-configured with context root `/minio` via `CONSOLE_SUBPATH: "/minio"`. This guarantees that console web assets and API calls are scoped cleanly under `/minio`, avoiding collisions with root-level applications or other path-based routes. If automatic browser redirection from the S3 API port (9000) to the Console is desired, `MINIO_BROWSER_REDIRECT_URL` can be configured with a fully qualified URL including scheme (e.g. `http://vuhive.local/minio` or `http://localhost:9001/minio`).
- **S3 API Endpoint (Port `9000`)**: `http://vuhive-infra-minio:9000` — configured as `s3.endpoint` in `vuhive-cloud`. When exposed via Ingress, routes under path `/s3`.
- **MinIO Console / WebUI (Port `9001`)**:
  ```bash
  kubectl port-forward -n vuhive-system svc/vuhive-infra-minio 9001:9001
  ```
  Navigate to `http://localhost:9001/minio` (or `http://localhost:9001` which redirects automatically) and sign in with root credentials (`vuhive-dev` / `vuhive-dev-secret`).

### Keycloak (OIDC Identity Provider & Authorization Server)

Keycloak provides OIDC authentication and token issuance for the control plane and developer CLI:
- **Image**: `quay.io/keycloak/keycloak:26.1.0`
- **Dedicated Context Root (`/auth`)**: Keycloak is deployed with `KC_HTTP_RELATIVE_PATH: "/auth"` (`keycloak.httpRelativePath: "/auth"`). All realm discovery (`/auth/realms/vuhive/.well-known/openid-configuration`), token issuance (`/auth/realms/vuhive/protocol/openid-connect/token`), and admin console routes are scoped under `/auth`. Keycloak Quarkus liveness and readiness health probes execute independently on management port `9000` (`/health/live`, `/health/ready`).
- **Database Backend & Isolation**: Automatically connects to the in-chart PostgreSQL instance (`vuhive-infra-postgresql`) using a dedicated database (`keycloak`) provisioned during initial startup via PostgreSQL `customScripts` (`02-init-keycloak-db.sh`). This isolates Keycloak's 87 internal IAM tables entirely from the core control plane and BFF tables residing in the `vuhive` database's `public` schema. An optional dedicated schema can also be specified via `keycloak.database.schema`.
- **Declarative Realm Import**: Imports `files/vuhive-realm.json` defining the `vuhive` realm with **zero pre-created users**, standard roles (`vuhive-admin`, `vuhive-deployer`, `vuhive-developer`, `vuhive-viewer`, `vuhive-runner`), groups (`/administrators`, `/deployers`, `/developers`, `/viewers`), and clients (`vuhive-cloud-api`, `vuhive-cloud-cli`, `vuhive-runner`, `vuhive-cloud-bff`).
- **Backchannel Logout Resolution**: Pre-configured with backchannel logout targeting the canonical BFF service `http://vuhive-vuhive-cloud-bff:8081/api/v1/bff/auth/backchannel-logout`.
- **Accessing Keycloak Admin Console via Port-Forwarding**:
  ```bash
  kubectl port-forward -n vuhive-system svc/vuhive-infra-vuhive-cloud-infra-keycloak 8082:8080
  ```
  Navigate to `http://localhost:8082/auth/admin` and sign in with admin credentials (`admin` / `admin`).

## Dedicated Context Roots & Path-Based Ingress

Every backing service in `vuhive-cloud-infra` is configured with its own dedicated context root rather than binding to root `/`.

> [!NOTE]
> **No Ingress Manifests Bundled in Infra Chart**:
> The `vuhive-cloud-infra` chart intentionally does not bundle Kubernetes Ingress resources. Ingress controllers (e.g. Ingress NGINX, Traefik), hostnames, path rules, and TLS certificates are managed per environment. Thanks to dedicated context roots, all services can be cleanly mapped under a single unified domain without URL rewrites, strip-prefix annotations, or asset collisions:

| Component | Default Context Root | Target Service Name (Namespace: `vuhive-system`) | Target Service Port | Ingress Path (Prefix) | Example URL |
|---|---|---|---|---|---|
| **Control Plane API** (app chart) | `/api/v1` | `vuhive-vuhive-cloud` | `8080` | `/api/v1` | `http://vuhive.local/api/v1` |
| **Web UI & BFF** (app chart) | `/` & `/api/v1/bff` | `vuhive-vuhive-cloud-bff` | `8081` | `/` | `http://vuhive.local/` |
| **OpenAPI Viewer** | `/docs` | `vuhive-infra-vuhive-cloud-infra-openapi-viewer` | `8080` | `/docs` | `http://vuhive.local/docs` |
| **Keycloak IAM** | `/auth` | `vuhive-infra-vuhive-cloud-infra-keycloak` | `8080` | `/auth` | `http://vuhive.local/auth` |
| **MinIO Console** | `/minio` | `vuhive-infra-minio-console` | `9001` | `/minio` | `http://vuhive.local/minio` |
| **MinIO S3 API** | `/s3` | `vuhive-infra-minio` | `9000` | `/s3` | `http://vuhive.local/s3` |

### Environment Ingress Configuration Example

Below is a complete example of a Kubernetes Ingress manifest configured for an environment using a unified hostname (e.g., `vuhive.local`):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vuhive-unified-ingress
  namespace: vuhive-system
spec:
  rules:
    - host: vuhive.local
      http:
        paths:
          # 1. Control Plane REST API
          - path: /api/v1
            pathType: Prefix
            backend:
              service:
                name: vuhive-vuhive-cloud
                port:
                  number: 8080
          # 2. OpenAPI Swagger UI Viewer
          - path: /docs
            pathType: Prefix
            backend:
              service:
                name: vuhive-infra-vuhive-cloud-infra-openapi-viewer
                port:
                  number: 8080
          # 3. Keycloak IAM (OIDC Discovery & Admin Console)
          - path: /auth
            pathType: Prefix
            backend:
              service:
                name: vuhive-infra-vuhive-cloud-infra-keycloak
                port:
                  number: 8080
          # 4. MinIO Web Console
          - path: /minio
            pathType: Prefix
            backend:
              service:
                name: vuhive-infra-minio-console
                port:
                  number: 9001
          # 5. MinIO S3 API
          - path: /s3
            pathType: Prefix
            backend:
              service:
                name: vuhive-infra-minio
                port:
                  number: 9000
          # 6. Web UI & BFF Dashboard (Root Catch-All)
          - path: /
            pathType: Prefix
            backend:
              service:
                name: vuhive-vuhive-cloud-bff
                port:
                  number: 8081
```

## Configuration Parameters

| Parameter | Description | Default |
|---|---|---|
| `postgresql.settings.superuserPassword` | PostgreSQL superuser password | `vuhive-dev-root` |
| `postgresql.userDatabase.name` | Application database name | `vuhive` |
| `postgresql.userDatabase.user` | Application database user | `vuhive` |
| `postgresql.userDatabase.password` | Application database password | `vuhive-dev` |
| `postgresql.customScripts` | Custom PostgreSQL initialization scripts mounted into `/docker-entrypoint-initdb.d` | Provisions `keycloak` database via `02-init-keycloak-db.sh` |
| `minio.rootUser` | MinIO root user | `vuhive-dev` |
| `minio.rootPassword` | MinIO root password | `vuhive-dev-secret` |
| `minio.buckets[0].name` | Default artifact bucket name | `vuhive-artifacts` |
| `minio.buckets[0].policy` | Default artifact bucket policy | `none` |
| `minio.environment.CONSOLE_SUBPATH` | MinIO Console UI context subpath | `"/minio"` |
| `keycloak.enabled` | Deploy Keycloak OIDC identity provider | `true` |
| `keycloak.image.repository` | Container image repository for Keycloak | `quay.io/keycloak/keycloak` |
| `keycloak.image.tag` | Container image tag | `26.1.0` |
| `keycloak.adminUser` | Keycloak bootstrap administrator username | `admin` |
| `keycloak.adminPassword` | Keycloak bootstrap administrator password | `admin` |
| `keycloak.httpRelativePath` | Keycloak HTTP relative context path (`KC_HTTP_RELATIVE_PATH`) | `"/auth"` |
| `keycloak.database.host` | Keycloak PostgreSQL host (defaults to in-chart service) | `""` |
| `keycloak.database.port` | Keycloak PostgreSQL port | `5432` |
| `keycloak.database.name` | Isolated Keycloak database name | `keycloak` |
| `keycloak.database.user` | Keycloak database user | `vuhive` |
| `keycloak.database.password` | Keycloak database password | `vuhive-dev` |
| `keycloak.database.schema` | Optional Keycloak database schema (`KC_DB_SCHEMA`) | `""` |
| `keycloak.service.port` | Keycloak service port | `8080` |
| `openapiViewer.enabled` | Deploy optional third-party OpenAPI viewer (Swagger UI) | `false` |
| `openapiViewer.image.repository` | Container image repository for OpenAPI viewer | `swaggerapi/swagger-ui` |
| `openapiViewer.image.tag` | Container image tag | `v5.18.2` |
| `openapiViewer.image.pullPolicy` | Container image pull policy | `IfNotPresent` |
| `openapiViewer.contextPath` | Dedicated context root for Swagger UI (`BASE_URL`) | `"/docs"` |
| `openapiViewer.specUrl` | Target URL to OpenAPI spec (fetched client-side by browser). Use `http://localhost:8080/openapi.json` for `kubectl port-forward` or `/openapi.json` for shared Ingress | `http://vuhive-vuhive-cloud:8080/openapi.json` |
| `openapiViewer.service.type` | Kubernetes service type | `ClusterIP` |
| `openapiViewer.service.port` | Kubernetes service port | `8080` |

> **Note:** Default credentials are intended for local development only.
> Always override secrets in production using `existingSecret` references.

