import * as React from 'react'
import {
  BarChart3,
  Activity,
  TrendingUp,
  Table as TableIcon,
  LineChart as ChartIcon,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { LatencyPercentileChart } from './LatencyPercentileChart'
import { ThroughputErrorCorrelationChart } from './ThroughputErrorCorrelationChart'
import { HistoricalComparisonChart } from './HistoricalComparisonChart'
import type { HistoricalRun } from '@/types/suite'

export interface VisualAnalyticsSectionProps {
  run: HistoricalRun
}

export const VisualAnalyticsSection: React.FC<VisualAnalyticsSectionProps> = ({ run }) => {
  const [activeTab, setActiveTab] = React.useState<'latency' | 'throughput' | 'historical'>('latency')
  const [isTabularView, setIsTabularView] = React.useState(false)

  return (
    <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xs space-y-6">
      {/* Section Header & View Mode Switch */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-base font-bold text-slate-900 dark:text-white">
              Visual Performance Analytics
            </h3>
            <HelpTooltip
              text="Interactive telemetry visualizations: latency percentile distribution (p50-p99), throughput vs error rate correlation, and multi-run historical regression tracking."
              label="Help for visual performance analytics"
            />
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Statistical distributions, dual-axis load correlation, and longitudinal test regression analysis.
          </p>
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setIsTabularView((prev) => !prev)}
          aria-label="Toggle data tables view"
          className="min-h-[38px] gap-2 border-slate-200 dark:border-slate-800 self-start sm:self-auto text-xs"
        >
          {isTabularView ? (
            <>
              <ChartIcon className="w-4 h-4 text-brand-600 dark:text-brand-400" />
              <span>Switch to Visual Charts</span>
            </>
          ) : (
            <>
              <TableIcon className="w-4 h-4 text-slate-500" />
              <span>Accessible Data Tables</span>
            </>
          )}
        </Button>
      </div>

      {isTabularView ? (
        /* Tabular Telemetry Breakdown Mode */
        <div className="space-y-6">
          <div className="rounded-xl bg-slate-50 dark:bg-slate-800/40 p-4 border border-slate-200/80 dark:border-slate-800">
            <div className="flex items-center gap-2 text-xs font-semibold text-slate-900 dark:text-white mb-2">
              <TableIcon className="w-4 h-4 text-brand-500" />
              <span>Tabular Telemetry Breakdown</span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
              Accessible, high-contrast semantic tables containing exact numerical observations for screen readers and tabular audits.
            </p>

            <div className="space-y-6 divide-y divide-slate-200 dark:divide-slate-800">
              <div className="pt-2 first:pt-0">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400 mb-2">
                  1. Latency Percentiles (p50, p90, p95, p99)
                </h4>
                <LatencyPercentileChart metrics={run.metrics} showTable={true} />
              </div>

              <div className="pt-6">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400 mb-2">
                  2. Throughput & Error Rate Correlation Over Time
                </h4>
                <ThroughputErrorCorrelationChart
                  runDurationMs={run.durationMs}
                  metrics={run.metrics}
                  showTable={true}
                />
              </div>

              <div className="pt-6">
                <h4 className="text-xs font-bold uppercase tracking-wider text-slate-600 dark:text-slate-400 mb-2">
                  3. Historical Latency Comparison (Last 10 Runs)
                </h4>
                <HistoricalComparisonChart
                  suiteId={run.suiteId}
                  currentRunId={run.id}
                  showTable={true}
                />
              </div>
            </div>
          </div>
        </div>
      ) : (
        /* Interactive Recharts Tabs Mode */
        <Tabs
          value={activeTab}
          onValueChange={(val) => setActiveTab(val as any)}
          className="w-full"
        >
          <TabsList className="w-full grid grid-cols-1 sm:grid-cols-3 gap-1 h-auto p-1 bg-slate-100 dark:bg-slate-800 rounded-xl">
            <TabsTrigger
              value="latency"
              className="gap-2 py-2 text-xs"
              aria-label="Latency distribution tab"
            >
              <BarChart3 className="w-3.5 h-3.5 text-teal-600 dark:text-teal-400" />
              <span>Latency Distribution</span>
            </TabsTrigger>

            <TabsTrigger
              value="throughput"
              className="gap-2 py-2 text-xs"
              aria-label="Throughput & error rate tab"
            >
              <Activity className="w-3.5 h-3.5 text-brand-600 dark:text-brand-400" />
              <span>Throughput & Error Rate</span>
            </TabsTrigger>

            <TabsTrigger
              value="historical"
              className="gap-2 py-2 text-xs"
              aria-label="Historical regression trend tab"
            >
              <TrendingUp className="w-3.5 h-3.5 text-indigo-600 dark:text-indigo-400" />
              <span>Historical Regression Trend</span>
            </TabsTrigger>
          </TabsList>

          <TabsContent value="latency" className="pt-4">
            <LatencyPercentileChart metrics={run.metrics} showTable={false} />
          </TabsContent>

          <TabsContent value="throughput" className="pt-4">
            <ThroughputErrorCorrelationChart
              runDurationMs={run.durationMs}
              metrics={run.metrics}
              showTable={false}
            />
          </TabsContent>

          <TabsContent value="historical" className="pt-4">
            <HistoricalComparisonChart
              suiteId={run.suiteId}
              currentRunId={run.id}
              showTable={false}
            />
          </TabsContent>
        </Tabs>
      )}
    </div>
  )
}
