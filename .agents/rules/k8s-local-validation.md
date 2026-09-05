---
trigger: always_on
description: Guidelines for on-demand local Kubernetes and Helm validation using Rancher Desktop ('rancher-desktop' context), ephemeral namespaces, local image builds, and diagnostic teardowns.
---

# Local Kubernetes Validation Guidelines (Rancher Desktop)

This rule governs how the agent performs local Kubernetes and Helm validations on behalf of the developer using a local Rancher Desktop cluster.

## 1. Trigger Policy & Operational Scope

- **On-Demand Execution:** Execute cluster and Helm validations ONLY when explicitly requested by the user (e.g., *"validate on k8s"*, *"smoke test on rancher desktop"*, *"test deployment with helm"*). Do NOT run Kubernetes commands during routine unit tests (`make test`) or standard code editing workflows.
- **Validation Scope:**
  - **Infrastructure Dependencies:** Ephemeral instances of PostgreSQL and MinIO/S3 deployed via Helm charts or lightweight manifests.
  - **Application Workloads:** `vuhive` control plane server (`cmd/server`), runner wrapper (`cmd/runner-wrapper`), and generated `batch/v1` `Job`s.

## 2. Cluster Context Safety & Guardrails

To prevent accidental mutations against remote, staging, or production clusters, the agent must strictly lock execution to the `rancher-desktop` context:

### A. Pre-Execution Context Verification & Auto-Switch
Before executing any `kubectl` or `helm` command, verify the active context:
```bash
CURRENT_CTX=$(kubectl config current-context 2>/dev/null || echo "none")
if [ "$CURRENT_CTX" != "rancher-desktop" ]; then
  kubectl config use-context rancher-desktop
fi
```

### B. Double-Lock Command Flags
Never rely solely on ambient kubeconfig settings. Explicitly append the context flag to EVERY command:
- For `kubectl`: `--context rancher-desktop`
  ```bash
  kubectl --context rancher-desktop get nodes
  ```
- For `helm`: `--kube-context rancher-desktop`
  ```bash
  helm --kube-context rancher-desktop list -A
  ```

## 3. Namespace Strategy & Resource Lifecycle

To maintain clean local state and prevent test collisions:

- **Isolated Ephemeral Namespaces:** Generate a unique, timestamped namespace for each validation session:
  ```bash
  SMOKE_NS="vuhive-smoke-$(date +%s)"
  kubectl --context rancher-desktop create namespace "${SMOKE_NS}"
  ```
- **Strict Teardown Contract:** Always tear down all deployed resources and delete the ephemeral namespace at the end of the validation session.
  ```bash
  kubectl --context rancher-desktop delete namespace "${SMOKE_NS}" --ignore-not-found=true --wait=false
  ```

## 4. Local Container Image Management

When testing local code changes against Rancher Desktop:

- **CLI Auto-Detection & BuildKit `--load` Requirement:**
  Modern Rancher Desktop with Docker 29+ utilizes the containerd snapshotter (`io.containerd.snapshotter.v1`) and BuildKit by default. Running a standard `docker build` stores multi-platform/attestation manifests in the Buildx cache without exporting them to the local CRI image store, which causes `ImagePullBackOff` in Kubernetes (`pull access denied`).
  - **Docker (Moby):** Always build with `--load --provenance=false`:
    ```bash
    docker build --load --provenance=false -t vuhive/server:local -f deploy/docker/server.Dockerfile .
    docker build --load --provenance=false -t vuhive/runner-init:local -f deploy/docker/runner-init.Dockerfile .
    ```
  - **containerd (nerdctl):** Build targeting the Kubernetes namespace (`k8s.io`):
    ```bash
    nerdctl --namespace k8s.io build -t vuhive/server:local -f deploy/docker/server.Dockerfile .
    nerdctl --namespace k8s.io build -t vuhive/runner-init:local -f deploy/docker/runner-init.Dockerfile .
    ```
- **Image Pull Policy:** Always set `imagePullPolicy: IfNotPresent` or `imagePullPolicy: Never` in Pod and Job manifests to guarantee Rancher Desktop uses the locally built daemon image without attempting to pull from external registries.

## 5. Deployment & Health Verification Workflow

Execute validation following this sequential workflow:

### Step 1: Ephemeral Namespace Setup
```bash
SMOKE_NS="vuhive-smoke-$(date +%s)"
kubectl --context rancher-desktop create namespace "${SMOKE_NS}"
kubectl --context rancher-desktop label namespace "${SMOKE_NS}" app.kubernetes.io/managed-by=vuhive-agent-smoke
```

### Step 2: Infra Bootstrap (`deploy/helm/vuhive-cloud-infra`)
Deploy the in-repo infrastructure chart providing PostgreSQL and standalone MinIO:
```bash
helm --kube-context rancher-desktop install vuhive-infra deploy/helm/vuhive-cloud-infra \
  --namespace "${SMOKE_NS}" \
  --wait --timeout=180s
```

