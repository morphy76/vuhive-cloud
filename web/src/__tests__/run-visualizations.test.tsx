import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import * as React from 'react'
import { axe } from 'vitest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { RecipeProvider } from '@/context/RecipeContext'
import { api } from '@/lib/api'
import type { HistoricalRun } from '@/types/suite'
import { LatencyPercentileChart } from '@/components/charts/LatencyPercentileChart'
import { ThroughputErrorCorrelationChart } from '@/components/charts/ThroughputErrorCorrelationChart'
import { HistoricalComparisonChart } from '@/components/charts/HistoricalComparisonChart'
import { VisualAnalyticsSection } from '@/components/charts/VisualAnalyticsSection'
import { RunSummaryDashboard } from '@/components/runs/RunSummaryDashboard'

const currentRun: HistoricalRun = {
  id: 'run-curr-100',
  suiteId: 'suite-payment',
  artifactId: 'art-arm64-1',
  runnerProfileId: 'profile-perf',
  status: 'COMPLETED',
  k8sJobName: 'vuhive-run-curr-100',
  k8sNamespace: 'vuhive-runners',
  startedAt: '2026-09-11T14:00:00Z',
  finishedAt: '2026-09-11T14:15:00Z',
  durationMs: 900000,
  exitCode: 0,
  slaPassed: true,
  createdAt: '2026-09-11T13:59:50Z',
  metrics: {
    totalIterations: 50000,
    totalRequests: 150000,
    avgTps: 3500.5,
    p50DurationMs: 25.4,
    p90DurationMs: 42.1,
    p95DurationMs: 58.7,
    p99DurationMs: 89.2,
    errorRatePct: 0.015,
  },
}

const pastRuns: HistoricalRun[] = [
  {
    id: 'run-past-090',
    suiteId: 'suite-payment',
    status: 'COMPLETED',
    startedAt: '2026-09-01T10:00:00Z',
    finishedAt: '2026-09-01T10:15:00Z',
    durationMs: 900000,
    exitCode: 0,
    slaPassed: true,
    createdAt: '2026-09-01T09:59:50Z',
    metrics: {
      totalIterations: 45000,
      totalRequests: 135000,
      avgTps: 3200,
      p50DurationMs: 22.0,
      p90DurationMs: 38.0,
      p95DurationMs: 52.0,
      p99DurationMs: 78.0,
      errorRatePct: 0.005,
    },
  },
  {
    id: 'run-past-091',
    suiteId: 'suite-payment',
    status: 'COMPLETED',
    startedAt: '2026-09-02T10:00:00Z',
    finishedAt: '2026-09-02T10:15:00Z',
    durationMs: 900000,
    exitCode: 0,
    slaPassed: true,
    createdAt: '2026-09-02T09:59:50Z',
    metrics: {
      totalIterations: 48000,
      totalRequests: 144000,
      avgTps: 3400,
      p50DurationMs: 24.0,
      p90DurationMs: 40.0,
      p95DurationMs: 55.0,
      p99DurationMs: 82.0,
      errorRatePct: 0.008,
    },
  },
  {
    id: 'run-past-092',
    suiteId: 'suite-payment',
    status: 'FAILED',
    startedAt: '2026-09-03T10:00:00Z',
    finishedAt: '2026-09-03T10:15:00Z',
    durationMs: 900000,
    exitCode: 1,
    slaPassed: false,
    createdAt: '2026-09-03T09:59:50Z',
    metrics: {
      totalIterations: 30000,
      totalRequests: 90000,
      avgTps: 1800,
      p50DurationMs: 65.0,
      p90DurationMs: 140.0,
      p95DurationMs: 210.0,
      p99DurationMs: 450.0,
      errorRatePct: 0.065,
    },
  },
  currentRun,
]

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
    },
  })
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createTestQueryClient()
  return {
    ...render(
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delayDuration={0}>
          <RecipeProvider>{ui}</RecipeProvider>
        </TooltipProvider>
      </QueryClientProvider>
    ),
    queryClient,
  }
}

