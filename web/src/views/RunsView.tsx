import React, { useState } from 'react'
import { PlayCircle, Play, BookOpen, Clock, AlertTriangle, AlertOctagon, CheckCircle2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VisuallyHidden } from '@/components/ui/visually-hidden'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { TriggerRunDialog } from '@/components/dialogs/TriggerRunDialog'
import { LiveRunMonitor } from '@/components/runs/LiveRunMonitor'
import { RunSummaryDashboard } from '@/components/runs/RunSummaryDashboard'
import { SummaryReportInspector } from '@/components/runs/SummaryReportInspector'
import { VirtualizedLogViewer } from '@/components/logs/VirtualizedLogViewer'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import { useRecipe } from '@/context/RecipeContext'
import { useRuns, useRunLogs } from '@/hooks/use-runs'
import { useSuites } from '@/hooks/use-suites'
import { useRunEvents } from '@/hooks/use-events'
import type { HistoricalRun } from '@/types/suite'

function formatDurationMs(ms?: number): string {
  if (!ms || ms <= 0) return '-'
  const totalSeconds = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${String(minutes).padStart(2, '0')}m ${String(seconds).padStart(2, '0')}s`
}

export const RunsView: React.FC = () => {
  const [isRunDialogOpen, setIsRunDialogOpen] = useState(false)
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null)
  const { openRecipe } = useRecipe()

  const { data: runs = [] } = useRuns()
  const { data: suites = [] } = useSuites()

  // Subscribe to live SSE status updates
  useRunEvents()

  const suiteMap = React.useMemo(() => {
    const map = new Map<string, string>()
    suites.forEach((s) => map.set(s.id, s.name))
    return map
  }, [suites])

  const selectedRun = React.useMemo(() => {
    if (!selectedRunId) return null
    return runs.find((r) => r.id === selectedRunId) || null
  }, [runs, selectedRunId])

  const [inspectorTab, setInspectorTab] = useState<'summary' | 'monitor' | 'logs' | 'report'>('summary')

  const isActive = selectedRun?.status === 'RUNNING' || selectedRun?.status === 'QUEUED'
  const { data: runLogs = '', isLoading: isLogsLoading } = useRunLogs(
    selectedRun?.id,
    isActive ? 3000 : false
  )

  // Automatically reset to summary tab when selecting a new terminal run, or monitor for running runs
  React.useEffect(() => {
    if (!selectedRun) return
    if (selectedRun.status === 'RUNNING' || selectedRun.status === 'QUEUED') {
      setInspectorTab('monitor')
    } else {
      setInspectorTab('summary')
    }
  }, [selectedRun?.id, selectedRun?.status])

  const handleRunTriggered = (newRun: HistoricalRun) => {
    setSelectedRunId(newRun.id)
    setInspectorTab('monitor')
  }

  const renderStatusBadge = (status: string) => {
    switch (status) {
      case 'RUNNING':
        return (
          <Badge variant="warning" className="gap-1.5 font-mono text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse" />
            RUNNING
          </Badge>
        )
      case 'QUEUED':
        return (
          <Badge variant="outline" className="gap-1.5 font-mono text-[11px] text-brand-600 dark:text-brand-400 border-brand-200 dark:border-brand-800">
            <Clock className="w-3 h-3 animate-spin" />
            QUEUED
          </Badge>
        )
      case 'COMPLETED':
        return (
          <Badge variant="success" className="gap-1 font-mono text-[11px]">
            <CheckCircle2 className="w-3 h-3" />
            COMPLETED
          </Badge>
        )
      case 'FAILED':
        return (
          <Badge variant="error" className="gap-1 font-mono text-[11px]">
            <AlertTriangle className="w-3 h-3" />
            FAILED
          </Badge>
        )
      case 'ABORTED':
        return (
          <Badge variant="error" className="gap-1 font-mono text-[11px] bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200 border border-slate-300 dark:border-slate-700">
            <AlertOctagon className="w-3 h-3 text-amber-500" />
            ABORTED
          </Badge>
        )
      default:
        return <Badge variant="outline">{status}</Badge>
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Execution Runs
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Live & Historical Test Runs: monitor runner pods, rendezvous barrier synchronization, and indexed KPIs.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <OfflinePreviewBadge />
          <Button
            variant="outline"
            onClick={() => openRecipe('recipe-4')}
            className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
            aria-label="View Runs API Recipe"
          >
            <BookOpen className="w-4 h-4 text-brand-600 dark:text-brand-400" />
            <span className="hidden sm:inline">API Recipe</span>
          </Button>
          <Button
            onClick={() => setIsRunDialogOpen(true)}
            className="min-h-[44px] gap-2"
            aria-label="New Execution"
          >
            <Play className="w-4 h-4 fill-current" />
            <span>New Execution</span>
          </Button>
        </div>
      </div>

      {/* Selected Run Inspection: Executive Summary or Live Execution Monitor */}
      {selectedRun && (
        <div className="space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-1">
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold text-slate-500 dark:text-slate-400">
                ACTIVE INSPECTION
              </span>
              <span className="font-mono text-xs text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                {selectedRun.id}
              </span>

              {/* View mode toggle */}
              <div className="flex items-center bg-slate-100 dark:bg-slate-800 p-0.5 rounded-lg ml-2">
                <button
                  type="button"
                  onClick={() => setInspectorTab('summary')}
                  className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                    inspectorTab === 'summary'
                      ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                      : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  Executive Summary
                </button>
                <button
                  type="button"
                  onClick={() => setInspectorTab('monitor')}
                  className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                    inspectorTab === 'monitor'
                      ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                      : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  Execution Details
                </button>
                <button
                  type="button"
                  onClick={() => setInspectorTab('logs')}
                  className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                    inspectorTab === 'logs'
                      ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                      : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  Execution Logs
                </button>
                <button
                  type="button"
                  onClick={() => setInspectorTab('report')}
                  className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                    inspectorTab === 'report'
                      ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                      : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  Telemetry Report
                </button>
              </div>
            </div>

            <button
              type="button"
              onClick={() => setSelectedRunId(null)}
              className="text-brand-600 hover:text-brand-700 dark:text-brand-400 text-xs font-medium cursor-pointer self-start sm:self-auto"
            >
              Close Inspector
            </button>
          </div>

          {inspectorTab === 'summary' ? (
            <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs p-6">
              <RunSummaryDashboard
                run={selectedRun}
                onClose={() => setSelectedRunId(null)}
                onViewLogs={() => setInspectorTab('logs')}
                onViewReport={() => setInspectorTab('report')}
              />
            </div>
          ) : inspectorTab === 'monitor' ? (
            <LiveRunMonitor
              run={selectedRun}
              onClose={() => setSelectedRunId(null)}
              onRunAborted={() => {}}
              onViewSummary={() => setInspectorTab('summary')}
              onViewLogs={() => setInspectorTab('logs')}
            />
          ) : inspectorTab === 'logs' ? (
            <VirtualizedLogViewer
              logs={runLogs}
              runId={selectedRun.id}
              title="Container Execution Logs (run.log)"
              subtitle={`Status: ${selectedRun.status} • Job: ${selectedRun.k8sJobName || 'vuhive-runners'}`}
              isLoading={isLogsLoading}
              onClose={() => setSelectedRunId(null)}
            />
          ) : (
            <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs p-6">
              <SummaryReportInspector
                runId={selectedRun.id}
                onClose={() => setSelectedRunId(null)}
              />
            </div>
          )}
        </div>
      )}

      {/* Runs Table */}
      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
            <caption>
              <VisuallyHidden>Execution runs with performance metrics and status</VisuallyHidden>
            </caption>
            <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th scope="col" className="px-6 py-4">Run Identifier</th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Status</span>
                    <HelpTooltip
                      text="Execution state of the test run (QUEUED, RUNNING, COMPLETED, FAILED, ABORTED)."
                      label="Help for execution status column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Job / Pods</span>
                    <HelpTooltip
                      text="Kubernetes Job name and runner pod workload assignment."
                      label="Help for pods column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Duration</span>
                    <HelpTooltip
                      text="Total elapsed wall-clock duration of the load-testing execution."
                      label="Help for duration column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Throughput</span>
                    <HelpTooltip
                      text="Transactions Per Second (TPS): average rate of successfully completed HTTP requests per second."
                      label="Help for throughput column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>p95 Latency</span>
                    <HelpTooltip
                      text="95th percentile latency: 95% of requests finished within this response time."
                      label="Help for p95 latency column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Error Rate</span>
                    <HelpTooltip
                      text="Percentage of failed HTTP transactions (HTTP 5xx status codes or network timeouts)."
                      label="Help for error rate column"
                    />
                  </div>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
              {runs.map((r) => {
                const suiteName = suiteMap.get(r.suiteId) || r.suiteId
                const isSelected = r.id === selectedRunId

                return (
                  <tr
                    key={r.id}
                    onClick={() => setSelectedRunId(r.id)}
                    className={`hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors cursor-pointer ${
                      isSelected ? 'bg-brand-50/40 dark:bg-brand-950/20' : ''
                    }`}
                  >
                    <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                      <div className="flex items-center gap-3">
                        <PlayCircle className="w-5 h-5 text-brand-500 flex-shrink-0" />
                        <div>
                          <div>{suiteName}</div>
                          <div className="text-xs text-slate-400 font-mono">{r.id}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {renderStatusBadge(r.status)}
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">
                      {r.k8sJobName || 'vuhive-runners'}
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">
                      {r.status === 'RUNNING' || r.status === 'QUEUED'
                        ? 'Active...'
                        : formatDurationMs(r.durationMs)}
                    </td>
                    <td className="px-6 py-4 font-semibold text-slate-900 dark:text-white font-mono text-xs">
                      {r.metrics?.avgTps ? `${r.metrics.avgTps.toLocaleString()} req/s` : '-'}
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">
                      {r.metrics?.p95DurationMs !== undefined ? `${r.metrics.p95DurationMs}ms` : '-'}
                    </td>
                    <td className="px-6 py-4 font-mono text-xs text-emerald-600 dark:text-emerald-400">
                      {r.metrics?.errorRatePct !== undefined
                        ? `${(r.metrics.errorRatePct * 100).toFixed(2)}%`
                        : '-'}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

      <TriggerRunDialog
        open={isRunDialogOpen}
        onOpenChange={setIsRunDialogOpen}
        onRunTriggered={handleRunTriggered}
      />
    </div>
  )
}