### Step 3: Database Schema Migrations
If the server or chart does not yet execute automated startup migrations (Issue #41), apply the Goose Up schema migration directly into PostgreSQL before launching the application:
```bash
sed -n '1,/-- +goose Down/p' internal/adapters/outbound/postgres/migrations/000001_init_schema.sql \
  | grep -v -- '-- +goose' \
  | kubectl --context rancher-desktop exec -i -n "${SMOKE_NS}" vuhive-infra-postgresql-0 -- psql -U vuhive -d vuhive
```

### Step 4: Workload Deployment (`deploy/helm/vuhive-cloud`)
Deploy the control plane with local images and namespace overrides:
- **Scoped RBAC Caveat:** Until Issue #43 is resolved, avoid defaulting `runner.namespace` / `builder.namespace` to external non-existent namespaces (`vuhive-runners`, `vuhive-system`). Explicitly point them to `${SMOKE_NS}` or set `rbac.clusterScoped=true`.
- **Callback URL Caveat:** To avoid DNS `ndots:5` search domain leaks and ensure proper path routing (Issue #47), explicitly set `apiCallbackUrl` to `http://vuhive-vuhive-cloud:8080/api/v1/runs/complete`:
```bash
helm --kube-context rancher-desktop install vuhive deploy/helm/vuhive-cloud \
  --namespace "${SMOKE_NS}" \
  --set image.repository=vuhive/server \
  --set image.tag=local \
  --set runner.namespace="${SMOKE_NS}" \
  --set builder.namespace="${SMOKE_NS}" \
  --set runner.initImage=vuhive/runner-init:local \
  --set apiCallbackUrl="http://vuhive-vuhive-cloud:8080/api/v1/runs/complete" \
  --wait --timeout=120s
```

### Step 5: Condition & Readiness Awaiting
```bash
kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" \
  --for=condition=ready pod -l app.kubernetes.io/name=vuhive-cloud --timeout=120s
```

### Step 6: Endpoint Smoke Probing
Deploy an in-cluster curl probe pod to verify HTTP endpoints:
```bash
kubectl --context rancher-desktop run curl-test -n "${SMOKE_NS}" --image=curlimages/curl:latest --restart=Never --command -- sleep 3600
kubectl --context rancher-desktop wait --for=condition=Ready pod/curl-test -n "${SMOKE_NS}" --timeout=60s

# Probe health and version
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i http://vuhive-vuhive-cloud:8080/healthz
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i http://vuhive-vuhive-cloud:8080/version
```

### Step 7: End-to-End Build & Runner Job Verification
1. **Create Active Test Suite & Runner Profile:**
   - Verify `POST /api/v1/profiles` creates a runner profile.
   - Insert an `ACTIVE` test suite into the database.
2. **Build Subsystem Verification:**
   - Package a valid Go test module (with `go.mod` and `main.go`).
   - POST to `http://vuhive-vuhive-cloud:8080/api/v1/suites/{id}/builds` with `platform=linux/arm64`.
   - Await completion of build job (`app.kubernetes.io/name=vuhive-builder`) and assert artifact status is `READY`.
3. **Runner Job Completion Verification:**
   - Create a test schedule (`POST /api/v1/schedules`) to instantiate a native Kubernetes `CronJob`.
   - Dispatch an ad-hoc Job:
     ```bash
     kubectl --context rancher-desktop create job test-runner-exec --from=cronjob/<cronjob-name> -n "${SMOKE_NS}"
     kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" --for=condition=complete job/test-runner-exec --timeout=120s
     ```
   - Verify exit status, log upload, and report upload in MinIO.

## 6. Failure Diagnostics Protocol (Dump Before Teardown)

If any pod fails, enters `CrashLoopBackOff`, or health checks time out:

1. **Pod State & Event Capture:** Inspect the full namespace status:
   ```bash
   kubectl --context rancher-desktop get pods -n "${SMOKE_NS}" -o wide
   kubectl --context rancher-desktop get events -n "${SMOKE_NS}" --sort-by='.metadata.creationTimestamp'
   ```
2. **Pod Inspection & Logs Dump:** Capture details and logs (including previous crashed containers):
   ```bash
   kubectl --context rancher-desktop describe pods -n "${SMOKE_NS}"
   kubectl --context rancher-desktop logs -n "${SMOKE_NS}" -l app.kubernetes.io/name=vuhive-cloud --all-containers=true --tail=200
   ```
3. **Structured Failure Report:** Emit all diagnostic dumps in the test summary before initiating cleanup.
4. **Guaranteed Teardown:** Proceed with ephemeral namespace deletion so no orphaned resources remain in Rancher Desktop.

## 7. Verification Summary Format

Upon completing any local cluster validation, the agent must output a structured summary table in the response:

| Phase | Component | Action / Check | Result | Details / Output |
| :--- | :--- | :--- | :--- | :--- |
| **Infra** | PostgreSQL | Helm install & readiness | PASS / FAIL | Pod ready in 25s |
| **Infra** | MinIO / S3 | Helm install & bucket init | PASS / FAIL | Buckets created |
| **App** | Control Plane | Deployment & `/healthz` probe | PASS / FAIL | HTTP 200 OK |
| **Build**| Build Subsystem | Ephemeral K8s compilation | PASS / FAIL | Binary compiled & uploaded |
| **Job**  | Runner Wrapper | `batch/v1` Job completion | PASS / FAIL | Exit code 0, report uploaded |
| **Watcher**| Informer Watcher | Status reconciliation | PASS / FAIL | TestRun updated to COMPLETED |
| **Cleanup**| Namespace | Teardown `${SMOKE_NS}` | SUCCESS | Namespace purged |

