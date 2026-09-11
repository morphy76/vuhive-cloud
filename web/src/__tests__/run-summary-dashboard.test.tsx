import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import * as React from 'react'
import { axe } from 'vitest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RunSummaryDashboard } from '@/components/runs/RunSummaryDashboard'
import { TooltipProvider } from '@/components/ui/tooltip'
import { RecipeProvider } from '@/context/RecipeContext'
import { api } from '@/lib/api'
import type { HistoricalRun } from '@/types/suite'
import type { RunnerProfile } from '@/types/profile'

const mockProfile: RunnerProfile = {
  id: 'profile-high-perf',
  name: 'high-perf-runner',
  description: 'High performance dedicated test runner',
  runner_image: 'alpine:3.20',
  cpu_request: '2000m',
  cpu_limit: '4000m',
  memory_request: '4Gi',
  memory_limit: '8Gi',
  node_selector: {
    'node-role.kubernetes.io/performance-runner': 'true',
  },
  affinity: {
    node_selector_terms: [
      {
        key: 'node.kubernetes.io/instance-type',
        operator: 'In',
        values: ['c5.4xlarge', 'c6i.4xlarge'],
      },
    ],
  },
  tolerations: [
    {
      key: 'dedicated',
      operator: 'Equal',
      value: 'loadgen',
      effect: 'NoSchedule',
    },
  ],
  active_deadline_seconds: 7200,
  runtime_class_name: null,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
}

const mockArtifacts = [
  {
    id: 'art-123-arm64',
    suiteId: 'suite-checkout',
    platform: 'linux/arm64',
    sha256Checksum: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
    status: 'READY' as const,
    createdAt: new Date().toISOString(),
  },
]

const completedPassedRun: HistoricalRun = {
  id: 'run-pass-101',
  suiteId: 'suite-checkout',
  artifactId: 'art-123-arm64',
  runnerProfileId: 'profile-high-perf',
  status: 'COMPLETED',
  k8sJobName: 'vuhive-run-run-pass-101',
  k8sNamespace: 'vuhive-runners',
  startedAt: '2026-09-11T14:00:00Z',
  finishedAt: '2026-09-11T14:15:00Z',
  durationMs: 900000, // 15m 00s
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
    errorRatePct: 0.0,
  },
}

const failedRun: HistoricalRun = {
  id: 'run-fail-202',
  suiteId: 'suite-checkout',
  artifactId: 'art-123-arm64',
  runnerProfileId: 'profile-high-perf',
  status: 'FAILED',
  k8sJobName: 'vuhive-run-run-fail-202',
  k8sNamespace: 'vuhive-runners',
  startedAt: '2026-09-11T15:00:00Z',
  finishedAt: '2026-09-11T15:05:30Z',
  durationMs: 330000, // 05m 30s
  exitCode: 1,
  slaPassed: false,
  createdAt: '2026-09-11T14:59:50Z',
  metrics: {
    totalIterations: 12000,
    totalRequests: 36000,
    avgTps: 1200,
    p50DurationMs: 85,
    p90DurationMs: 190,
    p95DurationMs: 320,
    p99DurationMs: 750,
    errorRatePct: 0.082, // 8.2% error rate (critical)
  },
}

const warningRun: HistoricalRun = {
  ...completedPassedRun,
  id: 'run-warn-303',
  metrics: {
    ...completedPassedRun.metrics,
    errorRatePct: 0.024, // 2.4% error rate (warning threshold)
  },
}

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

