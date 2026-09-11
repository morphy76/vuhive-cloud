/**
 * Types representing the vuhive summary.json telemetry report structure.
 */

export interface SummaryMetricItem {
  name: string
  type: 'counter' | 'rate' | 'duration' | 'gauge' | string
  count?: number
  value?: number
  rate?: number
  min?: number
  mean?: number
  p50?: number
  p90?: number
  p95?: number
  p99?: number
  max?: number
  [key: string]: any
}

export interface SummaryThresholdItem {
  metric: string
  stat: string
  operator: string
  target: string
  actual: string
  passed: boolean
}

export interface SummaryStepItem {
  name: string
  scenario?: string
  requests: number
  tps?: number
  p50_ms?: number
  p90_ms?: number
  p95_ms?: number
  p99_ms?: number
  failed_requests?: number
  error_rate_pct?: number
  status?: 'PASS' | 'FAILED' | string
  [key: string]: any
}

export interface SummaryReport {
  suite_name?: string
  scenario?: string
  version?: string
  commit?: string
  started_at?: string
  ended_at?: string
  duration?: number | string
  status?: string
  passed?: boolean
  sla_passed?: boolean
  total_iterations?: number
  total_requests?: number
  avg_tps?: number
  p50_duration_ms?: number
  p90_duration_ms?: number
  p95_duration_ms?: number
  p99_duration_ms?: number
  error_rate_pct?: number
  steps?: SummaryStepItem[]
  metrics?: SummaryMetricItem[]
  thresholds?: SummaryThresholdItem[]
  custom_metrics?: Record<string, any>
  [key: string]: any
}
