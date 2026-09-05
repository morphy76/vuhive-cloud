# vuhive-cloud Adoption Guide & API Recipes (Cookbook)

Welcome to the `vuhive-cloud` adoption cookbook. This guide provides an end-to-end walkthrough for test engineers, DevOps specialists, and platform architects looking to build, schedule, execute, and monitor distributed load testing workloads on Kubernetes with `vuhive-cloud`.

---

## Table of Contents

- [1. Core Domain Concepts](#1-core-domain-concepts)
- [2. Authoring & Packaging Load Tests](#2-authoring--packaging-load-tests)
  - [A. Test Scenario Structure](#a-test-scenario-structure)
  - [B. Packaging Source Archives](#b-packaging-source-archives)
  - [C. Scenario Configuration (`vuhive.yaml`)](#c-scenario-configuration-vuhiveyaml)
- [3. Control Plane API Recipes](#3-control-plane-api-recipes)
  - [Recipe 1: Registering a Test Suite & Uploading Source Packages](#recipe-1-registering-a-test-suite--uploading-source-packages)
  - [Recipe 2: Monitoring Build Status & Inspecting Artifacts](#recipe-2-monitoring-build-status--inspecting-artifacts)
  - [Recipe 3: Defining & Managing Reusable Runner Profiles](#recipe-3-defining--managing-reusable-runner-profiles)
  - [Recipe 4: Managing Scheduled Test Runs with Kubernetes CronJobs](#recipe-4-managing-scheduled-test-runs-with-kubernetes-cronjobs)
  - [Recipe 5: Dispatching Ad-Hoc Test Executions & Job Lifecycle](#recipe-5-dispatching-ad-hoc-test-executions--job-lifecycle)
  - [Recipe 6: Reporting Run Completion, Ingesting KPIs & Querying Historical Runs](#recipe-6-reporting-run-completion-ingesting-kpis--querying-historical-runs)
  - [Recipe 7: Synchronizing Distributed Multi-Pod Runs with Start Barrier](#recipe-7-synchronizing-distributed-multi-pod-runs-with-start-barrier)
  - [Recipe 8: Execution Diagnostics, Log Inspection & Troubleshooting](#recipe-8-execution-diagnostics-log-inspection--troubleshooting)

---

## 1. Core Domain Concepts

`vuhive-cloud` models load testing workflows through clean domain entities:

- **`TestSuite`**: A logical collection of load test scenarios representing a service or application under test.
- **`Artifact`**: A compiled, self-contained Linux executable generated dynamically by ephemeral Kubernetes build jobs from Go test sources.
- **`RunnerProfile`**: A reusable resource and scheduling specification declaring CPU/memory requests and limits, node selectors, tolerations, and node affinity rules.
- **`Schedule`**: A recurring execution rule mapped 1-to-1 with a native Kubernetes `batch/v1` `CronJob`.
- **`TestRun`**: An individual test execution instance tracking lifecycle states (`QUEUED` $\to$ `RUNNING` $\to$ `COMPLETED` / `FAILED` / `ABORTED`), capturing exit codes, preserving logs, and indexing performance KPIs.
- **`BarrierSession`**: A distributed synchronization rendezvous point enabling multi-pod test workers to align and start load generation simultaneously.

---

## 2. Authoring & Packaging Load Tests

### A. Test Scenario Structure

`vuhive-cloud` executes Go test modules implementing load testing scenarios with the [`github.com/morphy76/vuhive`](https://github.com/morphy76/vuhive) engine (current release `v1.1.5`). A minimal test suite consists of a `go.mod` and a Go source file defining the workload:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/morphy76/vuhive"
)

func main() {
	client := &http.Client{Timeout: 5 * time.Second}

	scenario := vuhive.NewScenario("User Checkout Flow").
		Step("Homepage", func(ctx context.Context) error {
			resp, err := client.Get("http://target-service.default.svc.cluster.local/healthz")
			if err != nil || resp.StatusCode != http.StatusOK {
				return fmt.Errorf("homepage check failed: %w", err)
			}
			return nil
		})

	engine := vuhive.NewEngine(vuhive.EngineConfig{
		DefaultDuration: 30 * time.Second,
		DefaultVUs:      10,
	})

	if err := engine.Run(scenario); err != nil {
		panic(err)
	}
}
```

The accompanying `go.mod`:

```text
module my-load-test

go 1.26

require github.com/morphy76/vuhive v1.1.5
```

### B. Packaging Source Archives

Compress your test scenario into a standard `.tar.gz` archive before uploading:

```bash
tar -czvf test-suite.tar.gz main.go go.mod
```

### C. Scenario Configuration (`vuhive.yaml`)

You can supply an optional `vuhive.yaml` configuration file within the package or upload it to configure runtime parameters (iterations, ramp-up rate, threshold SLAs):

```yaml
version: "1.0"
execution:
  vus: 50
  duration: 60s
  ramp_up: 10s
thresholds:
  p95_latency_ms: 250
  error_rate_pct: 1.0
```

---

## 3. Control Plane API Recipes

All examples assume the control plane is reachable at `http://vuhive-cloud.vuhive-system.svc.cluster.local:8080` (or `http://localhost:8080` when port-forwarded).

### Recipe 1: Registering a Test Suite & Uploading Source Packages

Upload the source archive to trigger an asynchronous compilation build job in Kubernetes.

```bash
curl -i -X POST http://localhost:8080/api/v1/suites/suite-auth-checkout/builds \
  -F "source=@test-suite.tar.gz" \
  -F "platform=linux/amd64"
```

> [!TIP]
> You can target `linux/amd64` or `linux/arm64`. If `platform` is omitted or set to `all`, artifacts for both architectures will be scheduled for compilation.

#### Response (`202 Accepted`):

```json
{
  "message": "build triggered successfully",
  "artifacts": [
    {
      "id": "c7a6e118-20ab-48d6-953b-e01140026e61",
      "suite_id": "suite-auth-checkout",
      "platform": "linux/amd64",
      "status": "PENDING",
      "created_at": "2026-09-05T10:00:00Z"
    }
  ]
}
```

The control plane creates an ephemeral Kubernetes `batch/v1` `Job` running `golang:1.26-alpine` in the builder namespace (`vuhive-system`). The job compiles the Go source into a statically linked binary and uploads it to the configured S3 bucket.

---

### Recipe 2: Monitoring Build Status & Inspecting Artifacts

Poll the artifact registry endpoint to verify build progress and obtain the compiled binary S3 key and SHA256 checksum:

```bash
curl -s http://localhost:8080/api/v1/suites/suite-auth-checkout/artifacts | jq .
```

#### Response (`200 OK`):

```json
{
  "artifacts": [
    {
      "id": "c7a6e118-20ab-48d6-953b-e01140026e61",
      "suite_id": "suite-auth-checkout",
      "platform": "linux/amd64",
      "s3_binary_key": "artifacts/suites/suite-auth-checkout/linux-amd64-c7a6e118",
      "sha256_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "status": "READY",
      "created_at": "2026-09-05T10:00:00Z"
    }
  ],
  "count": 1
}
```

When `status` reaches `READY`, the artifact is available for execution by runner pods. If compilation encounters compiler errors or syntax violations, `status` becomes `FAILED` and `error_message` contains the diagnostic logs.

---

### Recipe 3: Defining & Managing Reusable Runner Profiles

Runner Profiles decouple test suite logic from cluster compute topology. A profile encapsulates resource constraints, node affinity, and tolerations.

#### 1. Create a Runner Profile:

```bash
curl -i -X POST http://localhost:8080/api/v1/profiles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "high-cpu-isolated-runners",
    "description": "Dedicated performance testing node pool profile",
    "runner_image": "alpine:3.20",
    "cpu_request": "2000m",
    "cpu_limit": "4000m",
    "memory_request": "4Gi",
    "memory_limit": "8Gi",
    "node_selector": {
      "node-role.kubernetes.io/performance-runner": "true"
    },
    "affinity": {
      "node_selector_terms": [
        {
          "key": "topology.kubernetes.io/zone",
          "operator": "In",
          "values": ["us-east-1a", "us-east-1b"]
        }
      ]
    },
    "tolerations": [
      {
        "key": "performance-tests-only",
        "operator": "Exists",
        "effect": "NoSchedule"
      }
    ]
  }'
```

#### Response (`201 Created`):

```json
{
  "id": "e8d665b1-2e67-4228-8ab6-79c5b248a31e",
  "name": "high-cpu-isolated-runners",
  "description": "Dedicated performance testing node pool profile",
  "runner_image": "alpine:3.20",
  "cpu_request": "2000m",
  "cpu_limit": "4000m",
  "memory_request": "4Gi",
  "memory_limit": "8Gi",
  "node_selector": {
    "node-role.kubernetes.io/performance-runner": "true"
  },
  "affinity": {
    "node_selector_terms": [
      {
        "key": "topology.kubernetes.io/zone",
        "operator": "In",
        "values": ["us-east-1a", "us-east-1b"]
      }
    ]
  },
  "tolerations": [
    {
      "key": "performance-tests-only",
      "operator": "Exists",
      "effect": "NoSchedule"
    }
  ],
  "created_at": "2026-09-05T10:05:00Z",
  "updated_at": "2026-09-05T10:05:00Z"
}
```

#### 2. List Profiles:

```bash
curl -s http://localhost:8080/api/v1/profiles | jq .
```

#### 3. Update an Existing Profile:

```bash
curl -i -X PUT http://localhost:8080/api/v1/profiles/e8d665b1-2e67-4228-8ab6-79c5b248a31e \
  -H "Content-Type: application/json" \
  -d '{
    "name": "high-cpu-isolated-runners",
    "cpu_request": "4000m",
    "cpu_limit": "8000m",
    "memory_request": "8Gi",
    "memory_limit": "16Gi"
  }'
```

#### 4. Delete a Profile:

```bash
curl -i -X DELETE http://localhost:8080/api/v1/profiles/e8d665b1-2e67-4228-8ab6-79c5b248a31e
```

---

### Recipe 4: Managing Scheduled Test Runs with Kubernetes CronJobs

`vuhive-cloud` provides native Kubernetes CronJob orchestration. Creating a schedule registers a persistent record in PostgreSQL and instantiates a `batch/v1` `CronJob` in the runner namespace.

#### 1. Create a Scheduled Execution:

```bash
curl -i -X POST http://localhost:8080/api/v1/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "suite_id": "suite-auth-checkout",
    "artifact_id": "c7a6e118-20ab-48d6-953b-e01140026e61",
    "runner_profile_id": "e8d665b1-2e67-4228-8ab6-79c5b248a31e",
    "name": "nightly-checkout-benchmark",
    "cron_expression": "0 2 * * *"
  }'
```

#### Response (`201 Created`):

```json
{
  "id": "7fa1205c-d38e-4f51-b924-11883395bcf8",
  "suite_id": "suite-auth-checkout",
  "artifact_id": "c7a6e118-20ab-48d6-953b-e01140026e61",
  "runner_profile_id": "e8d665b1-2e67-4228-8ab6-79c5b248a31e",
  "name": "nightly-checkout-benchmark",
  "cron_expression": "0 2 * * *",
  "k8s_cronjob_name": "vuhive-sched-7fa1205c",
  "is_active": true,
  "created_at": "2026-09-05T10:10:00Z",
  "updated_at": "2026-09-05T10:10:00Z"
}
```

Verify the native Kubernetes CronJob:

```bash
kubectl get cronjob -n vuhive-runners vuhive-sched-7fa1205c
```

#### 2. Update Schedule Cadence:

```bash
curl -i -X PUT http://localhost:8080/api/v1/schedules/7fa1205c-d38e-4f51-b924-11883395bcf8 \
  -H "Content-Type: application/json" \
  -d '{
    "cron_expression": "*/30 * * * *"
  }'
```

#### 3. Delete a Schedule:

```bash
curl -i -X DELETE http://localhost:8080/api/v1/schedules/7fa1205c-d38e-4f51-b924-11883395bcf8
```
Deletes both the database aggregate and the underlying Kubernetes `CronJob`.

---

### Recipe 5: Dispatching Ad-Hoc Test Executions & Job Lifecycle

#### 1. Dispatching from a Configured Schedule Template:

To run an immediate ad-hoc test execution using the pre-configured runner profile, artifact, and environment from a Schedule, instantiate a Job from the CronJob:

```bash
kubectl create job nightly-adhoc-manual-1 \
  --from=cronjob/vuhive-sched-7fa1205c \
  -n vuhive-runners
```

The control plane Informer Watcher (`RunnerJobWatcher`) detects the newly spawned Job, inspects its owner references and labels, registers a correlated `TestRun` entity in `QUEUED` / `RUNNING` status, and tracks its execution lifecycle.

#### 2. Pod Lifecycle & Security Architecture:

When the runner Job spawns:
```text
┌────────────────────────────────────────────────────────────────────────┐
│ Kubernetes Runner Pod (Restricted Pod Security Standard)               │
│                                                                        │
│ 1. Init Container: runner-init                                         │
│    - Downloads target binary and vuhive.yaml from S3                  │
│    - Populates /shared emptyDir with runner executable & wrapper       │
│                                                                        │
│ 2. Main Container: runner (e.g. alpine:3.20)                           │
│    - Runs non-root (UID 10001), readOnlyRootFilesystem                 │
│    - Executes runner-wrapper in /shared                                │
│    - Captures stdout/stderr into /shared/run.log                       │
│    - Generates deterministic /shared/summary.json                      │
│                                                                        │
│ 3. Wrapper Finalization:                                               │
│    - Uploads run.log & summary.json to S3                             │
│    - Invokes POST /api/v1/runs/{id}/complete callback                 │
└────────────────────────────────────────────────────────────────────────┘
```

Monitor execution progress in Kubernetes:

```bash
kubectl wait --namespace vuhive-runners \
  --for=condition=complete job/nightly-adhoc-manual-1 \
  --timeout=180s
```

---

### Recipe 6: Reporting Run Completion, Ingesting KPIs & Querying Historical Runs

Upon workload completion, the runner wrapper uploads execution artifacts to S3 and notifies the control plane callback endpoint. This endpoint can also be invoked directly by custom CI/CD pipelines:

```bash
curl -i -X POST http://localhost:8080/api/v1/runs/98bc19d4-1a3b-4882-a982-ff012498beaa/complete \
  -H "Content-Type: application/json" \
  -d '{
    "exit_code": 0,
    "report_key": "runs/98bc19d4/summary.json",
    "logs_key": "runs/98bc19d4/run.log",
    "finished_at": "2026-09-05T10:15:30Z",
    "summary": {
      "total_iterations": 25000,
      "total_requests": 100000,
      "avg_tps": 1666.67,
      "p50_duration_ms": 12.4,
      "p90_duration_ms": 28.1,
      "p95_duration_ms": 45.2,
      "p99_duration_ms": 89.6,
      "error_rate_pct": 0.02,
      "status": "PASS"
    }
  }'
```

#### Response (`200 OK`):

```json
{
  "id": "98bc19d4-1a3b-4882-a982-ff012498beaa",
  "suite_id": "suite-auth-checkout",
  "artifact_id": "c7a6e118-20ab-48d6-953b-e01140026e61",
  "runner_profile_id": "e8d665b1-2e67-4228-8ab6-79c5b248a31e",
  "status": "COMPLETED",
  "exit_code": 0,
  "sla_passed": true,
  "metrics": {
    "total_iterations": 25000,
    "total_requests": 100000,
    "avg_tps": 1666.67,
    "p50_duration_ms": 12.4,
    "p90_duration_ms": 28.1,
    "p95_duration_ms": 45.2,
    "p99_duration_ms": 89.6,
    "error_rate_pct": 0.02
  },
  "s3_report_key": "runs/98bc19d4/summary.json",
  "s3_logs_key": "runs/98bc19d4/run.log",
  "started_at": "2026-09-05T10:14:30Z",
  "finished_at": "2026-09-05T10:15:30Z",
  "created_at": "2026-09-05T10:14:25Z"
}
```

Performance KPIs (`p50`, `p90`, `p95`, `p99`, `avg_tps`, `error_rate_pct`) are automatically indexed in PostgreSQL for querying, SLA threshold assertions, and historical trend analysis.

#### 2. Listing & Filtering Historical Test Runs:

Query runs across suites, statuses, schedules, or time ranges with pagination:

```bash
curl -s "http://localhost:8080/api/v1/runs?suite_id=suite-auth-checkout&status=COMPLETED&limit=10" | jq .
```

##### Response (`200 OK`):

```json
{
  "runs": [
    {
      "id": "98bc19d4-1a3b-4882-a982-ff012498beaa",
      "suite_id": "suite-auth-checkout",
      "artifact_id": "c7a6e118-20ab-48d6-953b-e01140026e61",
      "runner_profile_id": "e8d665b1-2e67-4228-8ab6-79c5b248a31e",
      "status": "COMPLETED",
      "started_at": "2026-09-05T10:14:30Z",
      "finished_at": "2026-09-05T10:15:30Z",
      "duration_ms": 60000,
      "exit_code": 0,
      "sla_passed": true,
      "metrics": {
        "total_iterations": 25000,
        "total_requests": 100000,
        "avg_tps": 1666.67,
        "p50_duration_ms": 12.4,
        "p90_duration_ms": 28.1,
        "p95_duration_ms": 45.2,
        "p99_duration_ms": 89.6,
        "error_rate_pct": 0.02
      },
      "s3_report_key": "runs/98bc19d4/summary.json",
      "s3_logs_key": "runs/98bc19d4/run.log",
      "created_at": "2026-09-05T10:14:25Z"
    }
  ],
  "count": 1,
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

#### 3. Inspecting Run Details & Indexed Performance KPIs:

Retrieve a single test run by its UUID:

```bash
curl -s http://localhost:8080/api/v1/runs/98bc19d4-1a3b-4882-a982-ff012498beaa | jq .
```


---

### Recipe 7: Synchronizing Distributed Multi-Pod Runs with Start Barrier

When distributing a massive load test across multiple worker pods, all workers must begin generating traffic at the exact same instant to accurately benchmark system concurrency and spike absorption. `vuhive-cloud` includes a built-in start barrier rendezvous coordinator.

#### 1. Worker Pods Await Rendezvous:

Each participating worker calls `/barrier/await` declaring its worker ID and total expected worker count:

```bash
# Executed by Worker 1
curl -s -X POST http://localhost:8080/api/v1/runs/dist-run-001/barrier/await \
  -H "Content-Type: application/json" \
  -d '{
    "worker_id": "worker-pod-0",
    "total_workers": 2,
    "timeout_ms": 30000,
    "release_delay_ms": 5000
  }'
```

```bash
# Executed by Worker 2
curl -s -X POST http://localhost:8080/api/v1/runs/dist-run-001/barrier/await \
  -H "Content-Type: application/json" \
  -d '{
    "worker_id": "worker-pod-1",
    "total_workers": 2,
    "timeout_ms": 30000,
    "release_delay_ms": 5000
  }'
```

#### Response (`200 OK` to both workers once the barrier releases):

```json
{
  "run_id": "dist-run-001",
  "status": "RELEASED",
  "total_workers": 2,
  "ready_workers": 2,
  "target_start_time": "2026-09-05T10:20:05.123456Z",
  "start_in_ms": 4982,
  "participants": [
    {
      "worker_id": "worker-pod-0",
      "status": "READY",
      "joined_at": "2026-09-05T10:20:00.100Z",
      "ready_at": "2026-09-05T10:20:00.123Z"
    },
    {
      "worker_id": "worker-pod-1",
      "status": "READY",
      "joined_at": "2026-09-05T10:20:00.105Z",
      "ready_at": "2026-09-05T10:20:00.123Z"
    }
  ]
}
```

Both workers receive a synchronized `target_start_time` and countdown delay (`start_in_ms`), releasing simultaneously without clock drift.

#### 2. Querying Barrier Status:

Inspect live rendezvous state:

```bash
curl -s http://localhost:8080/api/v1/runs/dist-run-001/barrier | jq .
```

#### 3. Aborting a Barrier Rendezvous:

If a worker encounters an unrecoverable initialization error prior to start, it can signal an immediate abort to release other waiting workers gracefully:

```bash
curl -i -X POST http://localhost:8080/api/v1/runs/dist-run-001/barrier/abort \
  -H "Content-Type: application/json" \
  -d '{
    "worker_id": "worker-pod-1",
    "reason": "Failed pre-allocating network socket pool"
  }'
```

---

### Recipe 8: Execution Diagnostics, Log Inspection & Troubleshooting

#### 1. Fetching Execution Logs & Reports via REST API:

Developers and CI/CD pipelines can retrieve logs and summary reports directly from the control plane without configuring cloud storage credentials or CLI tools:

##### Stream Raw Execution Logs:

```bash
curl -s http://localhost:8080/api/v1/runs/98bc19d4-1a3b-4882-a982-ff012498beaa/logs
```

Or generate a secure presigned S3 direct-download URL (valid for 15 minutes):

```bash
curl -s "http://localhost:8080/api/v1/runs/98bc19d4-1a3b-4882-a982-ff012498beaa/logs?presign=true" | jq .
```

##### Download Full Raw Summary Report (`summary.json`):

```bash
curl -s http://localhost:8080/api/v1/runs/98bc19d4-1a3b-4882-a982-ff012498beaa/report | jq .
```

Or obtain a presigned download URL:

```bash
curl -s "http://localhost:8080/api/v1/runs/98bc19d4-1a3b-4882-a982-ff012498beaa/report?presign=true" | jq .
```

#### 2. Alternative: Direct S3 / MinIO Object Storage Download:

For cluster administrators with object storage credentials, tools like the AWS CLI or MinIO client (`mc`) can access artifacts directly:

```bash
# Configure AWS CLI for local MinIO
export AWS_ACCESS_KEY_ID=vuhive-dev
export AWS_SECRET_ACCESS_KEY=vuhive-dev-secret
export AWS_ENDPOINT_URL=http://localhost:9000

# Download execution log
aws --endpoint-url=http://localhost:9000 s3 cp \
  s3://vuhive-artifacts/runs/98bc19d4/run.log ./run.log

# Download complete summary report
aws --endpoint-url=http://localhost:9000 s3 cp \
  s3://vuhive-artifacts/runs/98bc19d4/summary.json ./summary.json
```

#### 3. Inspecting Runner Pod Status via `kubectl`:

```bash
# List runner pods
kubectl get pods -n vuhive-runners -l app.kubernetes.io/managed-by=vuhive-cloud

# Check init container logs (artifact download & setup)
kubectl logs -n vuhive-runners pod/<pod-name> -c runner-init

# Check main runner container execution logs
kubectl logs -n vuhive-runners pod/<pod-name> -c runner
```

#### 4. Common Error Handling Reference:

| HTTP Status | Error String | Cause | Resolution |
|---|---|---|---|
| `400 Bad Request` | `invalid request payload: ...` | Malformed JSON or missing required fields. | Validate request body against API schema. |
| `404 Not Found` | `suite not found` or `artifact not found` | The requested UUID does not exist. | Verify IDs via `GET /api/v1/suites/{id}/artifacts`. |
| `404 Not Found` | `test run not found`, `report not found`, `logs not found` | The run ID does not exist or artifacts have not been uploaded to S3. | Verify run ID and verify run status is `COMPLETED` or `FAILED`. |
| `409 Conflict` | `build job already running` | A compilation job is already active for this suite/platform. | Await completion or check build job status in builder namespace. |
| `409 Conflict` | `test run is still in progress` | Report or logs queried while the runner pod is still running. | Await run completion (`COMPLETED` or `FAILED`) before fetching reports/logs. |
| `422 Unprocessable Entity` | `unsupported target platform` | Platform is not `linux/amd64` or `linux/arm64`. | Specify valid platform architecture. |

---

## 4. Next Steps

- **[OpenAPI 3.0.3 Specification](../api/openapi.yaml)**: Complete REST API contract, interactive endpoints, and request/response schemas.
- **[Main Project README](../README.md)**: System overview, architecture diagram, and repository roadmap.
- **[vuhive-cloud Helm Chart](../deploy/helm/vuhive-cloud/README.md)**: Production deployment instructions and configuration parameter reference.
- **[vuhive-cloud-infra Helm Chart](../deploy/helm/vuhive-cloud-infra/README.md)**: Local backing services guide (PostgreSQL + MinIO).
- **[Architecture Specification](../ARCHITECTURE_SPEC.md)**: Complete internal hexagonal architecture, DDL schemas, and domain models.
