import * as React from 'react'
import {
  Layers,
  Cpu,
  Activity,
  Zap,
  Repeat,
  Copy,
  Check,
  FileJson,
  FileText,
  Calendar,
  X,
  Server,
  ShieldCheck,
  ShieldAlert,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { useProfile } from '@/hooks/use-profiles'
import { useSuiteArtifacts } from '@/hooks/use-suites'
import { VisualAnalyticsSection } from '@/components/charts/VisualAnalyticsSection'
import type { HistoricalRun } from '@/types/suite'

export interface RunSummaryDashboardProps {
  run: HistoricalRun
  onClose?: () => void
}

function formatDuration(ms?: number): string {
  if (!ms || ms <= 0) return '00s'
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  const pad = (n: number) => String(n).padStart(2, '0')

  if (hours > 0) {
    return `${pad(hours)}h ${pad(minutes)}m ${pad(seconds)}s`
  }
  return `${pad(minutes)}m ${pad(seconds)}s`
}

function formatTimestamp(isoStr?: string): { local: string; utc: string } {
  if (!isoStr) return { local: 'N/A', utc: 'N/A' }
  try {
    const d = new Date(isoStr)
    return {
      local: d.toLocaleString(),
      utc: d.toISOString().replace('T', ' ').slice(0, 19) + ' UTC',
    }
  } catch {
    return { local: isoStr, utc: isoStr }
  }
}

export const RunSummaryDashboard: React.FC<RunSummaryDashboardProps> = ({
  run,
  onClose,
}) => {
  const [copiedChecksum, setCopiedChecksum] = React.useState(false)

  // Fetch contextual runner profile details
  const { data: profile } = useProfile(run.runnerProfileId || '')

  // Fetch artifacts for suite to locate matching checksum & platform
  const { data: artifacts = [] } = useSuiteArtifacts(run.suiteId)

  const matchedArtifact = React.useMemo(() => {
    if (!run.artifactId) return null
    return artifacts.find((a) => a.id === run.artifactId) || null
  }, [artifacts, run.artifactId])

  const checksum = matchedArtifact?.sha256Checksum || run.artifactId || 'N/A'
  const targetPlatform = matchedArtifact?.platform || (run.artifactId?.includes('arm64') ? 'linux/arm64' : 'linux/amd64')

  const durationStr = React.useMemo(() => {
    if (run.durationMs && run.durationMs > 0) {
      return formatDuration(run.durationMs)
    }
    if (run.startedAt && run.finishedAt) {
      const diff = new Date(run.finishedAt).getTime() - new Date(run.startedAt).getTime()
      if (diff > 0) return formatDuration(diff)
    }
    return '-'
  }, [run.durationMs, run.startedAt, run.finishedAt])

  const exitCode = run.exitCode !== undefined ? run.exitCode : (run.status === 'COMPLETED' ? 0 : 1)
  const isSlaPassed = run.slaPassed !== false && exitCode === 0 && run.status === 'COMPLETED'

  // Normalise error rate %
  const errorRateVal = React.useMemo(() => {
    if (run.metrics?.errorRatePct === undefined) return 0
    return run.metrics.errorRatePct <= 1.0 && run.metrics.errorRatePct > 0
      ? run.metrics.errorRatePct * 100
      : run.metrics.errorRatePct
  }, [run.metrics?.errorRatePct])

  const errorSeverity = React.useMemo(() => {
    if (errorRateVal <= 0.0001) return 'normal'
    if (errorRateVal < 5.0) return 'warning'
    return 'critical'
  }, [errorRateVal])

  const handleCopyChecksum = async () => {
    if (checksum && checksum !== 'N/A') {
      try {
        await navigator.clipboard.writeText(checksum)
        setCopiedChecksum(true)
        setTimeout(() => setCopiedChecksum(false), 2000)
      } catch (err) {
        console.error('Failed copying checksum to clipboard', err)
      }
    }
  }

  // Node Requirements / Placement string from profile
  const nodeRequirements = React.useMemo(() => {
    if (!profile) return 'Standard Node Pool'
    const parts: string[] = []
    if (profile.affinity?.node_selector_terms?.length) {
      for (const term of profile.affinity.node_selector_terms) {
        if (term.values?.length) {
          parts.push(term.values.join(', '))
        }
      }
    }
    if (profile.node_selector && Object.keys(profile.node_selector).length > 0) {
      for (const [k, v] of Object.entries(profile.node_selector)) {
        parts.push(`${k}=${v}`)
      }
    }
    return parts.length > 0 ? parts.join(' • ') : 'Standard Node Pool'
  }, [profile])

  const startFormatted = formatTimestamp(run.startedAt || run.createdAt)
  const finishFormatted = formatTimestamp(run.finishedAt)

  return (
    <div className="space-y-6">
      {/* Executive Header & Navigation */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-brand-600 dark:text-brand-400">
              Executive Test Run Summary
            </span>
            <span className="font-mono text-xs text-slate-400 dark:text-slate-500 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded-md">
              {run.id}
            </span>
          </div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white mt-1">
            Performance & SLA Audit
          </h2>
          <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400 mt-0.5">
            Indexed telemetry indicators, SLA evaluation, and execution environment metadata.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            asChild
            className="min-h-[40px] gap-2 border-slate-200 dark:border-slate-800"
          >
            <a
              href={`/api/v1/runs/${encodeURIComponent(run.id)}/report?presign=true`}
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Download summary JSON report"
            >
              <FileJson className="w-4 h-4 text-brand-600 dark:text-brand-400" />
              <span>Raw Report</span>
            </a>
          </Button>

          <Button
            variant="outline"
            size="sm"
            asChild
            className="min-h-[40px] gap-2 border-slate-200 dark:border-slate-800"
          >
            <a
              href={`/api/v1/runs/${encodeURIComponent(run.id)}/logs?presign=true`}
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Download execution logs"
            >
              <FileText className="w-4 h-4 text-slate-500" />
              <span>Run Logs</span>
            </a>
          </Button>

          {onClose && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={onClose}
              className="h-9 w-9 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
              aria-label="Close summary dashboard"
            >
              <X className="w-5 h-5" />
            </Button>
          )}
        </div>
      </div>

      {/* 1. SLA Compliance Banner */}
      <div
        role="status"
        aria-label={`${isSlaPassed ? 'SLA PASSED' : 'SLA FAILED'} - Exit Code: ${exitCode}, Duration: ${durationStr}`}
        className={`rounded-2xl p-6 shadow-sm border transition-colors ${
          isSlaPassed
            ? 'bg-emerald-600 dark:bg-emerald-700 text-white border-emerald-500'
            : 'bg-rose-600 dark:bg-rose-700 text-white border-rose-500'
        }`}
      >
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6">
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-2xl bg-white/15 backdrop-blur-xs flex-shrink-0">
              {isSlaPassed ? (
                <ShieldCheck className="w-8 h-8 text-white" aria-hidden="true" />
              ) : (
                <ShieldAlert className="w-8 h-8 text-white" aria-hidden="true" />
              )}
            </div>
            <div>
              <div className="flex items-center gap-3">
                <h3 className="text-2xl font-extrabold tracking-tight">
                  {isSlaPassed ? 'SLA PASSED' : 'SLA FAILED'}
                </h3>
                <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-white/20 text-white uppercase tracking-wider">
                  {run.status}
                </span>
              </div>
              <p className="text-sm text-white/90 mt-1 max-w-xl">
                {isSlaPassed
                  ? 'All configured response latency thresholds, error rate quotas, and test objectives were strictly satisfied.'
                  : 'Performance thresholds were breached or the runner process encountered execution failure during the run.'}
              </p>
            </div>
          </div>

          {/* Quick Technical KPIs on Banner */}
          <div className="flex flex-wrap items-center gap-3 border-t lg:border-t-0 lg:border-l border-white/20 pt-4 lg:pt-0 lg:pl-6">
            <div className="px-4 py-2 rounded-xl bg-white/10 backdrop-blur-xs">
              <div className="text-[11px] text-white/80 font-medium uppercase tracking-wider">
                Exit Code
              </div>
              <div className="font-mono text-base font-bold text-white mt-0.5">
                Exit Code: {exitCode}
              </div>
            </div>

            <div className="px-4 py-2 rounded-xl bg-white/10 backdrop-blur-xs">
              <div className="text-[11px] text-white/80 font-medium uppercase tracking-wider">
                Run Duration
              </div>
              <div className="font-mono text-base font-bold text-white mt-0.5">
                {durationStr}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 2. Primary KPI Metric Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Card 1: Total Iterations & Total Requests */}
        <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                Total Iterations & Requests
              </span>
              <HelpTooltip
                text="Completed scenario loops and total HTTP/messaging transactions generated during execution."
                label="Help for total iterations and requests"
              />
            </div>
            <div className="p-2 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
              <Repeat className="w-5 h-5" />
            </div>
          </div>

          <div className="mt-4 space-y-2">
            <div>
              <div className="text-2xl sm:text-3xl font-extrabold font-mono text-slate-900 dark:text-white">
                {run.metrics?.totalIterations !== undefined
                  ? run.metrics.totalIterations.toLocaleString()
                  : '-'}
              </div>
              <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">
                Total Iterations
              </div>
            </div>
            <div className="pt-2 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between">
              <span className="text-xs text-slate-500 dark:text-slate-400">Total Requests:</span>
              <span className="font-mono font-semibold text-slate-900 dark:text-white text-xs">
                {run.metrics?.totalRequests !== undefined
                  ? run.metrics.totalRequests.toLocaleString()
                  : '-'}
              </span>
            </div>
          </div>
        </div>

        {/* Card 2: Average Throughput (TPS) */}
        <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                Average Throughput
              </span>
              <HelpTooltip
                text="Transactions Per Second (TPS): average throughput sustained across the full scenario duration."
                label="Help for average throughput"
              />
            </div>
            <div className="p-2 rounded-xl bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400">
              <Zap className="w-5 h-5" />
            </div>
          </div>

          <div className="mt-4">
            <div className="flex items-baseline gap-1.5">
              <span className="text-2xl sm:text-3xl font-extrabold font-mono text-slate-900 dark:text-white">
                {run.metrics?.avgTps !== undefined
                  ? run.metrics.avgTps.toLocaleString(undefined, {
                      minimumFractionDigits: 1,
                      maximumFractionDigits: 2,
                    })
                  : '-'}
              </span>
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">
                req/s
              </span>
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-2">
              Sustained transaction throughput
            </div>
          </div>
        </div>

        {/* Card 3: Error Rate % with Warning Threshold */}
        <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                Error Rate
              </span>
              <HelpTooltip
                text="Percentage of failed requests (HTTP 5xx, gateway errors, or request timeouts) with color threshold warning."
                label="Help for error rate"
              />
            </div>
            <Badge
              variant={
                errorSeverity === 'normal'
                  ? 'success'
                  : errorSeverity === 'warning'
                  ? 'warning'
                  : 'error'
              }
            >
              {errorSeverity === 'normal'
                ? 'Normal'
                : errorSeverity === 'warning'
                ? 'Warning'
                : 'Critical'}
            </Badge>
          </div>

          <div className="mt-4">
            <div
              className={`text-2xl sm:text-3xl font-extrabold font-mono ${
                errorSeverity === 'normal'
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : errorSeverity === 'warning'
                  ? 'text-amber-600 dark:text-amber-400'
                  : 'text-rose-600 dark:text-rose-400'
              }`}
            >
              {errorRateVal.toFixed(2)}%
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-2">
              {errorSeverity === 'normal'
                ? '0.00% failure rate (zero anomalies)'
                : errorSeverity === 'warning'
                ? 'Non-zero error rate within tolerance threshold (<5%)'
                : 'Excessive error rate exceeding safety threshold (≥5%)'}
            </div>
          </div>
        </div>

        {/* Card 4: Latency Percentiles Summary Badges */}
        <div className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                Latency Percentiles
              </span>
              <HelpTooltip
                text="Response duration distribution percentiles (p50, p90, p95, p99) measured across all transactions."
                label="Help for latency percentiles"
              />
            </div>
            <div className="p-2 rounded-xl bg-teal-50 dark:bg-teal-950/60 text-teal-600 dark:text-teal-400">
              <Activity className="w-5 h-5" />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2 mt-2">
            <div className="p-2 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
              <div className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
                p50
              </div>
              <div className="font-mono text-sm font-bold text-slate-900 dark:text-white mt-0.5">
                {run.metrics?.p50DurationMs !== undefined
                  ? `${run.metrics.p50DurationMs.toFixed(1)} ms`
                  : '-'}
              </div>
            </div>

            <div className="p-2 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
              <div className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
                p90
              </div>
              <div className="font-mono text-sm font-bold text-slate-900 dark:text-white mt-0.5">
                {run.metrics?.p90DurationMs !== undefined
                  ? `${run.metrics.p90DurationMs.toFixed(1)} ms`
                  : '-'}
              </div>
            </div>

            <div className="p-2 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
              <div className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
                p95
              </div>
              <div className="font-mono text-sm font-bold text-slate-900 dark:text-white mt-0.5">
                {run.metrics?.p95DurationMs !== undefined
                  ? `${run.metrics.p95DurationMs.toFixed(1)} ms`
                  : '-'}
              </div>
            </div>

            <div className="p-2 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800">
              <div className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
                p99
              </div>
              <div className="font-mono text-sm font-bold text-slate-900 dark:text-white mt-0.5">
                {run.metrics?.p99DurationMs !== undefined
                  ? `${run.metrics.p99DurationMs.toFixed(1)} ms`
                  : '-'}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 3. Interactive Visual Analytics (Latency Percentiles, Throughput Correlation, Historical Trends) */}
      <VisualAnalyticsSection run={run} />

      {/* 4. Execution Environment & Technical Metadata */}
      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xs space-y-6">
        <div>
          <h3 className="text-base font-bold text-slate-900 dark:text-white">
            Execution Metadata & Cluster Topology
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Cryptographic artifact verification, runner resource bounds, and Kubernetes scheduling context.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Runner Profile Metadata */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800 flex flex-col justify-between">
            <div>
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-500 dark:text-slate-400">
                <Cpu className="w-4 h-4 text-brand-500" />
                <span>Runner Profile</span>
              </div>
              <div className="font-semibold text-sm text-slate-900 dark:text-white mt-2">
                {profile?.name || run.runnerProfileId || 'default-runner'}
              </div>
              <div className="text-xs text-slate-500 dark:text-slate-400 mt-1 font-mono">
                CPU: {profile?.cpu_request || '500m'} - {profile?.cpu_limit || '1000m'}
              </div>
              <div className="text-xs text-slate-500 dark:text-slate-400 font-mono">
                RAM: {profile?.memory_request || '512Mi'} - {profile?.memory_limit || '1Gi'}
              </div>
            </div>
            <div className="mt-3 pt-2 border-t border-slate-200/60 dark:border-slate-800 text-[11px] text-slate-400 truncate">
              Image: {profile?.runner_image || 'alpine:3.20'}
            </div>
          </div>

          {/* Kubernetes Node & Job Information */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800 flex flex-col justify-between">
            <div>
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-500 dark:text-slate-400">
                <Server className="w-4 h-4 text-indigo-500" />
                <span>Kubernetes Topology</span>
              </div>
              <div className="font-mono text-xs font-semibold text-slate-900 dark:text-white mt-2 truncate">
                {run.k8sJobName || `vuhive-run-${run.id}`}
              </div>
              <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Namespace:{' '}
                <span className="font-mono font-medium text-slate-800 dark:text-slate-200">
                  {run.k8sNamespace || 'vuhive-runners'}
                </span>
              </div>
            </div>
            <div className="mt-3 pt-2 border-t border-slate-200/60 dark:border-slate-800 text-[11px] text-slate-500 dark:text-slate-400 truncate">
              Target Nodes: <span className="font-mono font-semibold">{nodeRequirements}</span>
            </div>
          </div>

          {/* Artifact Checksum & Platform */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-500 dark:text-slate-400">
                  <Layers className="w-4 h-4 text-teal-500" />
                  <span>Compiled Artifact</span>
                </div>
                <Badge variant="outline" className="font-mono text-[10px]">
                  {targetPlatform}
                </Badge>
              </div>

              <div className="mt-2">
                <div className="text-xs text-slate-400">SHA256 Checksum</div>
                <div className="flex items-center gap-2 mt-1">
                  <span className="font-mono text-xs font-medium text-slate-800 dark:text-slate-200 truncate max-w-[140px] sm:max-w-[160px]">
                    {checksum}
                  </span>
                  {checksum !== 'N/A' && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={handleCopyChecksum}
                      className="h-7 w-7 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
                      aria-label="Copy checksum"
                    >
                      {copiedChecksum ? (
                        <Check className="w-3.5 h-3.5 text-emerald-500" />
                      ) : (
                        <Copy className="w-3.5 h-3.5" />
                      )}
                    </Button>
                  )}
                </div>
              </div>
            </div>

            <div className="mt-3 pt-2 border-t border-slate-200/60 dark:border-slate-800 text-[11px] text-slate-400 truncate">
              ID: {run.artifactId || 'default-binary'}
            </div>
          </div>

          {/* Start & Finish Timestamps */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/80 dark:border-slate-800 flex flex-col justify-between">
            <div>
              <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-500 dark:text-slate-400">
                <Calendar className="w-4 h-4 text-amber-500" />
                <span>Execution Window</span>
              </div>

              <div className="mt-2 space-y-1.5">
                <div>
                  <div className="text-[10px] text-slate-400 uppercase font-semibold">Started</div>
                  <div className="text-xs font-medium text-slate-800 dark:text-slate-200">
                    {startFormatted.local}
                  </div>
                </div>
                <div>
                  <div className="text-[10px] text-slate-400 uppercase font-semibold">Finished</div>
                  <div className="text-xs font-medium text-slate-800 dark:text-slate-200">
                    {finishFormatted.local}
                  </div>
                </div>
              </div>
            </div>

            <div className="mt-3 pt-2 border-t border-slate-200/60 dark:border-slate-800 text-[11px] text-slate-400 flex items-center justify-between">
              <span>Wall-Clock:</span>
              <span className="font-mono font-bold text-slate-700 dark:text-slate-300">
                {durationStr}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
