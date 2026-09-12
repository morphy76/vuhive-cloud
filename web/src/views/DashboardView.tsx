import React from 'react'
import {
  Activity,
  Layers,
  CalendarClock,
  CheckCircle2,
  Play,
  Cpu,
  ArrowUpRight,
  AlertTriangle,
  RefreshCw,
  Clock,
  PlayCircle,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { TriggerRunDialog } from '@/components/dialogs/TriggerRunDialog'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import { useDashboard } from '@/hooks/use-dashboard'
import type { RouteId } from '@/types/navigation'
import type { HistoricalRun } from '@/types/suite'

function formatDurationMs(ms?: number): string {
  if (!ms || ms <= 0) return '-'
  const totalSeconds = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${String(minutes).padStart(2, '0')}m ${String(seconds).padStart(2, '0')}s`
}

function renderRunStatusBadge(status: string) {
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
          ABORTED
        </Badge>
      )
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

export const DashboardView: React.FC<{ onNavigate?: (route: RouteId) => void }> = ({ onNavigate }) => {
  const [isRunDialogOpen, setIsRunDialogOpen] = React.useState(false)
  const { data: dashboard, isLoading, isError, error, refetch } = useDashboard()

  const isUp = dashboard?.control_plane_status === 'UP'

  const stats = [
    {
      title: 'Active Runners',
      value: dashboard ? String(dashboard.active_runs_count) : isLoading ? '-' : '0',
      trend: dashboard?.control_plane_status === 'UP' ? 'Live orchestrator' : 'Orchestrator degraded',
      icon: Activity,
      color: 'text-brand-500 bg-brand-50 dark:bg-brand-950/50',
      help: 'Number of currently executing Kubernetes runner pods across all active test runs.',
    },
    {
      title: 'Test Suites',
      value: dashboard ? String(dashboard.suites_count) : isLoading ? '-' : '0',
      trend: `${dashboard?.recent_suites?.length ?? 0} registered recently`,
      icon: Layers,
      color: 'text-indigo-500 bg-indigo-50 dark:bg-indigo-950/50',
      help: 'Configured load test scenarios and cross-compiled execution artifacts.',
    },
    {
      title: 'Cron Schedules',
      value: dashboard ? String(dashboard.active_schedules_count) : isLoading ? '-' : '0',
      trend: 'Recurring automated jobs',
      icon: CalendarClock,
      color: 'text-amber-500 bg-amber-50 dark:bg-amber-950/50',
      help: 'Native Kubernetes CronJobs orchestrating automated recurring test executions.',
    },
    {
      title: 'SLA Pass Rate',
      value: dashboard ? `${dashboard.sla_pass_rate.toFixed(1)}%` : isLoading ? '-' : '100.0%',
      trend: `Across ${dashboard?.total_runs_count ?? 0} runs`,
      icon: CheckCircle2,
      color: 'text-emerald-500 bg-emerald-50 dark:bg-emerald-950/50',
      help: 'Percentage of test runs satisfying all latency percentiles (p50, p90, p95, p99) and error rate SLAs.',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header section */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Control Plane Overview
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Real-time cluster orchestrator health, runner workload status, and performance telemetry.
          </p>
        </div>

        {/* Quick action buttons */}
        <div className="flex items-center gap-3">
          <OfflinePreviewBadge />
          <Button
            variant="outline"
            onClick={() => onNavigate?.('runs')}
            className="min-h-[44px] gap-2"
          >
            <Activity className="w-4 h-4" />
            <span>View Runs</span>
          </Button>
          <Button
            onClick={() => setIsRunDialogOpen(true)}
            className="min-h-[44px] gap-2"
          >
            <Play className="w-4 h-4 fill-current" />
            <span>Trigger Run</span>
          </Button>
        </div>
      </div>

      {/* Error state banner */}
      {isError && (
        <div className="p-4 rounded-xl border border-red-200 bg-red-50 dark:bg-red-950/40 dark:border-red-800/60 flex items-center justify-between text-red-700 dark:text-red-300">
          <div className="flex items-center gap-3">
            <AlertTriangle className="w-5 h-5 flex-shrink-0 text-red-500" />
            <div>
              <p className="text-sm font-semibold">Failed to load dashboard telemetry</p>
              <p className="text-xs text-red-600/80 dark:text-red-400/80">
                {(error as Error)?.message || 'Unable to communicate with the Go BFF gateway.'}
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            className="border-red-300 dark:border-red-700 hover:bg-red-100 dark:hover:bg-red-900/50 text-xs gap-1.5"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            <span>Retry</span>
          </Button>
        </div>
      )}

      {/* KPI Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, i) => {
          const Icon = stat.icon
          return (
            <div
              key={i}
              className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <span className="text-sm font-medium text-slate-500 dark:text-slate-400">
                    {stat.title}
                  </span>
                  <HelpTooltip text={stat.help} label={`Help for ${stat.title}`} />
                </div>
                <div className={`p-2 rounded-xl ${stat.color}`}>
                  <Icon className="w-4 h-4" />
                </div>
              </div>
              <div className="mt-4">
                <div className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white">
                  {stat.value}
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  {stat.trend}
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {/* Orchestrator Architecture Status */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kubernetes Engine Card */}
        <div className="lg:col-span-2 p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Cpu className="w-5 h-5 text-brand-500" />
              <h2 className="text-base font-semibold text-slate-900 dark:text-white">
                Cluster Execution Pipeline
              </h2>
            </div>
            <Badge variant={isUp ? 'success' : 'error'}>
              {isUp ? 'Synchronized' : 'Degraded'}
            </Badge>
          </div>

          <div className="space-y-3">
            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/60 dark:border-slate-800 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-1.5 text-sm font-medium text-slate-900 dark:text-white">
                  <span>Ephemeral Compilation Subsystem</span>
                  <HelpTooltip
                    text="Cross-compiles Go source packages with AST validation into static binaries for linux/amd64 and linux/arm64."
                    label="Help for compilation subsystem"
                  />
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  Target architectures: linux/amd64, linux/arm64
                </div>
              </div>
              <Badge variant={isUp ? 'success' : 'warning'}>{isUp ? 'Ready' : 'Degraded'}</Badge>
            </div>

            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/60 dark:border-slate-800 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-1.5 text-sm font-medium text-slate-900 dark:text-white">
                  <span>Distributed Start Barrier Rendezvous</span>
                  <HelpTooltip
                    text="Zero clock-skew distributed rendezvous barrier coordinating simultaneous test execution across all runner pods."
                    label="Help for distributed start barrier"
                  />
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  Zero clock-skew distributed synchronized firing
                </div>
              </div>
              <Badge variant={isUp ? 'success' : 'warning'}>{isUp ? 'Active' : 'Offline'}</Badge>
            </div>

            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/60 dark:border-slate-800 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-1.5 text-sm font-medium text-slate-900 dark:text-white">
                  <span>Telemetry & KPI Indexer</span>
                  <HelpTooltip
                    text="Extracts summary.json reports to index p50, p90, p95, p99 percentiles, TPS throughput, and SLA compliance into PostgreSQL."
                    label="Help for telemetry indexer"
                  />
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  summary.json parser (p50, p90, p95, p99, TPS, SLA)
                </div>
              </div>
              <Badge variant="success">Listening</Badge>
            </div>
          </div>
        </div>

        {/* Quick Links & Documentation Card */}
        <div className="p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-2">
              API & Spec Documentation
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
              Explore interactive endpoints, OpenAPI 3.1 definitions, and control plane recipes.
            </p>

            <TooltipProvider>
              <div className="space-y-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <a
                      href="/api/openapi.yaml"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center justify-between p-3 rounded-xl hover:bg-slate-50 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-800 transition-colors text-xs font-medium text-slate-700 dark:text-slate-300"
                    >
                      <span>OpenAPI 3.1 Specification</span>
                      <ArrowUpRight className="w-4 h-4 text-slate-400" />
                    </a>
                  </TooltipTrigger>
                  <TooltipContent>
                    View the full OpenAPI 3.1 specification for the Control Plane API
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      onClick={() => onNavigate?.('cookbook')}
                      className="w-full flex items-center justify-between p-3 rounded-xl hover:bg-slate-50 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-800 transition-colors text-xs font-medium text-slate-700 dark:text-slate-300 text-left cursor-pointer"
                    >
                      <span>Control Plane Cookbook</span>
                      <ArrowUpRight className="w-4 h-4 text-slate-400" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>
                    Read the in-app developer cookbook for advanced configuration recipes
                  </TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          </div>

          <div className="mt-6 pt-4 border-t border-slate-200 dark:border-slate-800 text-[11px] text-slate-400">
            vuhive-cloud • Reactive Go BFF & Embedded React 19 PWA
          </div>
        </div>
      </div>

      {/* Recent Run Activity Section */}
      <div className="p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-brand-500" />
            <h2 className="text-base font-semibold text-slate-900 dark:text-white">
              Recent Run Activity
            </h2>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onNavigate?.('runs')}
            className="text-xs font-medium text-brand-600 dark:text-brand-400 gap-1.5"
            aria-label="View All Runs"
          >
            <span>View All Runs</span>
            <ArrowUpRight className="w-4 h-4" />
          </Button>
        </div>

        {dashboard?.recent_runs && dashboard.recent_runs.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
              <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
                <tr>
                  <th scope="col" className="px-4 py-3">Run Identifier</th>
                  <th scope="col" className="px-4 py-3">Status</th>
                  <th scope="col" className="px-4 py-3">Duration</th>
                  <th scope="col" className="px-4 py-3">Avg TPS</th>
                  <th scope="col" className="px-4 py-3">p95 Latency</th>
                  <th scope="col" className="px-4 py-3 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                {dashboard.recent_runs.map((run: HistoricalRun) => (
                  <tr
                    key={run.id}
                    onClick={() => onNavigate?.('runs')}
                    className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors cursor-pointer"
                  >
                    <td className="px-4 py-3 font-medium text-slate-900 dark:text-white">
                      <div className="flex items-center gap-2">
                        <PlayCircle className="w-4 h-4 text-brand-500 flex-shrink-0" />
                        <span className="font-mono text-xs">{run.id}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      {renderRunStatusBadge(run.status)}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {run.status === 'RUNNING' || run.status === 'QUEUED'
                        ? 'Active...'
                        : formatDurationMs(run.durationMs)}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs font-semibold text-slate-900 dark:text-white">
                      {run.metrics?.avgTps ? `${run.metrics.avgTps.toLocaleString()} req/s` : '-'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {run.metrics?.p95DurationMs !== undefined ? `${run.metrics.p95DurationMs}ms` : '-'}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          onNavigate?.('runs')
                        }}
                        className="text-xs text-brand-600 dark:text-brand-400 h-8 px-2"
                      >
                        Inspect
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="py-8 text-center text-xs text-slate-500 dark:text-slate-400">
            No execution runs recorded yet. Trigger a run or configure a schedule to start testing.
          </div>
        )}
      </div>

      <TriggerRunDialog
        open={isRunDialogOpen}
        onOpenChange={setIsRunDialogOpen}
      />
    </div>
  )
}
