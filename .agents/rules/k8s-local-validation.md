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

- **CLI Auto-Detection:** Detect whether Rancher Desktop is configured with `dockerd` (Moby) or `containerd`:
  - If `docker info >/dev/null 2>&1` succeeds, build using `docker build`:
    ```bash
    docker build -t vuhive/server:local -f deploy/docker/server.Dockerfile .
    ```
  - If using `nerdctl` (containerd runtime), build targeting the Kubernetes namespace (`k8s.io`):
    ```bash
    nerdctl --namespace k8s.io build -t vuhive/server:local -f deploy/docker/server.Dockerfile .
    ```
- **Image Pull Policy:** Always set `imagePullPolicy: IfNotPresent` or `imagePullPolicy: Never` in Pod and Job manifests to guarantee Rancher Desktop uses the locally built daemon image without attempting to pull from external registries.

## 5. Deployment & Health Verification Workflow

Execute validation following this sequential workflow:

1. **Namespace Setup:** Create `${SMOKE_NS}` with standard labels (`app.kubernetes.io/managed-by: vuhive-agent-smoke`).
2. **Infra Bootstrap:** Deploy required dependencies (Postgres, MinIO) into `${SMOKE_NS}` via Helm:
   ```bash
   helm --kube-context rancher-desktop install smoke-pg oci://registry-1.docker.io/bitnamicharts/postgresql \
     --namespace "${SMOKE_NS}" \
     --set auth.database=vuhive \
     --set auth.username=vuhive \
     --set auth.password=secretpassword \
     --wait --timeout=120s
   ```
3. **Workload Deployment:** Deploy control plane and dispatch test runner `Job`s into `${SMOKE_NS}`.
4. **Condition & Readiness Awaiting:**
   ```bash
   kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" \
     --for=condition=ready pod -l app=vuhive-server --timeout=120s
   ```
5. **Endpoint Smoke Probing:** Use `kubectl port-forward` or temporary in-cluster curl pods to verify HTTP endpoints (`/healthz`, `/version`).
6. **Job Completion Verification:** Wait for runner batch jobs to reach complete condition and verify exit status:
   ```bash
   kubectl --context rancher-desktop wait --namespace "${SMOKE_NS}" \
     --for=condition=complete job/<job-name> --timeout=120s
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
   kubectl --context rancher-desktop logs -n "${SMOKE_NS}" -l app=vuhive-server --all-containers=true --tail=200
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
| **Job** | Runner Wrapper | `batch/v1` Job completion | PASS / FAIL | Exit code 0, 100 iterations |
| **Cleanup**| Namespace | Teardown `${SMOKE_NS}` | SUCCESS | Namespace purged |
