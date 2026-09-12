export type TestSuiteState = 'DRAFT' | 'ACTIVE' | 'ARCHIVED'
export type BuildStatus = 'READY' | 'BUILDING' | 'FAILED' | 'PENDING' | 'CANCELLED'
export type RunExecutionStatus = 'QUEUED' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'ABORTED'

export interface TestSuite {
  id: string
  name: string
  description: string
  state: TestSuiteState
  createdAt: string
  updatedAt: string
  buildStatus?: BuildStatus
  platforms?: string[]
  runCount?: number
  lastRunAt?: string
}

export interface SuiteConfiguration {
  id: string
  suiteId: string
  name: string
  contentYaml: string
  s3ConfigKey?: string
  isDefault: boolean
  createdAt: string
}

export interface CompiledArtifact {
  id: string
  suiteId: string
  platform: string
  s3BinaryKey?: string
  sha256Checksum?: string
  buildLogsS3Key?: string
  status: BuildStatus
  errorMessage?: string
  createdAt: string
}

export interface RunMetrics {
  totalIterations?: number
  totalRequests?: number
  avgTps?: number
  p50DurationMs?: number
  p90DurationMs?: number
  p95DurationMs?: number
  p99DurationMs?: number
  errorRatePct?: number
}

export interface HistoricalRun {
  id: string
  suiteId: string
  artifactId?: string
  configurationId?: string
  runnerProfileId?: string
  scheduleId?: string
  status: RunExecutionStatus
  k8sJobName?: string
  startedAt?: string
  finishedAt?: string
  durationMs?: number
  exitCode?: number
  slaPassed?: boolean
  metrics?: RunMetrics
  k8sNamespace?: string
  abortReason?: string
  activeDeadlineSeconds?: number
  createdAt: string
}

export interface TriggerRunInput {
  suite_id: string
  artifact_id: string
  runner_profile_id: string
  configuration_id?: string
  runner_namespace?: string
  active_deadline_seconds?: number
}
