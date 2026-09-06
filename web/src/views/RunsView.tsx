import React from 'react'
import { PlayCircle, CheckCircle2, Play } from 'lucide-react'

export const RunsView: React.FC = () => {
  const sampleRuns = [
    {
      id: 'run-9f8e7d6c',
      suite: 'Checkout & Payment Stress Test',
      status: 'RUNNING',
      runners: '8 / 8 Pods',
      duration: '04m 12s',
      tps: '1,420 req/s',
      p95: '42ms',
      errorRate: '0.01%',
    },
    {
      id: 'run-3a2b1c0d',
      suite: 'Product Catalog High Throughput',
      status: 'COMPLETED',
      runners: '16 Pods',
      duration: '15m 00s',
      tps: '4,850 req/s',
      p95: '68ms',
      errorRate: '0.00%',
    },
    {
      id: 'run-7b6a5c4d',
      suite: 'OAuth2 Token Grant Barrier Test',
      status: 'COMPLETED',
      runners: '4 Pods',
      duration: '05m 00s',
      tps: '920 req/s',
      p95: '18ms',
      errorRate: '0.00%',
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Execution Runs
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Live & Historical Test Runs: monitor runner pods, rendezvous barrier synchronization, and indexed KPIs.
          </p>
        </div>

        <button
          type="button"
          className="inline-flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold bg-brand-600 hover:bg-brand-700 text-white shadow-sm transition-all min-h-[44px]"
        >
          <Play className="w-4 h-4 fill-current" />
          <span>New Execution</span>
        </button>
      </div>

      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
            <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th className="px-6 py-4">Run Identifier</th>
                <th className="px-6 py-4">Status</th>
                <th className="px-6 py-4">Pods</th>
                <th className="px-6 py-4">Duration</th>
                <th className="px-6 py-4">Throughput</th>
                <th className="px-6 py-4">p95 Latency</th>
                <th className="px-6 py-4">Error Rate</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
              {sampleRuns.map((r) => (
                <tr key={r.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors">
                  <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                    <div className="flex items-center gap-3">
                      <PlayCircle className="w-5 h-5 text-brand-500 flex-shrink-0" />
                      <div>
                        <div>{r.suite}</div>
                        <div className="text-xs text-slate-400 font-mono">{r.id}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    {r.status === 'RUNNING' ? (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-50 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300">
                        <span className="w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
                        RUNNING
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        COMPLETED
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 font-mono text-xs">{r.runners}</td>
                  <td className="px-6 py-4 font-mono text-xs">{r.duration}</td>
                  <td className="px-6 py-4 font-semibold text-slate-900 dark:text-white font-mono text-xs">{r.tps}</td>
                  <td className="px-6 py-4 font-mono text-xs">{r.p95}</td>
                  <td className="px-6 py-4 font-mono text-xs text-emerald-600 dark:text-emerald-400">{r.errorRate}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
