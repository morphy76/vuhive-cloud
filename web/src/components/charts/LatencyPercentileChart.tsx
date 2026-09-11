import * as React from 'react'
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Cell,
} from 'recharts'
import type { RunMetrics } from '@/types/suite'

export interface LatencyPercentileChartProps {
  metrics?: RunMetrics
  showTable?: boolean
}

interface PercentileDataPoint {
  key: string
  label: string
  value: number
  description: string
  color: string
}

const PERCENTILE_CONFIG: {
  key: 'p50' | 'p90' | 'p95' | 'p99'
  label: string
  description: string
  color: string
}[] = [
  { key: 'p50', label: 'p50', description: 'Median Response Time (50% of requests)', color: '#0d9488' }, // teal-600
  { key: 'p90', label: 'p90', description: '90th Percentile Response Time', color: '#6366f1' }, // indigo-500
  { key: 'p95', label: 'p95', description: '95th Percentile (SLA Benchmark)', color: '#f59e0b' }, // amber-500
  { key: 'p99', label: 'p99', description: 'Tail Latency (99th Percentile)', color: '#f43f5e' }, // rose-500
]

export const LatencyPercentileChart: React.FC<LatencyPercentileChartProps> = ({
  metrics,
  showTable = false,
}) => {
  const data: PercentileDataPoint[] = React.useMemo(() => {
    if (!metrics) return []

    const p50 = metrics.p50DurationMs ?? 0
    const p90 = metrics.p90DurationMs ?? 0
    const p95 = metrics.p95DurationMs ?? 0
    const p99 = metrics.p99DurationMs ?? 0

    if (p50 === 0 && p90 === 0 && p95 === 0 && p99 === 0) {
      return []
    }

    return [
      {
        key: 'p50',
        label: 'p50',
        value: Number(p50.toFixed(2)),
        description: PERCENTILE_CONFIG[0].description,
        color: PERCENTILE_CONFIG[0].color,
      },
      {
        key: 'p90',
        label: 'p90',
        value: Number(p90.toFixed(2)),
        description: PERCENTILE_CONFIG[1].description,
        color: PERCENTILE_CONFIG[1].color,
      },
      {
        key: 'p95',
        label: 'p95',
        value: Number(p95.toFixed(2)),
        description: PERCENTILE_CONFIG[2].description,
        color: PERCENTILE_CONFIG[2].color,
      },
      {
        key: 'p99',
        label: 'p99',
        value: Number(p99.toFixed(2)),
        description: PERCENTILE_CONFIG[3].description,
        color: PERCENTILE_CONFIG[3].color,
      },
    ]
  }, [metrics])

  if (!metrics || data.length === 0) {
    return (
      <div
        role="region"
        aria-label="Latency percentile distribution chart"
        className="flex h-64 w-full items-center justify-center rounded-xl border border-dashed border-slate-200 dark:border-slate-800 p-6 text-center text-sm text-slate-500 dark:text-slate-400"
      >
        <span>No latency metrics available for this run.</span>
      </div>
    )
  }

  return (
    <div
      role="region"
      aria-label="Latency percentile distribution chart"
      className="space-y-4"
    >
      <div className="h-72 w-full pt-2" aria-hidden="true">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
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
              dataKey="label"
              tickLine={false}
              stroke="#94a3b8"
              fontSize={12}
              fontWeight={600}
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
                  const item = payload[0].payload as PercentileDataPoint
                  return (
                    <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-3 shadow-lg">
                      <div className="flex items-center gap-2">
                        <span
                          className="h-2.5 w-2.5 rounded-full"
                          style={{ backgroundColor: item.color }}
                        />
                        <span className="font-mono text-xs font-bold text-slate-900 dark:text-white uppercase">
                          {item.label}
                        </span>
                      </div>
                      <div className="font-mono text-base font-extrabold text-slate-900 dark:text-white mt-1">
                        {item.value.toFixed(1)} ms
                      </div>
                      <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5 max-w-[200px]">
                        {item.description}
                      </div>
                    </div>
                  )
                }
                return null
              }}
            />
            <Bar dataKey="value" radius={[6, 6, 0, 0]} maxBarSize={56}>
              {data.map((entry) => (
                <Cell key={`cell-${entry.key}`} fill={entry.color} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      {/* Semantic Accessible Tabular Fallback */}
      <div className={showTable ? 'block mt-4' : 'sr-only'}>
        <table
          aria-label="Latency percentile data"
          className="w-full text-left text-xs border-collapse"
        >
          <caption className="sr-only">
            Detailed response duration metrics measured in milliseconds
          </caption>
          <thead>
            <tr className="border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400 font-semibold">
              <th scope="col" className="py-2 px-3">Percentile</th>
              <th scope="col" className="py-2 px-3">Response Time (ms)</th>
              <th scope="col" className="py-2 px-3">Description</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60 font-mono">
            {data.map((row) => (
              <tr key={row.key} className="hover:bg-slate-50 dark:hover:bg-slate-800/40">
                <td className="py-2 px-3 font-semibold text-slate-900 dark:text-white">
                  {row.label}
                </td>
                <td className="py-2 px-3 text-slate-800 dark:text-slate-200">
                  {row.value.toFixed(1)} ms
                </td>
                <td className="py-2 px-3 font-sans text-slate-500 dark:text-slate-400">
                  {row.description}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
