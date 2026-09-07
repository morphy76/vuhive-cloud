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
- **BuildKit Disk Pressure & Kubelet ImageGC Pruning:**
  Iterative builds accumulate BuildKit layer cache in the VM disk. If disk usage crosses Kubelet's `ImageGCHighThresholdPercent` (80%-85%), Kubelet triggers continuous ImageGC (`ImageGCFailed`), sweeping unreferenced `--load` images within ~60 seconds and causing `ErrImageNeverPull` or `ErrImagePull`. Run `make docker-prune` (or `docker builder prune -f`) to reclaim disk space and resolve the eviction loop.

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
- **Callback URL (Issue #47 Resolved):** The chart automatically resolves `apiCallbackUrl` with `/api/v1/runs/complete` and mitigates `ndots:5` search leaks (using the unqualified service name when `runner.namespace` matches the release namespace). Explicit `--set apiCallbackUrl=...` is optional.
```bash
helm --kube-context rancher-desktop install vuhive deploy/helm/vuhive-cloud \
  --namespace "${SMOKE_NS}" \
  -f deploy/helm/vuhive-cloud/values-dev.yaml \
  --set runner.namespace="${SMOKE_NS}" \
  --set builder.namespace="${SMOKE_NS}" \
  --wait --timeout=120s
```

### Step 5: Condition & Readiness Awaiting
```bash
kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" \
  --for=condition=ready pod -l app.kubernetes.io/name=vuhive-cloud --timeout=120s
```

### Step 6: Endpoint Smoke Probing & File Staging Setup

Deploy an in-cluster probe pod to verify HTTP endpoints and stage test packages. Choose between two supported probe patterns:

#### Option A: Minimal Curl Probe (Default)
Uses `curlimages/curl:latest` for a minimal, lightweight probe footprint.
> [!WARNING]
> **No `tar` Binary in Minimal Curl Image**: `curlimages/curl:latest` does NOT contain `tar`. Attempting to use `kubectl cp` will fail with `command terminated with exit code 3` (`tar: not found`). When using Option A, files MUST be staged into the container using the **Base64 Stdin Pipeline** in Step 7.

```bash
kubectl --context rancher-desktop run curl-test -n "${SMOKE_NS}" --image=curlimages/curl:latest --restart=Never --command -- sleep 3600
kubectl --context rancher-desktop wait --for=condition=Ready pod/curl-test -n "${SMOKE_NS}" --timeout=60s

# Probe health and version
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i http://vuhive-vuhive-cloud:8080/healthz
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i http://vuhive-vuhive-cloud:8080/version
```

#### Option B: Alpine Utility Probe (Native `kubectl cp` Support)
If native `kubectl cp` support is desired without base64 piping, spin up an Alpine probe pre-loaded with both `curl` and `tar`:

```bash
kubectl --context rancher-desktop run curl-test -n "${SMOKE_NS}" --image=alpine:3.20 --restart=Never --command -- sh -c "apk add --no-cache curl tar && sleep 3600"
kubectl --context rancher-desktop wait --for=condition=Ready pod/curl-test -n "${SMOKE_NS}" --timeout=60s

# Probe health and version
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i http://vuhive-vuhive-cloud:8080/healthz
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i http://vuhive-vuhive-cloud:8080/version
```

### Step 7: End-to-End Build & Runner Job Verification

Execute full end-to-end verification following this sequential workflow:

#### 1. Create Active Test Suite & Runner Profile
Create a reusable runner profile via the control plane REST API:
```bash
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- curl -s -i -X POST http://vuhive-vuhive-cloud:8080/api/v1/profiles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "smoke-profile",
    "description": "Smoke test runner profile",
    "runner_image": "alpine:3.20",
    "cpu_request": "100m",
    "cpu_limit": "500m",
    "memory_request": "128Mi",
    "memory_limit": "512Mi"
  }'
```

Insert an `ACTIVE` test suite into the database:
```bash
kubectl --context rancher-desktop exec -i -n "${SMOKE_NS}" vuhive-infra-postgresql-0 -- psql -U vuhive -d vuhive -c \
  "INSERT INTO test_suites (id, name, description, status, created_at, updated_at) VALUES ('smoke-suite-01', 'Smoke Suite', 'Local validation test suite', 'ACTIVE', NOW(), NOW()) ON CONFLICT (id) DO NOTHING;"
```

#### 2. Build Subsystem Verification (File Staging & Ephemeral Compilation)

Package a minimal Go load test module locally:
```bash
mkdir -p /tmp/smoke-test-module
cat << 'EOF' > /tmp/smoke-test-module/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/morphy76/vuhive"
)

func main() {
	scenario := vuhive.NewScenario("Smoke Test").
		Step("Ping", func(ctx context.Context) error {
			resp, err := http.Get("http://vuhive-vuhive-cloud:8080/healthz")
			if err != nil || resp.StatusCode != http.StatusOK {
				return fmt.Errorf("ping failed: %w", err)
			}
			return nil
		})

	engine := vuhive.NewEngine(vuhive.EngineConfig{
		DefaultDuration: 5 * time.Second,
		DefaultVUs:      1,
	})

	if err := engine.Run(scenario); err != nil {
		panic(err)
	}
}
EOF

cat << 'EOF' > /tmp/smoke-test-module/go.mod
module smoke-test

go 1.26

require github.com/morphy76/vuhive v1.1.5
EOF

tar -czf /tmp/smoke-suite.tar.gz -C /tmp/smoke-test-module main.go go.mod
```

**Stage the archive into the probe container**:
- **Option A (Base64 Stdin Pipeline - Preferred for `curlimages/curl:latest`)**:
  Streams the archive over standard input and decodes it inside the probe without requiring `tar`:
  ```bash
  base64 < /tmp/smoke-suite.tar.gz | kubectl --context rancher-desktop exec -i -n "${SMOKE_NS}" curl-test -- sh -c 'base64 -d > /tmp/smoke-suite.tar.gz'
  ```
- **Option B (Native `kubectl cp` - When using Option B Alpine Probe)**:
  ```bash
  kubectl --context rancher-desktop cp /tmp/smoke-suite.tar.gz "${SMOKE_NS}"/curl-test:/tmp/smoke-suite.tar.gz
  ```

**Trigger asynchronous build compilation**:
```bash
# Target linux/arm64 for Apple Silicon Rancher Desktop, or linux/amd64 for x86_64
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- \
  curl -s -i -X POST "http://vuhive-vuhive-cloud:8080/api/v1/suites/smoke-suite-01/builds" \
  -F "source=@/tmp/smoke-suite.tar.gz" \
  -F "platform=linux/arm64"
```

**Await completion of ephemeral build job and assert artifact status**:
```bash
# Wait for the compilation job to complete
kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" \
  --for=condition=complete job -l app.kubernetes.io/name=vuhive-builder --timeout=120s

# Verify artifact is READY and checksum is populated
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- \
  curl -s "http://vuhive-vuhive-cloud:8080/api/v1/suites/smoke-suite-01/artifacts"
```

#### 3. Runner Job Completion Verification

Trigger an ad-hoc test run via REST API:
```bash
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- \
  curl -s -i -X POST "http://vuhive-vuhive-cloud:8080/api/v1/runs" \
  -H "Content-Type: application/json" \
  -d '{
    "suite_id": "smoke-suite-01",
    "profile_id": "smoke-profile"
  }'
```

Alternatively, create a test schedule to verify native Kubernetes `CronJob` management:
```bash
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- \
  curl -s -i -X POST "http://vuhive-vuhive-cloud:8080/api/v1/schedules" \
  -H "Content-Type: application/json" \
  -d '{
    "suite_id": "smoke-suite-01",
    "profile_id": "smoke-profile",
    "cron_expression": "0 0 31 2 *"
  }'

# Dispatch manual execution from CronJob
kubectl --context rancher-desktop create job test-runner-exec --from=cronjob/<cronjob-name> -n "${SMOKE_NS}"
```

**Wait for runner Job completion**:
```bash
kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" \
  --for=condition=complete job -l app.kubernetes.io/managed-by=vuhive-cloud --timeout=120s
```

**Verify exit status, log upload, and report upload**:
```bash
# Fetch latest runs and verify status is COMPLETED
kubectl --context rancher-desktop exec -n "${SMOKE_NS}" curl-test -- \
  curl -s "http://vuhive-vuhive-cloud:8080/api/v1/runs?suite_id=smoke-suite-01"
```

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