describe('RunSummaryDashboard', () => {
  beforeEach(() => {
    vi.spyOn(api, 'getProfiles').mockResolvedValue([mockProfile])
    vi.spyOn(api, 'getProfile').mockResolvedValue(mockProfile)
    vi.spyOn(api, 'getSuiteArtifacts').mockResolvedValue(mockArtifacts)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('SLA Compliance Banner', () => {
    it('renders prominent SLA PASSED banner with exit code and run duration', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      const banner = screen.getByRole('status')
      expect(banner).toBeInTheDocument()
      expect(within(banner).getByText('SLA PASSED')).toBeInTheDocument()
      expect(within(banner).getByText(/Exit Code:\s*0/i)).toBeInTheDocument()
      expect(within(banner).getByText(/15m 00s/i)).toBeInTheDocument()
    })

    it('renders prominent SLA FAILED banner with exit code and run duration on failure', async () => {
      renderWithProviders(<RunSummaryDashboard run={failedRun} />)

      const banner = screen.getByRole('status')
      expect(banner).toBeInTheDocument()
      expect(within(banner).getByText('SLA FAILED')).toBeInTheDocument()
      expect(within(banner).getByText(/Exit Code:\s*1/i)).toBeInTheDocument()
      expect(within(banner).getByText(/05m 30s/i)).toBeInTheDocument()
    })

    it('is accessible to screen readers with clear status and label', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      const banner = screen.getByRole('status')
      expect(banner).toHaveAttribute('aria-label', expect.stringContaining('SLA PASSED'))
    })
  })

  describe('KPI Metric Cards', () => {
    it('displays Total Iterations and Total Requests with thousands separator', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      expect(screen.getByText('50,000')).toBeInTheDocument()
      expect(screen.getByText('Total Iterations')).toBeInTheDocument()
      expect(screen.getByText('150,000')).toBeInTheDocument()
      expect(screen.getByText(/Total Requests/i)).toBeInTheDocument()
    })

    it('displays Average Throughput (TPS)', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      expect(screen.getByText('Average Throughput')).toBeInTheDocument()
      expect(screen.getByText(/3,500.5/)).toBeInTheDocument()
      expect(screen.getByText(/req\/s/i)).toBeInTheDocument()
    })

    it('displays Error Rate with normal color coding for 0% error', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      expect(screen.getByText('Error Rate')).toBeInTheDocument()
      expect(screen.getByText('0.00%')).toBeInTheDocument()
      const badge = screen.getByText('Normal')
      expect(badge).toBeInTheDocument()
    })

    it('displays Error Rate with warning threshold color coding for >0% and <5%', async () => {
      renderWithProviders(<RunSummaryDashboard run={warningRun} />)

      expect(screen.getByText('2.40%')).toBeInTheDocument()
      const warningBadge = screen.getByText('Warning')
      expect(warningBadge).toBeInTheDocument()
    })

    it('displays Error Rate with critical threshold color coding for >=5%', async () => {
      renderWithProviders(<RunSummaryDashboard run={failedRun} />)

      expect(screen.getByText('8.20%')).toBeInTheDocument()
      const criticalBadge = screen.getByText('Critical')
      expect(criticalBadge).toBeInTheDocument()
    })

    it('displays Latency Percentiles summary badges (p50, p90, p95, p99)', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      expect(screen.getByText(/Latency Percentiles/i)).toBeInTheDocument()
      expect(screen.getByText(/p50/i)).toBeInTheDocument()
      expect(screen.getByText(/25.4\s*ms/i)).toBeInTheDocument()
      expect(screen.getByText(/p90/i)).toBeInTheDocument()
      expect(screen.getByText(/42.1\s*ms/i)).toBeInTheDocument()
      expect(screen.getByText(/p95/i)).toBeInTheDocument()
      expect(screen.getByText(/58.7\s*ms/i)).toBeInTheDocument()
      expect(screen.getByText(/p99/i)).toBeInTheDocument()
      expect(screen.getByText(/89.2\s*ms/i)).toBeInTheDocument()
    })
  })

  describe('Execution Metadata', () => {
    it('displays runner profile used, Kubernetes node info, artifact checksum, and timestamps', async () => {
      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      await waitFor(() => {
        // Runner profile
        expect(screen.getByText(/high-perf-runner/i)).toBeInTheDocument()
        // K8s job & namespace
        expect(screen.getByText('vuhive-run-run-pass-101')).toBeInTheDocument()
        expect(screen.getByText('vuhive-runners')).toBeInTheDocument()
        // K8s node requirements / instance type
        expect(screen.getByText(/c5.4xlarge/i)).toBeInTheDocument()
        // Artifact checksum
        expect(screen.getByText(/9f86d081/i)).toBeInTheDocument()
        // Start & finish timestamps
        expect(screen.getByText(/Started/i)).toBeInTheDocument()
        expect(screen.getByText(/Finished/i)).toBeInTheDocument()
      })
    })

    it('allows copying artifact checksum to clipboard', async () => {
      const writeTextMock = vi.fn().mockResolvedValue(undefined)
      Object.assign(navigator, {
        clipboard: {
          writeText: writeTextMock,
        },
      })

      renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)

      await waitFor(() => {
        expect(screen.getByText(/9f86d081/i)).toBeInTheDocument()
      })

      const copyBtn = screen.getByRole('button', { name: /copy checksum/i })
      fireEvent.click(copyBtn)

      expect(writeTextMock).toHaveBeenCalledWith('9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08')
    })
  })

  describe('Zero Accessibility Violations (WCAG 2.1 AA)', () => {
    it('has zero accessibility violations for passed run', async () => {
      const { container } = renderWithProviders(<RunSummaryDashboard run={completedPassedRun} />)
      await waitFor(() => {
        expect(screen.getByText('SLA PASSED')).toBeInTheDocument()
      })
      const results = await axe(container)
      expect(results).toHaveNoViolations()
    })

    it('has zero accessibility violations for failed run', async () => {
      const { container } = renderWithProviders(<RunSummaryDashboard run={failedRun} />)
      await waitFor(() => {
        expect(screen.getByText('SLA FAILED')).toBeInTheDocument()
      })
      const results = await axe(container)
      expect(results).toHaveNoViolations()
    })
  })
})
