import * as React from 'react'
import {
  Clock,
  Play,
  CheckCircle2,
  AlertOctagon,
  AlertTriangle,
  Layers,
  Cpu,
  ShieldAlert,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { AbortConfirmationDialog } from '@/components/dialogs/AbortConfirmationDialog'
import { useAbortRun } from '@/hooks/use-runs'
import { useRunEvents } from '@/hooks/use-events'
import type { HistoricalRun, RunExecutionStatus } from '@/types/suite'

export interface LiveRunMonitorProps {
  run: HistoricalRun
  onClose?: () => void
  onRunAborted?: (run: HistoricalRun) => void
}

function formatDuration(ms: number): string {
  if (ms <= 0) return '00s'
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

export const LiveRunMonitor: React.FC<LiveRunMonitorProps> = ({
  run: initialRun,
  onClose,
  onRunAborted,
}) => {
  const [run, setRun] = React.useState<HistoricalRun>(initialRun)
  const [isAbortDialogOpen, setIsAbortDialogOpen] = React.useState(false)
  const [elapsedMs, setElapsedMs] = React.useState<number>(() => {
    if (initialRun.durationMs && initialRun.durationMs > 0) return initialRun.durationMs
    const start = new Date(initialRun.startedAt || initialRun.createdAt).getTime()
    return Math.max(0, Date.now() - start)
  })

  const abortRunMutation = useAbortRun()

  // Sync state if prop changes
  React.useEffect(() => {
    setRun(initialRun)
  }, [initialRun])

  // Listen for live SSE run status updates
  useRunEvents(run.id, (event) => {
    setRun((prev) => ({
      ...prev,
      status: event.status as RunExecutionStatus,
      startedAt: event.started_at || prev.startedAt,
      finishedAt: event.finished_at || prev.finishedAt,
      durationMs: event.duration_ms !== undefined ? event.duration_ms : prev.durationMs,
      exitCode: event.exit_code !== undefined ? event.exit_code : prev.exitCode,
      slaPassed: event.sla_passed !== undefined ? event.sla_passed : prev.slaPassed,
      k8sJobName: event.k8s_job_name || prev.k8sJobName,
      metrics: event.metrics
        ? {
            totalIterations: event.metrics.total_iterations,
            totalRequests: event.metrics.total_requests,
            avgTps: event.metrics.avg_tps,
            p50DurationMs: event.metrics.p50_duration_ms,
            p90DurationMs: event.metrics.p90_duration_ms,
            p95DurationMs: event.metrics.p95_duration_ms,
            p99DurationMs: event.metrics.p99_duration_ms,
            errorRatePct: event.metrics.error_rate_pct,
          }
        : prev.metrics,
    }))
  })

  // 1-second interval timer for active runs
  React.useEffect(() => {
    const isActive = run.status === 'QUEUED' || run.status === 'RUNNING'
    if (!isActive) {
      if (run.durationMs && run.durationMs > 0) {
        setElapsedMs(run.durationMs)
      }
      return
    }

    const timer = setInterval(() => {
      const start = new Date(run.startedAt || run.createdAt).getTime()
      setElapsedMs(Math.max(0, Date.now() - start))
    }, 1000)

    return () => clearInterval(timer)
  }, [run.status, run.startedAt, run.createdAt, run.durationMs])

  const canAbort = run.status === 'QUEUED' || run.status === 'RUNNING'

  const handleAbortConfirm = async (reason: string) => {
    try {
      const updated = await abortRunMutation.mutateAsync({ id: run.id, reason })
      setRun(updated)
      setIsAbortDialogOpen(false)
      onRunAborted?.(updated)
    } catch (err) {
      console.error('Failed to abort run:', err)
    }
  }

  return (
    <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-brand-50 dark:bg-brand-950/50 text-brand-600 dark:text-brand-400">
            <Play className="w-5 h-5 fill-current" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-base font-bold text-slate-900 dark:text-white">
                Live Execution Monitor
              </h2>
              <span className="font-mono text-xs text-slate-400 dark:text-slate-500 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded-md">
                {run.id}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              Real-time Kubernetes batch/v1 load test telemetry & lifecycle control
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {canAbort && (
            <Button
              type="button"
              onClick={() => setIsAbortDialogOpen(true)}
              className="bg-rose-600 hover:bg-rose-700 text-white font-semibold text-xs min-h-[38px] px-4 gap-2 shadow-xs focus-visible:ring-rose-500"
              aria-label="Abort Test"
            >
              <ShieldAlert className="w-4 h-4" />
              <span>Abort Test</span>
            </Button>
          )}

          {onClose && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={onClose}
              className="h-8 w-8 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
              aria-label="Close Monitor"
            >
              <X className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>

      <div className="p-6 space-y-6">
        {/* State Timeline Stepper & Live Duration Timer */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {/* Phase Stepper */}
          <div className="lg:col-span-2 rounded-xl p-4 bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800 flex flex-col justify-center">
            <div className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-3">
              Execution Phase Timeline
            </div>
            <div className="flex items-center justify-between relative">
              {/* Step 1: QUEUED */}
              <div className="flex flex-col items-center relative z-10">
                <div
                  className={`w-9 h-9 rounded-full flex items-center justify-center border-2 transition-colors ${
                    run.status === 'QUEUED'
                      ? 'bg-brand-50 border-brand-500 text-brand-600 dark:bg-brand-950 dark:border-brand-400 dark:text-brand-300 ring-4 ring-brand-100 dark:ring-brand-900/50'
                      : 'bg-emerald-50 border-emerald-500 text-emerald-600 dark:bg-emerald-950 dark:border-emerald-500 dark:text-emerald-400'
                  }`}
                >
                  {run.status === 'QUEUED' ? (
                    <Clock className="w-4 h-4 animate-pulse" />
                  ) : (
                    <CheckCircle2 className="w-4 h-4" />
                  )}
                </div>
                <span className="text-xs font-semibold text-slate-900 dark:text-slate-100 mt-1.5">
                  QUEUED
                </span>
                <span className="text-[10px] text-slate-500 dark:text-slate-400">Job Enqueued</span>
              </div>

              <div
                className={`flex-1 h-0.5 mx-2 transition-colors ${
                  run.status !== 'QUEUED'
                    ? 'bg-emerald-500 dark:bg-emerald-600'
                    : 'bg-slate-200 dark:bg-slate-700'
                }`}
              />

              {/* Step 2: RUNNING */}
              <div className="flex flex-col items-center relative z-10">
                <div
                  className={`w-9 h-9 rounded-full flex items-center justify-center border-2 transition-colors ${
                    run.status === 'RUNNING'
                      ? 'bg-amber-50 border-amber-500 text-amber-600 dark:bg-amber-950 dark:border-amber-400 dark:text-amber-300 ring-4 ring-amber-100 dark:ring-amber-900/50'
                      : run.status === 'COMPLETED'
                      ? 'bg-emerald-50 border-emerald-500 text-emerald-600 dark:bg-emerald-950 dark:border-emerald-500 dark:text-emerald-400'
                      : run.status === 'ABORTED' || run.status === 'FAILED'
                      ? 'bg-slate-100 border-slate-400 text-slate-600 dark:bg-slate-800 dark:border-slate-600 dark:text-slate-400'
                      : 'bg-white border-slate-200 text-slate-300 dark:bg-slate-900 dark:border-slate-800'
                  }`}
                >
                  {run.status === 'RUNNING' ? (
                    <Play className="w-4 h-4 fill-current animate-pulse" />
                  ) : run.status === 'COMPLETED' ? (
                    <CheckCircle2 className="w-4 h-4" />
                  ) : (
                    <span className="text-xs font-mono font-bold">2</span>
                  )}
                </div>
                <span className="text-xs font-semibold text-slate-900 dark:text-slate-100 mt-1.5">
                  RUNNING
                </span>
                <span className="text-[10px] text-slate-500 dark:text-slate-400">Generating Load</span>
              </div>

              <div
                className={`flex-1 h-0.5 mx-2 transition-colors ${
                  run.status === 'COMPLETED'
                    ? 'bg-emerald-500 dark:bg-emerald-600'
                    : run.status === 'FAILED' || run.status === 'ABORTED'
                    ? 'bg-rose-500 dark:bg-rose-600'
                    : 'bg-slate-200 dark:bg-slate-700'
                }`}
              />

              {/* Step 3: Terminal Status */}
              <div className="flex flex-col items-center relative z-10">
                <div
                  className={`w-9 h-9 rounded-full flex items-center justify-center border-2 transition-colors ${
                    run.status === 'COMPLETED'
                      ? 'bg-emerald-500 border-emerald-600 text-white ring-4 ring-emerald-100 dark:ring-emerald-900/50'
                      : run.status === 'FAILED'
                      ? 'bg-rose-500 border-rose-600 text-white ring-4 ring-rose-100 dark:ring-rose-900/50'
                      : run.status === 'ABORTED'
                      ? 'bg-slate-600 border-slate-700 text-white ring-4 ring-slate-100 dark:ring-slate-800'
                      : 'bg-white border-slate-200 text-slate-300 dark:bg-slate-900 dark:border-slate-800'
                  }`}
                >
                  {run.status === 'COMPLETED' ? (
                    <CheckCircle2 className="w-4 h-4" />
                  ) : run.status === 'FAILED' ? (
                    <AlertTriangle className="w-4 h-4" />
                  ) : run.status === 'ABORTED' ? (
                    <AlertOctagon className="w-4 h-4" />
                  ) : (
                    <CheckCircle2 className="w-4 h-4" />
                  )}
                </div>
                <span className="text-xs font-semibold text-slate-900 dark:text-slate-100 mt-1.5">
                  {run.status === 'ABORTED'
                    ? 'ABORTED'
                    : run.status === 'FAILED'
                    ? 'FAILED'
                    : 'COMPLETED'}
                </span>
                <span className="text-[10px] text-slate-500 dark:text-slate-400">
                  {run.status === 'ABORTED'
                    ? 'Terminated'
                    : run.status === 'FAILED'
                    ? 'Failed'
                    : 'Finalized'}
                </span>
              </div>
            </div>
          </div>

          {/* Running Duration Timer Card */}
          <div className="rounded-xl p-4 bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800 flex flex-col justify-between">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Wall-Clock Duration
              </span>
              {(run.status === 'RUNNING' || run.status === 'QUEUED') && (
                <span className="flex items-center gap-1.5 text-[11px] font-semibold text-amber-600 dark:text-amber-400">
                  <span className="w-2 h-2 rounded-full bg-amber-500 animate-ping" />
                  Live
                </span>
              )}
            </div>
            <div className="my-2">
              <div className="text-3xl font-mono font-bold tracking-tight text-slate-900 dark:text-white">
                {formatDuration(elapsedMs)}
              </div>
              <div className="text-[11px] text-slate-400 mt-0.5 font-mono">
                Started: {run.startedAt ? new Date(run.startedAt).toLocaleTimeString() : 'Pending'}
              </div>
            </div>
            <div className="text-[11px] text-slate-500 dark:text-slate-400">
              {run.activeDeadlineSeconds
                ? `Max execution deadline: ${run.activeDeadlineSeconds}s`
                : 'Default execution deadline: 3600s'}
            </div>
          </div>
        </div>

        {/* Abort Reason Banner (if aborted) */}
        {run.status === 'ABORTED' && (
          <div className="p-3.5 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800 text-xs text-amber-900 dark:text-amber-200 flex items-start gap-2.5">
            <AlertOctagon className="w-4 h-4 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <span className="font-semibold">Execution Aborted: </span>
              <span>{run.abortReason || 'Manual cancellation request via control plane.'}</span>
            </div>
          </div>
        )}

        {/* Technical Metadata & Kubernetes Context */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/30 border border-slate-200 dark:border-slate-800">
            <div className="text-[11px] text-slate-400 flex items-center gap-1">
              <Layers className="w-3.5 h-3.5" />
              <span>Job Name</span>
            </div>
            <div className="font-mono text-xs font-semibold text-slate-800 dark:text-slate-200 mt-1 truncate">
              {run.k8sJobName || 'vuhive-run-' + run.id}
            </div>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/30 border border-slate-200 dark:border-slate-800">
            <div className="text-[11px] text-slate-400 flex items-center gap-1">
              <Cpu className="w-3.5 h-3.5" />
              <span>Namespace</span>
            </div>
            <div className="font-mono text-xs font-semibold text-slate-800 dark:text-slate-200 mt-1 truncate">
              {run.k8sNamespace || 'vuhive-runners'}
            </div>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/30 border border-slate-200 dark:border-slate-800">
            <div className="text-[11px] text-slate-400">Target Platform</div>
            <div className="font-mono text-xs font-semibold text-slate-800 dark:text-slate-200 mt-1 truncate">
              {run.artifactId ? 'linux/arm64' : 'linux/amd64'}
            </div>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/30 border border-slate-200 dark:border-slate-800">
            <div className="text-[11px] text-slate-400">Execution Result</div>
            <div className="mt-1">
              {run.status === 'COMPLETED' ? (
                <Badge variant={run.slaPassed !== false ? 'success' : 'error'}>
                  {run.slaPassed !== false ? 'SLA Passed' : 'SLA Breached'}
                </Badge>
              ) : run.status === 'RUNNING' ? (
                <Badge variant="warning">Running</Badge>
              ) : run.status === 'ABORTED' ? (
                <Badge variant="error">Aborted</Badge>
              ) : (
                <Badge variant="outline">Queued</Badge>
              )}
            </div>
          </div>
        </div>

        {/* Live / Summary Performance KPIs */}
        {run.metrics && (
          <div className="rounded-xl p-4 bg-slate-50 dark:bg-slate-800/40 border border-slate-200 dark:border-slate-800 space-y-3">
            <div className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Execution Performance KPIs
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              <div>
                <div className="text-[11px] text-slate-400">Throughput</div>
                <div className="text-lg font-mono font-bold text-slate-900 dark:text-white">
                  {run.metrics.avgTps?.toLocaleString() ?? 0} <span className="text-xs font-normal text-slate-400">req/s</span>
                </div>
              </div>

              <div>
                <div className="text-[11px] text-slate-400">p95 Latency</div>
                <div className="text-lg font-mono font-bold text-slate-900 dark:text-white">
                  {run.metrics.p95DurationMs ?? 0} <span className="text-xs font-normal text-slate-400">ms</span>
                </div>
              </div>

              <div>
                <div className="text-[11px] text-slate-400">Error Rate</div>
                <div className="text-lg font-mono font-bold text-emerald-600 dark:text-emerald-400">
                  {((run.metrics.errorRatePct ?? 0) * 100).toFixed(2)}%
                </div>
              </div>

              <div>
                <div className="text-[11px] text-slate-400">Total Requests</div>
                <div className="text-lg font-mono font-bold text-slate-900 dark:text-white">
                  {run.metrics.totalRequests?.toLocaleString() ?? 0}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Abort Confirmation Dialog */}
      <AbortConfirmationDialog
        open={isAbortDialogOpen}
        onOpenChange={setIsAbortDialogOpen}
        run={run}
        onConfirm={handleAbortConfirm}
        isAborting={abortRunMutation.isPending}
      />
    </div>
  )
}