describe('Recharts Performance Visualizations (Issue #73)', () => {
  beforeEach(() => {
    vi.spyOn(api, 'getRuns').mockResolvedValue(pastRuns)
    vi.spyOn(api, 'getSuiteRuns').mockResolvedValue(pastRuns)
    vi.spyOn(api, 'getProfiles').mockResolvedValue([])
    vi.spyOn(api, 'getSuiteArtifacts').mockResolvedValue([])
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('1. LatencyPercentileChart', () => {
    it('renders latency percentiles p50, p90, p95, p99 with millisecond values', () => {
      renderWithProviders(<LatencyPercentileChart metrics={currentRun.metrics} />)

      // ARIA landmark / region
      const chartRegion = screen.getByRole('region', { name: /latency percentile distribution/i })
      expect(chartRegion).toBeInTheDocument()

      // Should include tabular fallback accessible to screen readers
      const table = screen.getByRole('table', { name: /latency percentile data/i })
      expect(table).toBeInTheDocument()

      expect(within(table).getByText('p50')).toBeInTheDocument()
      expect(within(table).getByText('25.4 ms')).toBeInTheDocument()
      expect(within(table).getByText('p90')).toBeInTheDocument()
      expect(within(table).getByText('42.1 ms')).toBeInTheDocument()
      expect(within(table).getByText('p95')).toBeInTheDocument()
      expect(within(table).getByText('58.7 ms')).toBeInTheDocument()
      expect(within(table).getByText('p99')).toBeInTheDocument()
      expect(within(table).getByText('89.2 ms')).toBeInTheDocument()
    })

    it('handles missing or undefined metrics gracefully', () => {
      renderWithProviders(<LatencyPercentileChart metrics={undefined} />)
      expect(screen.getByText(/no latency metrics available/i)).toBeInTheDocument()
    })
  })

  describe('2. ThroughputErrorCorrelationChart', () => {
    it('renders dual-axis throughput (TPS) and error rate correlation chart with tabular fallback', () => {
      renderWithProviders(
        <ThroughputErrorCorrelationChart
          runDurationMs={currentRun.durationMs}
          metrics={currentRun.metrics}
        />
      )

      const region = screen.getByRole('region', { name: /throughput and error rate correlation/i })
      expect(region).toBeInTheDocument()

      const table = screen.getByRole('table', { name: /throughput and error rate data/i })
      expect(table).toBeInTheDocument()

      // Primary TPS axis column and Error Rate % axis column
      expect(within(table).getByText(/throughput \(req\/s\)/i)).toBeInTheDocument()
      expect(within(table).getByText(/error rate \(%\)/i)).toBeInTheDocument()
    })
  })

  describe('3. HistoricalComparisonChart', () => {
    it('queries and renders trend comparison across historical runs of test suite', async () => {
      renderWithProviders(
        <HistoricalComparisonChart
          suiteId={currentRun.suiteId}
          currentRunId={currentRun.id}
        />
      )

      const region = screen.getByRole('region', { name: /historical latency comparison/i })
      expect(region).toBeInTheDocument()

      await waitFor(() => {
        const table = screen.getByRole('table', { name: /historical latency comparison data/i })
        expect(table).toBeInTheDocument()
        expect(within(table).getByText('run-past-090')).toBeInTheDocument()
        expect(within(table).getByText('run-past-092')).toBeInTheDocument()
        expect(within(table).getByText(currentRun.id)).toBeInTheDocument()
      })
    })
  })

  describe('4. VisualAnalyticsSection Container & Toggle', () => {
    it('allows toggling between Visual Charts and Accessible Data Tables views', async () => {
      renderWithProviders(<VisualAnalyticsSection run={currentRun} />)

      expect(screen.getByText(/Visual Performance Analytics/i)).toBeInTheDocument()

      // Toggle button for tabular view
      const viewToggle = screen.getByRole('button', { name: /toggle data tables view/i })
      expect(viewToggle).toBeInTheDocument()

      // Default shows charts tabs
      expect(screen.getByRole('tab', { name: /latency distribution/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /throughput & error rate/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /historical regression trend/i })).toBeInTheDocument()

      // Click toggle to show data table view
      fireEvent.click(viewToggle)
      expect(screen.getByText(/Tabular Telemetry Breakdown/i)).toBeInTheDocument()
    })
  })

  describe('5. Integration in RunSummaryDashboard', () => {
    it('embeds visual analytics inside RunSummaryDashboard', async () => {
      renderWithProviders(<RunSummaryDashboard run={currentRun} />)

      await waitFor(() => {
        expect(screen.getByText(/Visual Performance Analytics/i)).toBeInTheDocument()
        expect(screen.getByRole('tab', { name: /latency distribution/i })).toBeInTheDocument()
      })
    })
  })

  describe('6. Zero Accessibility Violations (WCAG 2.1 AA)', () => {
    it('has zero accessibility violations in VisualAnalyticsSection', async () => {
      const { container } = renderWithProviders(<VisualAnalyticsSection run={currentRun} />)
      await waitFor(() => {
        expect(screen.getByText(/Visual Performance Analytics/i)).toBeInTheDocument()
      })
      const results = await axe(container)
      expect(results).toHaveNoViolations()
    })
  })
})
