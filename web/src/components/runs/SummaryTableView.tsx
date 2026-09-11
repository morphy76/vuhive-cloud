import React, { useState, useMemo } from 'react'
import {
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Search,
  Download,
  CheckCircle2,
  AlertTriangle,
  Layers,
  Activity,
  ShieldCheck,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { exportToCsvFile, type CsvColumn } from '@/lib/export-utils'
import type {
  SummaryReport,
  SummaryStepItem,
  SummaryMetricItem,
  SummaryThresholdItem,
} from '@/types/report'

export interface SummaryTableViewProps {
  report: SummaryReport
  runId?: string
  searchQuery?: string
  className?: string
}

type TableTab = 'steps' | 'metrics' | 'thresholds'
type SortDirection = 'asc' | 'desc'

function formatDurationValue(val?: any): string {
  if (val === undefined || val === null) return '-'
  const num = typeof val === 'number' ? val : parseFloat(String(val))
  if (isNaN(num)) return String(val)
  // If value is in nanoseconds (> 1000000), convert to ms
  if (num > 1000000 && Number.isInteger(num)) {
    return `${(num / 1000000).toFixed(2)} ms`
  }
  return `${num.toFixed(1)} ms`
}

function formatNumber(val?: number): string {
  if (val === undefined || val === null) return '-'
  return val.toLocaleString()
}

function formatRate(rate?: number): string {
  if (rate === undefined || rate === null) return '-'
  const pct = rate <= 1.0 && rate > 0 ? rate * 100 : rate
  return `${pct.toFixed(2)}%`
}

// Fallback extractor in case steps are represented within metrics array
function extractSteps(report: SummaryReport): SummaryStepItem[] {
  if (report.steps && Array.isArray(report.steps) && report.steps.length > 0) {
    return report.steps
  }

  // Derive steps from metrics if explicit steps array wasn't provided
  const stepMap = new Map<string, Partial<SummaryStepItem>>()
  const metrics = report.metrics || []

  for (const m of metrics) {
    if (m.name.startsWith('vuhive.step.')) {
      const parts = m.name.split('.')
      const stepName = parts.slice(2, -1).join('.') || parts[2]
      const metricType = parts[parts.length - 1]

      const existing = stepMap.get(stepName) || { name: stepName, requests: 0 }
      if (metricType === 'duration') {
        existing.p50_ms = typeof m.p50 === 'number' ? (m.p50 > 1000000 ? m.p50 / 1000000 : m.p50) : undefined
        existing.p90_ms = typeof m.p90 === 'number' ? (m.p90 > 1000000 ? m.p90 / 1000000 : m.p90) : undefined
        existing.p95_ms = typeof m.p95 === 'number' ? (m.p95 > 1000000 ? m.p95 / 1000000 : m.p95) : undefined
        existing.p99_ms = typeof m.p99 === 'number' ? (m.p99 > 1000000 ? m.p99 / 1000000 : m.p99) : undefined
        if (m.count) existing.requests = m.count
      } else if (metricType === 'failed') {
        existing.failed_requests = m.count || 0
      }
      stepMap.set(stepName, existing)
    }
  }

  if (stepMap.size > 0) {
    return Array.from(stepMap.values()).map((s) => {
      const reqs = s.requests || 1
      const failed = s.failed_requests || 0
      const errPct = (failed / reqs) * 100
      return {
        name: s.name || 'step',
        requests: s.requests || 0,
        tps: s.tps,
        p50_ms: s.p50_ms,
        p90_ms: s.p90_ms,
        p95_ms: s.p95_ms,
        p99_ms: s.p99_ms,
        failed_requests: failed,
        error_rate_pct: errPct,
        status: failed === 0 ? 'PASS' : 'FAILED',
      }
    })
  }

  // Fallback single scenario step if none parsed
  if (report.scenario || report.suite_name) {
    return [
      {
        name: report.scenario || report.suite_name || 'Scenario Execution',
        requests: report.total_requests || 0,
        tps: report.avg_tps,
        p50_ms: report.p50_duration_ms,
        p90_ms: report.p90_duration_ms,
        p95_ms: report.p95_duration_ms,
        p99_ms: report.p99_duration_ms,
        failed_requests: report.error_rate_pct && report.total_requests ? Math.round((report.error_rate_pct / 100) * report.total_requests) : 0,
        error_rate_pct: report.error_rate_pct,
        status: report.passed !== false ? 'PASS' : 'FAILED',
      },
    ]
  }

  return []
}

export const SummaryTableView: React.FC<SummaryTableViewProps> = ({
  report,
  runId = 'run',
  searchQuery: externalSearchQuery,
  className = '',
}) => {
  const [activeTab, setActiveTab] = useState<TableTab>('steps')
  const [internalSearchQuery, setInternalSearchQuery] = useState('')
  const [sortKey, setSortKey] = useState<string>('name')
  const [sortDir, setSortDir] = useState<SortDirection>('asc')

  const activeSearchQuery = externalSearchQuery !== undefined ? externalSearchQuery : internalSearchQuery

  const steps = useMemo(() => extractSteps(report), [report])
  const metrics = useMemo(() => report.metrics || [], [report.metrics])
  const thresholds = useMemo(() => report.thresholds || [], [report.thresholds])

  const handleSort = (key: string) => {
    if (sortKey === key) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  // Filtered and Sorted Steps
  const filteredSteps = useMemo(() => {
    let list = steps
    if (activeSearchQuery) {
      const q = activeSearchQuery.toLowerCase()
      list = list.filter((s) => s.name.toLowerCase().includes(q) || (s.status && s.status.toLowerCase().includes(q)))
    }
    return [...list].sort((a, b) => {
      const valA = (a as any)[sortKey]
      const valB = (b as any)[sortKey]
      if (valA === undefined || valA === null) return 1
      if (valB === undefined || valB === null) return -1
      if (typeof valA === 'number' && typeof valB === 'number') {
        return sortDir === 'asc' ? valA - valB : valB - valA
      }
      return sortDir === 'asc' ? String(valA).localeCompare(String(valB)) : String(valB).localeCompare(String(valA))
    })
  }, [steps, activeSearchQuery, sortKey, sortDir])

  // Filtered and Sorted Metrics
  const filteredMetrics = useMemo(() => {
    let list = metrics
    if (activeSearchQuery) {
      const q = activeSearchQuery.toLowerCase()
      list = list.filter((m) => m.name.toLowerCase().includes(q) || m.type.toLowerCase().includes(q))
    }
    return [...list].sort((a, b) => {
      const valA = (a as any)[sortKey]
      const valB = (b as any)[sortKey]
      if (valA === undefined || valA === null) return 1
      if (valB === undefined || valB === null) return -1
      if (typeof valA === 'number' && typeof valB === 'number') {
        return sortDir === 'asc' ? valA - valB : valB - valA
      }
      return sortDir === 'asc' ? String(valA).localeCompare(String(valB)) : String(valB).localeCompare(String(valA))
    })
  }, [metrics, activeSearchQuery, sortKey, sortDir])

  // Filtered and Sorted Thresholds
  const filteredThresholds = useMemo(() => {
    let list = thresholds
    if (activeSearchQuery) {
      const q = activeSearchQuery.toLowerCase()
      list = list.filter((t) => t.metric.toLowerCase().includes(q) || t.stat.toLowerCase().includes(q))
    }
    return [...list].sort((a, b) => {
      const valA = (a as any)[sortKey]
      const valB = (b as any)[sortKey]
      if (valA === undefined || valA === null) return 1
      if (valB === undefined || valB === null) return -1
      return sortDir === 'asc' ? String(valA).localeCompare(String(valB)) : String(valB).localeCompare(String(valA))
    })
  }, [thresholds, activeSearchQuery, sortKey, sortDir])

  // CSV Export for the active tab
  const handleExportActiveTabCsv = () => {
    if (activeTab === 'steps') {
      const columns: CsvColumn<SummaryStepItem>[] = [
        { key: 'name', header: 'Step / Endpoint Name' },
        { key: 'requests', header: 'Total Requests' },
        { key: 'tps', header: 'Throughput (TPS)', formatter: (v) => (typeof v === 'number' ? v.toFixed(2) : '') },
        { key: 'p50_ms', header: 'p50 Latency (ms)', formatter: (v) => (typeof v === 'number' ? v.toFixed(1) : '') },
        { key: 'p90_ms', header: 'p90 Latency (ms)', formatter: (v) => (typeof v === 'number' ? v.toFixed(1) : '') },
        { key: 'p95_ms', header: 'p95 Latency (ms)', formatter: (v) => (typeof v === 'number' ? v.toFixed(1) : '') },
        { key: 'p99_ms', header: 'p99 Latency (ms)', formatter: (v) => (typeof v === 'number' ? v.toFixed(1) : '') },
        { key: 'failed_requests', header: 'Failed Requests' },
        { key: 'error_rate_pct', header: 'Error Rate %', formatter: (v) => (typeof v === 'number' ? `${v.toFixed(2)}%` : '') },
        { key: 'status', header: 'Status' },
      ]
      exportToCsvFile(columns, filteredSteps, `summary-${runId}-steps.csv`)
    } else if (activeTab === 'metrics') {
      const columns: CsvColumn<SummaryMetricItem>[] = [
        { key: 'name', header: 'Metric Name' },
        { key: 'type', header: 'Metric Type' },
        { key: 'count', header: 'Count' },
        { key: 'rate', header: 'Rate' },
        { key: 'mean', header: 'Mean' },
        { key: 'p50', header: 'p50' },
        { key: 'p90', header: 'p90' },
        { key: 'p95', header: 'p95' },
        { key: 'p99', header: 'p99' },
        { key: 'max', header: 'Max' },
      ]
      exportToCsvFile(columns, filteredMetrics, `summary-${runId}-metrics.csv`)
    } else {
      const columns: CsvColumn<SummaryThresholdItem>[] = [
        { key: 'metric', header: 'Metric Name' },
        { key: 'stat', header: 'Stat' },
        { key: 'operator', header: 'Operator' },
        { key: 'target', header: 'Target' },
        { key: 'actual', header: 'Actual Measured' },
        { key: 'passed', header: 'Passed', formatter: (v) => (v ? 'PASSED' : 'FAILED') },
      ]
      exportToCsvFile(columns, filteredThresholds, `summary-${runId}-thresholds.csv`)
    }
  }

  const renderSortIndicator = (key: string) => {
    if (sortKey !== key) {
      return <ArrowUpDown className="w-3.5 h-3.5 opacity-40 ml-1 inline" />
    }
    return sortDir === 'asc' ? (
      <ArrowUp className="w-3.5 h-3.5 text-brand-600 dark:text-brand-400 ml-1 inline" />
    ) : (
      <ArrowDown className="w-3.5 h-3.5 text-brand-600 dark:text-brand-400 ml-1 inline" />
    )
  }

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Category Tabs & Filter Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 bg-slate-50 dark:bg-slate-900/60 rounded-xl border border-slate-200/80 dark:border-slate-800">
        <div className="flex items-center gap-1.5 p-1 bg-slate-200/70 dark:bg-slate-800 rounded-lg">
          <button
            type="button"
            onClick={() => {
              setActiveTab('steps')
              setSortKey('name')
            }}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
              activeTab === 'steps'
                ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
            }`}
          >
            <Layers className="w-3.5 h-3.5" />
            <span>Scenario Steps</span>
            <Badge variant="outline" className="text-[10px] px-1 py-0 ml-1 font-mono">
              {steps.length}
            </Badge>
          </button>

          <button
            type="button"
            onClick={() => {
              setActiveTab('metrics')
              setSortKey('name')
            }}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
              activeTab === 'metrics'
                ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
            }`}
          >
            <Activity className="w-3.5 h-3.5" />
            <span>Telemetry Metrics</span>
            <Badge variant="outline" className="text-[10px] px-1 py-0 ml-1 font-mono">
              {metrics.length}
            </Badge>
          </button>

          <button
            type="button"
            onClick={() => {
              setActiveTab('thresholds')
              setSortKey('metric')
            }}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
              activeTab === 'thresholds'
                ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs'
                : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
            }`}
          >
            <ShieldCheck className="w-3.5 h-3.5" />
            <span>SLA Thresholds</span>
            <Badge variant="outline" className="text-[10px] px-1 py-0 ml-1 font-mono">
              {thresholds.length}
            </Badge>
          </button>
        </div>

        <div className="flex items-center gap-2">
          {externalSearchQuery === undefined && (
            <div className="relative max-w-xs">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400" />
              <input
                type="text"
                value={internalSearchQuery}
                onChange={(e) => setInternalSearchQuery(e.target.value)}
                placeholder="Filter table rows..."
                className="pl-8 pr-7 py-1 text-xs bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-brand-500 text-slate-900 dark:text-white"
              />
              {internalSearchQuery && (
                <button
                  type="button"
                  onClick={() => setInternalSearchQuery('')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                  aria-label="Clear filter"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </div>
          )}

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleExportActiveTabCsv}
            className="h-8 text-xs gap-1.5 border-slate-200 dark:border-slate-800"
            aria-label="Export CSV"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export CSV</span>
          </Button>
        </div>
      </div>

      {/* 1. SCENARIO STEPS TABLE */}
      {activeTab === 'steps' && (
        <div className="border border-slate-200 dark:border-slate-800 rounded-xl overflow-x-auto shadow-xs">
          <table className="w-full text-xs text-left text-slate-700 dark:text-slate-300">
            <thead className="text-[11px] font-semibold uppercase tracking-wider bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400">
              <tr>
                <th
                  scope="col"
                  className="px-4 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white"
                  onClick={() => handleSort('name')}
                >
                  Step / Endpoint Name {renderSortIndicator('name')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('requests')}
                >
                  Requests {renderSortIndicator('requests')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('tps')}
                >
                  TPS {renderSortIndicator('tps')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p50_ms')}
                >
                  p50 {renderSortIndicator('p50_ms')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p90_ms')}
                >
                  p90 {renderSortIndicator('p90_ms')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p95_ms')}
                >
                  p95 {renderSortIndicator('p95_ms')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p99_ms')}
                >
                  p99 {renderSortIndicator('p99_ms')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('error_rate_pct')}
                >
                  Error Rate {renderSortIndicator('error_rate_pct')}
                </th>
                <th
                  scope="col"
                  className="px-4 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-center"
                  onClick={() => handleSort('status')}
                >
                  Status {renderSortIndicator('status')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800 bg-white dark:bg-slate-900">
              {filteredSteps.length === 0 ? (
                <tr>
                  <td colSpan={9} className="px-4 py-8 text-center text-slate-400">
                    No scenario steps found matching your filter criteria.
                  </td>
                </tr>
              ) : (
                filteredSteps.map((step, idx) => (
                  <tr
                    key={idx}
                    className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
                  >
                    <td className="px-4 py-3 font-semibold font-mono text-slate-900 dark:text-white max-w-xs truncate">
                      {step.name}
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-medium">
                      {formatNumber(step.requests)}
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-medium text-brand-600 dark:text-brand-400">
                      {step.tps !== undefined ? `${step.tps.toFixed(1)} req/s` : '-'}
                    </td>
                    <td className="px-3 py-3 text-right font-mono">
                      {step.p50_ms !== undefined ? `${step.p50_ms.toFixed(1)} ms` : '-'}
                    </td>
                    <td className="px-3 py-3 text-right font-mono">
                      {step.p90_ms !== undefined ? `${step.p90_ms.toFixed(1)} ms` : '-'}
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-medium">
                      {step.p95_ms !== undefined ? `${step.p95_ms.toFixed(1)} ms` : '-'}
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-semibold text-slate-900 dark:text-white">
                      {step.p99_ms !== undefined ? `${step.p99_ms.toFixed(1)} ms` : '-'}
                    </td>
                    <td className="px-3 py-3 text-right font-mono">
                      <span
                        className={
                          (step.error_rate_pct || 0) > 0
                            ? 'text-rose-600 dark:text-rose-400 font-bold'
                            : 'text-slate-400'
                        }
                      >
                        {formatRate(step.error_rate_pct)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Badge
                        variant={step.status === 'PASS' ? 'success' : 'error'}
                        className="text-[10px] px-2 py-0.5"
                      >
                        {step.status || 'PASS'}
                      </Badge>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* 2. METRICS TELEMETRY TABLE */}
      {activeTab === 'metrics' && (
        <div className="border border-slate-200 dark:border-slate-800 rounded-xl overflow-x-auto shadow-xs">
          <table className="w-full text-xs text-left text-slate-700 dark:text-slate-300">
            <thead className="text-[11px] font-semibold uppercase tracking-wider bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400">
              <tr>
                <th
                  scope="col"
                  className="px-4 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white"
                  onClick={() => handleSort('name')}
                >
                  Metric Identifier {renderSortIndicator('name')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white"
                  onClick={() => handleSort('type')}
                >
                  Type {renderSortIndicator('type')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('count')}
                >
                  Count / Value {renderSortIndicator('count')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('mean')}
                >
                  Mean {renderSortIndicator('mean')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p50')}
                >
                  p50 {renderSortIndicator('p50')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p95')}
                >
                  p95 {renderSortIndicator('p95')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('p99')}
                >
                  p99 {renderSortIndicator('p99')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-right"
                  onClick={() => handleSort('max')}
                >
                  Max {renderSortIndicator('max')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800 bg-white dark:bg-slate-900">
              {filteredMetrics.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-4 py-8 text-center text-slate-400">
                    No metric entries found in telemetry report.
                  </td>
                </tr>
              ) : (
                filteredMetrics.map((m, idx) => (
                  <tr
                    key={idx}
                    className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
                  >
                    <td className="px-4 py-3 font-mono font-semibold text-slate-900 dark:text-white">
                      {m.name}
                    </td>
                    <td className="px-3 py-3">
                      <Badge variant="outline" className="font-mono text-[10px]">
                        {m.type}
                      </Badge>
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-medium">
                      {m.count !== undefined
                        ? formatNumber(m.count)
                        : m.value !== undefined
                        ? m.value.toLocaleString()
                        : m.rate !== undefined
                        ? formatRate(m.rate)
                        : '-'}
                    </td>
                    <td className="px-3 py-3 text-right font-mono">
                      {formatDurationValue(m.mean)}
                    </td>
                    <td className="px-3 py-3 text-right font-mono">
                      {formatDurationValue(m.p50)}
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-medium">
                      {formatDurationValue(m.p95)}
                    </td>
                    <td className="px-3 py-3 text-right font-mono font-semibold text-slate-900 dark:text-white">
                      {formatDurationValue(m.p99)}
                    </td>
                    <td className="px-3 py-3 text-right font-mono">
                      {formatDurationValue(m.max)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* 3. SLA THRESHOLDS TABLE */}
      {activeTab === 'thresholds' && (
        <div className="border border-slate-200 dark:border-slate-800 rounded-xl overflow-x-auto shadow-xs">
          <table className="w-full text-xs text-left text-slate-700 dark:text-slate-300">
            <thead className="text-[11px] font-semibold uppercase tracking-wider bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400">
              <tr>
                <th
                  scope="col"
                  className="px-4 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white"
                  onClick={() => handleSort('metric')}
                >
                  Metric Identifier {renderSortIndicator('metric')}
                </th>
                <th
                  scope="col"
                  className="px-3 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white"
                  onClick={() => handleSort('stat')}
                >
                  Stat {renderSortIndicator('stat')}
                </th>
                <th scope="col" className="px-3 py-3">
                  Operator Condition
                </th>
                <th scope="col" className="px-3 py-3">
                  SLA Target Quota
                </th>
                <th scope="col" className="px-3 py-3 font-mono">
                  Actual Result
                </th>
                <th
                  scope="col"
                  className="px-4 py-3 cursor-pointer hover:text-slate-900 dark:hover:text-white text-center"
                  onClick={() => handleSort('passed')}
                >
                  Compliance Status {renderSortIndicator('passed')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800 bg-white dark:bg-slate-900">
              {filteredThresholds.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-400">
                    No SLA thresholds defined or recorded in report.
                  </td>
                </tr>
              ) : (
                filteredThresholds.map((t, idx) => (
                  <tr
                    key={idx}
                    className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
                  >
                    <td className="px-4 py-3 font-mono font-semibold text-slate-900 dark:text-white">
                      {t.metric}
                    </td>
                    <td className="px-3 py-3 font-mono uppercase text-slate-500">
                      {t.stat}
                    </td>
                    <td className="px-3 py-3 font-mono font-semibold text-indigo-600 dark:text-indigo-400">
                      {t.operator}
                    </td>
                    <td className="px-3 py-3 font-mono font-medium">
                      {t.target}
                    </td>
                    <td className="px-3 py-3 font-mono font-bold text-slate-900 dark:text-white">
                      {t.actual}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Badge
                        variant={t.passed ? 'success' : 'error'}
                        className="gap-1 text-[10px] px-2 py-0.5 inline-flex items-center"
                      >
                        {t.passed ? (
                          <>
                            <CheckCircle2 className="w-3 h-3" />
                            <span>PASSED</span>
                          </>
                        ) : (
                          <>
                            <AlertTriangle className="w-3 h-3" />
                            <span>BREACHED</span>
                          </>
                        )}
                      </Badge>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
