import * as React from 'react'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ReferenceDot,
} from 'recharts'
import { useSuiteRuns } from '@/hooks/use-suites'
import type { HistoricalRun } from '@/types/suite'

export interface HistoricalComparisonChartProps {
  suiteId: string
  currentRunId?: string
  showTable?: boolean
}

interface HistoricalRunDataPoint {
  runId: string
  shortId: string
  timestamp: string
  dateLabel: string
  status: string
  isCurrent: boolean
  slaPassed: boolean
  p50: number
  p90: number
  p95: number
  p99: number
}

export const HistoricalComparisonChart: React.FC<HistoricalComparisonChartProps> = ({
  suiteId,
  currentRunId,
  showTable = false,
}) => {
  const { data: runs = [], isLoading } = useSuiteRuns(suiteId)

  const data: HistoricalRunDataPoint[] = React.useMemo(() => {
    if (!runs || runs.length === 0) return []

    // Filter to runs that have metrics and sort chronologically ascending
    const sorted = [...runs]
      .filter((r) => r.metrics && (r.metrics.p50DurationMs || r.metrics.p95DurationMs))
      .sort((a, b) => {
        const timeA = new Date(a.createdAt || a.startedAt || 0).getTime()
        const timeB = new Date(b.createdAt || b.startedAt || 0).getTime()
        return timeA - timeB
      })
      .slice(-10) // Last 10 runs

    return sorted.map((r: HistoricalRun) => {
      const d = new Date(r.createdAt || r.startedAt || Date.now())
      const dateLabel = `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
      return {
        runId: r.id,
        shortId: r.id.length > 8 ? r.id.slice(-8) : r.id,
        timestamp: d.toISOString(),
        dateLabel,
        status: r.status,
        isCurrent: r.id === currentRunId,
        slaPassed: r.slaPassed ?? (r.status === 'COMPLETED'),
        p50: Number((r.metrics?.p50DurationMs ?? 0).toFixed(1)),
        p90: Number((r.metrics?.p90DurationMs ?? 0).toFixed(1)),
        p95: Number((r.metrics?.p95DurationMs ?? 0).toFixed(1)),
        p99: Number((r.metrics?.p99DurationMs ?? 0).toFixed(1)),
      }
    })
  }, [runs, currentRunId])

  const currentPoint = React.useMemo(() => {
    return data.find((p) => p.isCurrent)
  }, [data])

  if (isLoading) {
    return (
      <div
        role="region"
        aria-label="Historical latency comparison chart"
        className="flex h-64 w-full items-center justify-center rounded-xl border border-dashed border-slate-200 dark:border-slate-800 p-6 text-sm text-slate-500"
      >
        <span>Loading historical test run telemetry...</span>
      </div>
    )
  }

  if (data.length === 0) {
    return (
      <div
        role="region"
        aria-label="Historical latency comparison chart"
        className="flex h-64 w-full items-center justify-center rounded-xl border border-dashed border-slate-200 dark:border-slate-800 p-6 text-center text-sm text-slate-500 dark:text-slate-400"
      >
        <span>No historical execution runs with indexed metrics found for this suite.</span>
      </div>
    )
  }

  return (
    <div
      role="region"
      aria-label="Historical latency comparison chart"
      className="space-y-4"
    >
      <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
        <span>Displaying last {data.length} test runs (chronological trend)</span>
        {currentPoint && (
          <span className="flex items-center gap-1.5 font-semibold text-brand-600 dark:text-brand-400">
            <span className="h-2 w-2 rounded-full bg-brand-500" />
            Current Run: {currentPoint.shortId}
          </span>
        )}
      </div>

      <div className="h-72 w-full pt-2" aria-hidden="true">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            data={data}
            margin={{ top: 10, right: 20, left: 0, bottom: 20 }}
          >
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="#94a3b8"
              opacity={0.25}
              vertical={false}
            />
            <XAxis
              dataKey="dateLabel"
              tickLine={false}
              stroke="#94a3b8"
              fontSize={10}
            />
            <YAxis
              tickLine={false}
              stroke="#94a3b8"
              fontSize={11}
              tickFormatter={(v) => `${v}ms`}
            />
            <Tooltip
              content={({ active, payload }) => {
                if (active && payload && payload.length) {
                  const pt = payload[0].payload as HistoricalRunDataPoint
                  return (
                    <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-3.5 shadow-lg font-mono text-xs">
                      <div className="flex items-center justify-between gap-3 border-b border-slate-100 dark:border-slate-800 pb-2">
                        <span className="font-bold text-slate-900 dark:text-white">
                          Run {pt.shortId}
                        </span>
                        <span
                          className={`px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase ${
                            pt.slaPassed
                              ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300'
                              : 'bg-rose-100 text-rose-800 dark:bg-rose-950 dark:text-rose-300'
                          }`}
                        >
                          {pt.slaPassed ? 'SLA OK' : 'SLA FAIL'}
                        </span>
                      </div>
                      <div className="text-[11px] text-slate-400 mt-1.5 font-sans">
                        {pt.dateLabel} ({pt.runId})
                      </div>
                      <div className="mt-2.5 grid grid-cols-2 gap-2 text-[11px]">
                        <div className="text-teal-600 dark:text-teal-400">
                          p50: <span className="font-bold">{pt.p50} ms</span>
                        </div>
                        <div className="text-indigo-600 dark:text-indigo-400">
                          p90: <span className="font-bold">{pt.p90} ms</span>
                        </div>
                        <div className="text-amber-600 dark:text-amber-400">
                          p95: <span className="font-bold">{pt.p95} ms</span>
                        </div>
                        <div className="text-rose-600 dark:text-rose-400">
                          p99: <span className="font-bold">{pt.p99} ms</span>
                        </div>
                      </div>
                      {pt.isCurrent && (
                        <div className="mt-2 pt-1.5 border-t border-slate-100 dark:border-slate-800 text-[10px] font-sans font-semibold text-brand-600 dark:text-brand-400">
                          ★ Currently Selected Run
                        </div>
                      )}
                    </div>
                  )
                }
                return null
              }}
            />
            <Legend
              verticalAlign="top"
              align="right"
              wrapperStyle={{ paddingBottom: '8px', fontSize: '11px' }}
            />
            <Line
              type="monotone"
              dataKey="p50"
              name="p50 (ms)"
              stroke="#0d9488"
              strokeWidth={2}
              dot={{ r: 3 }}
            />
            <Line
              type="monotone"
              dataKey="p90"
              name="p90 (ms)"
              stroke="#6366f1"
              strokeWidth={2}
              dot={{ r: 3 }}
            />
            <Line
              type="monotone"
              dataKey="p95"
              name="p95 (ms)"
              stroke="#f59e0b"
              strokeWidth={2.5}
              dot={{ r: 4 }}
            />
            <Line
              type="monotone"
              dataKey="p99"
              name="p99 (ms)"
              stroke="#f43f5e"
              strokeWidth={2}
              strokeDasharray="3 3"
              dot={{ r: 3 }}
            />
            {currentPoint && (
              <ReferenceDot
                x={currentPoint.dateLabel}
                y={currentPoint.p95}
                r={6}
                fill="#f59e0b"
                stroke="#ffffff"
                strokeWidth={2}
              />
            )}
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Semantic Accessible Tabular Fallback */}
      <div className={showTable ? 'block mt-4 overflow-x-auto' : 'sr-only'}>
        <table
          aria-label="Historical latency comparison data"
          className="w-full text-left text-xs border-collapse"
        >
          <caption className="sr-only">
            Historical response latency comparison across the last 10 test runs
          </caption>
          <thead>
            <tr className="border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400 font-semibold">
              <th scope="col" className="py-2 px-3">Run ID</th>
              <th scope="col" className="py-2 px-3">Date</th>
              <th scope="col" className="py-2 px-3">Status</th>
              <th scope="col" className="py-2 px-3">p50 (ms)</th>
              <th scope="col" className="py-2 px-3">p90 (ms)</th>
              <th scope="col" className="py-2 px-3">p95 (ms)</th>
              <th scope="col" className="py-2 px-3">p99 (ms)</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-mono">
            {data.map((row) => (
              <tr
                key={row.runId}
                className={
                  row.isCurrent
                    ? 'bg-brand-50/50 dark:bg-brand-950/30 font-semibold'
                    : 'hover:bg-slate-50 dark:hover:bg-slate-800/40'
                }
              >
                <td className="py-2 px-3 font-semibold text-slate-900 dark:text-white">
                  <span>{row.runId}</span>
                  {row.isCurrent && (
                    <span className="ml-1.5 text-[10px] text-brand-600 dark:text-brand-400 font-normal">
                      (current)
                    </span>
                  )}
                </td>
                <td className="py-2 px-3 font-sans text-slate-600 dark:text-slate-400">
                  {row.dateLabel}
                </td>
                <td className="py-2 px-3 font-sans">
                  <span
                    className={`px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase ${
                      row.slaPassed
                        ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300'
                        : 'bg-rose-100 text-rose-800 dark:bg-rose-950 dark:text-rose-300'
                    }`}
                  >
                    {row.status}
                  </span>
                </td>
                <td className="py-2 px-3 text-teal-600 dark:text-teal-400">{row.p50} ms</td>
                <td className="py-2 px-3 text-indigo-600 dark:text-indigo-400">{row.p90} ms</td>
                <td className="py-2 px-3 text-amber-600 dark:text-amber-400 font-bold">{row.p95} ms</td>
                <td className="py-2 px-3 text-rose-600 dark:text-rose-400">{row.p99} ms</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
