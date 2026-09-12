import { RouteId } from '@/types/navigation'

export type RecipeId =
  | 'recipe-1'
  | 'recipe-2'
  | 'recipe-3'
  | 'recipe-4'
  | 'recipe-5'
  | 'recipe-6'
  | 'recipe-7'

export interface RecipeStep {
  id: string
  title: string
  description: string
  endpoint: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  generateCurl: (params: Record<string, string>) => string
}

export interface ParamField {
  key: string
  label: string
  placeholder: string
  defaultValue: string
  description?: string
}

export interface RecipeDefinition {
  id: RecipeId
  title: string
  subtitle: string
  description: string
  cookbookRef: string
  contextRoute: RouteId | 'builds' | 'profiles' | 'results' | 'active-run'
  paramFields: ParamField[]
  defaultParams: Record<string, string>
  steps: RecipeStep[]
}

export const RECIPES: RecipeDefinition[] = [
  {
    id: 'recipe-1',
    title: 'Recipe 1: Registering a Test Suite & Uploading Source Packages',
    subtitle: 'Suites Page Context',
    description:
      'Registers a new test suite aggregate in DRAFT state, attaches scenario YAML configuration, and uploads the Go source package archive to trigger ephemeral cross-compilation in Kubernetes.',
    cookbookRef: 'Recipe 1',
    contextRoute: 'suites',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'suiteName',
        label: 'Suite Name',
        placeholder: 'suite-e2e-checkout',
        defaultValue: 'suite-e2e-checkout',
      },
      {
        key: 'suiteDesc',
        label: 'Suite Description',
        placeholder: 'Checkout service end-to-end load testing suite',
        defaultValue: 'Checkout service end-to-end load testing suite',
      },
      {
        key: 'suiteId',
        label: 'Suite ID (for configs & builds)',
        placeholder: '3e04a02e-bf34-4398-8b40-6389bca12c97',
        defaultValue: '3e04a02e-bf34-4398-8b40-6389bca12c97',
      },
      {
        key: 'platform',
        label: 'Target Platform Architecture',
        placeholder: 'linux/amd64',
        defaultValue: 'linux/amd64',
      },
      {
        key: 'sourceFile',
        label: 'Source Archive Path',
        placeholder: 'test-suite.tar.gz',
        defaultValue: 'test-suite.tar.gz',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      suiteName: 'suite-e2e-checkout',
      suiteDesc: 'Checkout service end-to-end load testing suite',
      suiteId: '3e04a02e-bf34-4398-8b40-6389bca12c97',
      platform: 'linux/amd64',
      sourceFile: 'test-suite.tar.gz',
    },
    steps: [
      {
        id: 'step-create-suite',
        title: 'Step 1: Create New Test Suite',
        description: 'Initialize a managed test suite aggregate in DRAFT state.',
        endpoint: '/api/v1/suites',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/suites \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "name": "${p.suiteName || 'suite-e2e-checkout'}",\n    "description": "${p.suiteDesc || 'Load testing suite'}"\n  }'`,
      },
      {
        id: 'step-attach-config',
        title: 'Step 2: Attach Scenario Configuration (vuhive.yaml)',
        description:
          'Upload execution parameters (VUs, duration, ramp-up stages, latency SLA thresholds).',
        endpoint: '/api/v1/suites/{id}/configs',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/suites/${p.suiteId || '3e04a02e-bf34-4398-8b40-6389bca12c97'}/configs \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "name": "staging-load",\n    "content_yaml": "version: \\"1.0\\"\\ndefault_scenario: standard_load\\nscenarios:\\n  standard_load:\\n    type: constant_vus\\n    vus: 50\\n    ramp_up: 10s\\n    run_period: 60s\\n    ramp_down: 5s\\n    vu_timeout: 5s\\n    thresholds:\\n      - metric: vuhive.http.req_duration\\n        stat: p95\\n        operator: \\"<\\"\\n        target: \\"250ms\\"\\n      - metric: vuhive.http.req_failed\\n        stat: rate\\n        operator: \\"<=\\"\\n        target: \\"0.01\\"\\n",\n    "is_default": true\n  }'`,
      },
      {
        id: 'step-upload-source',
        title: 'Step 3: Upload Source Archive & Trigger Compilation',
        description:
          'Upload Go source archive (.tar.gz, .tar.bz2, or .zip) to perform AST static validation and dispatch an ephemeral Kubernetes build Job.',
        endpoint: '/api/v1/suites/{id}/builds',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/suites/${p.suiteId || '3e04a02e-bf34-4398-8b40-6389bca12c97'}/builds \\\n  -F "source=@${p.sourceFile || 'test-suite.tar.gz'}" \\\n  -F "platform=${p.platform || 'linux/amd64'}"`,
      },
    ],
  },
  {
    id: 'recipe-2',
    title: 'Recipe 2: Monitoring Build Status & Inspecting Logs',
    subtitle: 'Builds View Context',
    description:
      'Polls the artifact registry for compilation progress, retrieves S3 binary keys and SHA256 checksums, and handles fast-retry mechanisms for failed compilation attempts.',
    cookbookRef: 'Recipe 2 & 2b',
    contextRoute: 'builds',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'suiteId',
        label: 'Suite ID',
        placeholder: 'suite-auth-checkout',
        defaultValue: 'suite-auth-checkout',
      },
      {
        key: 'platform',
        label: 'Platform Architecture',
        placeholder: 'linux/amd64',
        defaultValue: 'linux/amd64',
      },
      {
        key: 'fixedSourceFile',
        label: 'Fixed Source Archive Path',
        placeholder: 'test-suite-fixed.tar.gz',
        defaultValue: 'test-suite-fixed.tar.gz',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      suiteId: 'suite-auth-checkout',
      platform: 'linux/amd64',
      fixedSourceFile: 'test-suite-fixed.tar.gz',
    },
    steps: [
      {
        id: 'step-query-artifacts',
        title: 'Step 1: Poll Build Artifacts Status',
        description:
          'Check compilation status (PENDING, BUILDING, READY, FAILED), binary S3 key, and SHA256 checksum.',
        endpoint: '/api/v1/suites/{id}/artifacts',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/suites/${p.suiteId || 'suite-auth-checkout'}/artifacts | jq .`,
      },
      {
        id: 'step-retry-build',
        title: 'Step 2: Retry Failed Compilation',
        description:
          'Re-upload corrected source code. Control plane resets FAILED status, cleans stale jobs, and dispatches fresh build.',
        endpoint: '/api/v1/suites/{id}/builds',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/suites/${p.suiteId || 'suite-auth-checkout'}/builds \\\n  -F "source=@${p.fixedSourceFile || 'test-suite-fixed.tar.gz'}" \\\n  -F "platform=${p.platform || 'linux/amd64'}"`,
      },
    ],
  },
  {
    id: 'recipe-3',
    title: 'Recipe 3: Defining Reusable Runner Profiles',
    subtitle: 'Profiles Page Context',
    description:
      'Creates and manages compute topologies, node affinity selectors, tolerations, and CPU/memory constraints decoupled from test scenario code.',
    cookbookRef: 'Recipe 3',
    contextRoute: 'profiles',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'profileName',
        label: 'Profile Name',
        placeholder: 'high-cpu-isolated-runners',
        defaultValue: 'high-cpu-isolated-runners',
      },
      {
        key: 'profileId',
        label: 'Profile ID (for update/delete)',
        placeholder: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
        defaultValue: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
      },
      {
        key: 'cpuRequest',
        label: 'CPU Request',
        placeholder: '2000m',
        defaultValue: '2000m',
      },
      {
        key: 'cpuLimit',
        label: 'CPU Limit',
        placeholder: '4000m',
        defaultValue: '4000m',
      },
      {
        key: 'memoryRequest',
        label: 'Memory Request',
        placeholder: '4Gi',
        defaultValue: '4Gi',
      },
      {
        key: 'memoryLimit',
        label: 'Memory Limit',
        placeholder: '8Gi',
        defaultValue: '8Gi',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      profileName: 'high-cpu-isolated-runners',
      profileId: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
      cpuRequest: '2000m',
      cpuLimit: '4000m',
      memoryRequest: '4Gi',
      memoryLimit: '8Gi',
    },
    steps: [
      {
        id: 'step-create-profile',
        title: 'Step 1: Create Runner Profile',
        description: 'Define CPU/memory limits, runner container image, node selectors, and tolerations.',
        endpoint: '/api/v1/profiles',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/profiles \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "name": "${p.profileName || 'high-cpu-isolated-runners'}",\n    "description": "Dedicated performance testing node pool profile",\n    "runner_image": "alpine:3.20",\n    "cpu_request": "${p.cpuRequest || '2000m'}",\n    "cpu_limit": "${p.cpuLimit || '4000m'}",\n    "memory_request": "${p.memoryRequest || '4Gi'}",\n    "memory_limit": "${p.memoryLimit || '8Gi'}",\n    "node_selector": {\n      "node-role.kubernetes.io/performance-runner": "true"\n    }\n  }'`,
      },
      {
        id: 'step-list-profiles',
        title: 'Step 2: List Available Profiles',
        description: 'Query all registered runner profiles across the cluster.',
        endpoint: '/api/v1/profiles',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/profiles | jq .`,
      },
      {
        id: 'step-update-profile',
        title: 'Step 3: Update Existing Profile',
        description: 'Update resource requests or limits for future test runs.',
        endpoint: '/api/v1/profiles/{id}',
        method: 'PUT',
        generateCurl: (p) =>
          `curl -i -X PUT ${p.baseUrl || 'http://localhost:8080'}/api/v1/profiles/${p.profileId || 'e8d665b1-2e67-4228-8ab6-79c5b248a31e'} \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "name": "${p.profileName || 'high-cpu-isolated-runners'}",\n    "cpu_request": "${p.cpuRequest || '4000m'}",\n    "cpu_limit": "${p.cpuLimit || '8000m'}",\n    "memory_request": "${p.memoryRequest || '8Gi'}",\n    "memory_limit": "${p.memoryLimit || '16Gi'}"\n  }'`,
      },
    ],
  },
  {
    id: 'recipe-4',
    title: 'Recipe 4: Triggering Ad-Hoc Test Runs',
    subtitle: 'Runs Page Context',
    description:
      'Dispatches an immediate ad-hoc load test execution with an active TestSuite, compiled Artifact, and RunnerProfile, spawning a managed batch/v1 Job.',
    cookbookRef: 'Recipe 5',
    contextRoute: 'runs',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'suiteId',
        label: 'Suite ID',
        placeholder: 'suite-auth-checkout',
        defaultValue: 'suite-auth-checkout',
      },
      {
        key: 'artifactId',
        label: 'Artifact ID',
        placeholder: 'c7a6e118-20ab-48d6-953b-e01140026e61',
        defaultValue: 'c7a6e118-20ab-48d6-953b-e01140026e61',
      },
      {
        key: 'profileId',
        label: 'Runner Profile ID',
        placeholder: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
        defaultValue: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
      },
      {
        key: 'configId',
        label: 'Configuration ID (optional)',
        placeholder: 'd1a85f64-5717-4562-b3fc-2c963f66afa7',
        defaultValue: 'd1a85f64-5717-4562-b3fc-2c963f66afa7',
      },
      {
        key: 'runId',
        label: 'Run ID (for status query)',
        placeholder: 'a1b2c3d4-e5f6-7890-abcd-ef0123456789',
        defaultValue: 'a1b2c3d4-e5f6-7890-abcd-ef0123456789',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      suiteId: 'suite-auth-checkout',
      artifactId: 'c7a6e118-20ab-48d6-953b-e01140026e61',
      profileId: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
      configId: 'd1a85f64-5717-4562-b3fc-2c963f66afa7',
      runId: 'a1b2c3d4-e5f6-7890-abcd-ef0123456789',
    },
    steps: [
      {
        id: 'step-trigger-run',
        title: 'Step 1: Dispatch Ad-Hoc Execution',
        description: 'POST request to trigger an immediate test execution.',
        endpoint: '/api/v1/runs',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "suite_id": "${p.suiteId || 'suite-auth-checkout'}",\n    "artifact_id": "${p.artifactId || 'c7a6e118-20ab-48d6-953b-e01140026e61'}",\n    "runner_profile_id": "${p.profileId || 'e8d665b1-2e67-4228-8ab6-79c5b248a31e'}",\n    "configuration_id": "${p.configId || 'd1a85f64-5717-4562-b3fc-2c963f66afa7'}"\n  }'`,
      },
      {
        id: 'step-check-run-status',
        title: 'Step 2: Inspect Execution Status',
        description: 'Query live run state, pods status, and initial telemetry.',
        endpoint: '/api/v1/runs/{id}',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs/${p.runId || 'a1b2c3d4-e5f6-7890-abcd-ef0123456789'} | jq .`,
      },
    ],
  },
  {
    id: 'recipe-5',
    title: 'Recipe 5: Setting Up Scheduled Test Runs',
    subtitle: 'Schedules Page Context',
    description:
      'Creates native Kubernetes batch/v1 CronJob schedules, updates cadence intervals, and triggers immediate out-of-band executions from configured CronJob templates.',
    cookbookRef: 'Recipe 4',
    contextRoute: 'schedules',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'scheduleName',
        label: 'Schedule Name',
        placeholder: 'nightly-checkout-benchmark',
        defaultValue: 'nightly-checkout-benchmark',
      },
      {
        key: 'cronExpr',
        label: 'Cron Expression',
        placeholder: '0 2 * * *',
        defaultValue: '0 2 * * *',
      },
      {
        key: 'suiteId',
        label: 'Suite ID',
        placeholder: 'suite-auth-checkout',
        defaultValue: 'suite-auth-checkout',
      },
      {
        key: 'artifactId',
        label: 'Artifact ID',
        placeholder: 'c7a6e118-20ab-48d6-953b-e01140026e61',
        defaultValue: 'c7a6e118-20ab-48d6-953b-e01140026e61',
      },
      {
        key: 'profileId',
        label: 'Runner Profile ID',
        placeholder: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
        defaultValue: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
      },
      {
        key: 'scheduleId',
        label: 'Schedule ID (for update/delete)',
        placeholder: '7fa1205c-d38e-4f51-b924-11883395bcf8',
        defaultValue: '7fa1205c-d38e-4f51-b924-11883395bcf8',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      scheduleName: 'nightly-checkout-benchmark',
      cronExpr: '0 2 * * *',
      suiteId: 'suite-auth-checkout',
      artifactId: 'c7a6e118-20ab-48d6-953b-e01140026e61',
      profileId: 'e8d665b1-2e67-4228-8ab6-79c5b248a31e',
      scheduleId: '7fa1205c-d38e-4f51-b924-11883395bcf8',
    },
    steps: [
      {
        id: 'step-create-schedule',
        title: 'Step 1: Create Scheduled Execution',
        description: 'Registers database schedule aggregate and provisions Kubernetes CronJob.',
        endpoint: '/api/v1/schedules',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/schedules \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "suite_id": "${p.suiteId || 'suite-auth-checkout'}",\n    "artifact_id": "${p.artifactId || 'c7a6e118-20ab-48d6-953b-e01140026e61'}",\n    "runner_profile_id": "${p.profileId || 'e8d665b1-2e67-4228-8ab6-79c5b248a31e'}",\n    "name": "${p.scheduleName || 'nightly-checkout-benchmark'}",\n    "cron_expression": "${p.cronExpr || '0 2 * * *'}"\n  }'`,
      },
      {
        id: 'step-update-schedule',
        title: 'Step 2: Update Schedule Cadence',
        description: 'Reconfigures cron recurrence expression on the native Kubernetes CronJob.',
        endpoint: '/api/v1/schedules/{id}',
        method: 'PUT',
        generateCurl: (p) =>
          `curl -i -X PUT ${p.baseUrl || 'http://localhost:8080'}/api/v1/schedules/${p.scheduleId || '7fa1205c-d38e-4f51-b924-11883395bcf8'} \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "cron_expression": "*/30 * * * *"\n  }'`,
      },
      {
        id: 'step-delete-schedule',
        title: 'Step 3: Delete Schedule',
        description: 'Tears down database record and deletes the underlying Kubernetes CronJob.',
        endpoint: '/api/v1/schedules/{id}',
        method: 'DELETE',
        generateCurl: (p) =>
          `curl -i -X DELETE ${p.baseUrl || 'http://localhost:8080'}/api/v1/schedules/${p.scheduleId || '7fa1205c-d38e-4f51-b924-11883395bcf8'}`,
      },
    ],
  },
  {
    id: 'recipe-6',
    title: 'Recipe 6: Querying Test Results, Logs & KPI Metrics',
    subtitle: 'Results View Context',
    description:
      'Queries indexed performance KPIs (p50, p90, p95, p99, TPS, error rates), streams raw container logs, and downloads raw summary JSON reports.',
    cookbookRef: 'Recipe 6 & 9',
    contextRoute: 'results',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'runId',
        label: 'Run ID',
        placeholder: '98bc19d4-1a3b-4882-a982-ff012498beaa',
        defaultValue: '98bc19d4-1a3b-4882-a982-ff012498beaa',
      },
      {
        key: 'suiteId',
        label: 'Suite ID (for filtering)',
        placeholder: 'suite-auth-checkout',
        defaultValue: 'suite-auth-checkout',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      runId: '98bc19d4-1a3b-4882-a982-ff012498beaa',
      suiteId: 'suite-auth-checkout',
    },
    steps: [
      {
        id: 'step-query-kpis',
        title: 'Step 1: Query Execution Details & Indexed KPIs',
        description: 'Fetch indexed response latency percentiles, SLA evaluation, and status.',
        endpoint: '/api/v1/runs/{id}',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs/${p.runId || '98bc19d4-1a3b-4882-a982-ff012498beaa'} | jq .`,
      },
      {
        id: 'step-stream-logs',
        title: 'Step 2: Stream Execution Logs',
        description: 'Stream execution logs or append ?presign=true for signed S3 URL.',
        endpoint: '/api/v1/runs/{id}/logs',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs/${p.runId || '98bc19d4-1a3b-4882-a982-ff012498beaa'}/logs`,
      },
      {
        id: 'step-fetch-report',
        title: 'Step 3: Download Raw Summary Report (summary.json)',
        description: 'Download the full vuhive engine summary report JSON directly from storage.',
        endpoint: '/api/v1/runs/{id}/report',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs/${p.runId || '98bc19d4-1a3b-4882-a982-ff012498beaa'}/report | jq .`,
      },
      {
        id: 'step-filter-runs',
        title: 'Step 4: List & Filter Historical Runs',
        description: 'Query runs with pagination, filtering by suite ID and completion status.',
        endpoint: '/api/v1/runs',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s "${p.baseUrl || 'http://localhost:8080'}/api/v1/runs?suite_id=${p.suiteId || 'suite-auth-checkout'}&status=COMPLETED&limit=10" | jq .`,
      },
    ],
  },
  {
    id: 'recipe-7',
    title: 'Recipe 7: Cancelling / Aborting Running Executions',
    subtitle: 'Active Run Context',
    description:
      'Immediately stops an in-flight execution, triggers graceful SIGTERM log flushes on runner pods, deletes the underlying Kubernetes Job, and records abort diagnostics.',
    cookbookRef: 'Recipe 8',
    contextRoute: 'active-run',
    paramFields: [
      {
        key: 'baseUrl',
        label: 'Control Plane Base URL',
        placeholder: 'http://localhost:8080',
        defaultValue: 'http://localhost:8080',
      },
      {
        key: 'runId',
        label: 'Active Run ID',
        placeholder: 'run-9f8e7d6c',
        defaultValue: 'run-9f8e7d6c',
      },
      {
        key: 'reason',
        label: 'Abort Reason',
        placeholder: 'Observed unexpected latency spike exceeding SLA boundaries',
        defaultValue: 'Observed unexpected latency spike exceeding SLA boundaries',
      },
      {
        key: 'requestedBy',
        label: 'Requested By',
        placeholder: 'operator-dashboard',
        defaultValue: 'operator-dashboard',
      },
    ],
    defaultParams: {
      baseUrl: 'http://localhost:8080',
      runId: 'run-9f8e7d6c',
      reason: 'Observed unexpected latency spike exceeding SLA boundaries',
      requestedBy: 'operator-dashboard',
    },
    steps: [
      {
        id: 'step-abort-run',
        title: 'Step 1: Abort In-Flight Execution',
        description: 'Dispatches emergency abort request to cancel the active run.',
        endpoint: '/api/v1/runs/{id}/abort',
        method: 'POST',
        generateCurl: (p) =>
          `curl -i -X POST ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs/${p.runId || 'run-9f8e7d6c'}/abort \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "reason": "${p.reason || 'Observed unexpected latency spike'}",\n    "requested_by": "${p.requestedBy || 'operator-dashboard'}"\n  }'`,
      },
      {
        id: 'step-verify-aborted',
        title: 'Step 2: Verify Aborted State',
        description: 'Verify status is ABORTED and check abort diagnostics and teardown timestamp.',
        endpoint: '/api/v1/runs/{id}',
        method: 'GET',
        generateCurl: (p) =>
          `curl -s ${p.baseUrl || 'http://localhost:8080'}/api/v1/runs/${p.runId || 'run-9f8e7d6c'} | jq .`,
      },
    ],
  },
]

export function getRecipeById(id: RecipeId): RecipeDefinition | undefined {
  return RECIPES.find((r) => r.id === id)
}

export function getRecipeForRoute(route: RouteId | string): RecipeDefinition {
  switch (route) {
    case 'suites':
      return RECIPES[0] // recipe-1
    case 'builds':
      return RECIPES[1] // recipe-2
    case 'profiles':
      return RECIPES[2] // recipe-3
    case 'runs':
      return RECIPES[3] // recipe-4
    case 'schedules':
      return RECIPES[4] // recipe-5
    case 'results':
      return RECIPES[5] // recipe-6
    case 'active-run':
      return RECIPES[6] // recipe-7
    case 'dashboard':
    default:
      return RECIPES[0] // recipe-1
  }
}
