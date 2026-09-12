import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import * as React from 'react'
import { axe } from 'vitest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { DashboardView } from '@/views/DashboardView'
import { api, type DashboardData } from '@/lib/api'

const mockDashboardData: DashboardData = {
  bff_status: 'UP',
  bff_version: 'v0.1.0',
  control_plane_status: 'UP',
  control_plane_version: 'v0.1.0',
  active_runs_count: 7,
  suites_count: 14,
  recent_suites: [
    {
      id: 'suite-1',
      name: 'E-Commerce Peak Stress',
      description: 'Checkout funnel load test',
      state: 'ACTIVE',
      created_at: new Date().toISOString(),
    },
  ],
  profiles_count: 4,
  profiles_summary: [
    {
      id: 'prof-1',
      name: 'standard-runner',
      runner_image: 'alpine:3.20',
    },
  ],
  active_schedules_count: 5,
  recent_runs: [
    {
      id: 'run-recent-1',
      suiteId: 'suite-1',
      artifactId: 'art-1',
      runnerProfileId: 'prof-1',
      status: 'COMPLETED',
      k8sJobName: 'job-recent-1',
      durationMs: 45000,
      exitCode: 0,
      slaPassed: true,
      metrics: {
        totalIterations: 1000,
        totalRequests: 5000,
        avgTps: 150,
        p50DurationMs: 15,
        p90DurationMs: 25,
        p95DurationMs: 35,
        p99DurationMs: 50,
        errorRatePct: 0.0,
      },
      createdAt: new Date().toISOString(),
    },
    {
      id: 'run-recent-2',
      suiteId: 'suite-1',
      artifactId: 'art-1',
      runnerProfileId: 'prof-1',
      status: 'RUNNING',
      k8sJobName: 'job-recent-2',
      createdAt: new Date().toISOString(),
    },
  ],
  sla_pass_rate: 98.7,
  total_runs_count: 32,
  timestamp: new Date().toISOString(),
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>{ui}</TooltipProvider>
    </QueryClientProvider>
  )
}

describe('DashboardView Component', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders live KPI metrics from useDashboard', async () => {
    vi.spyOn(api, 'getDashboard').mockResolvedValue(mockDashboardData)

    renderWithProviders(<DashboardView />)

    await waitFor(() => {
      // Control plane overview header
      expect(screen.getByText('Control Plane Overview')).toBeInTheDocument()
      // Active runners metric: 7
      expect(screen.getByText('7')).toBeInTheDocument()
      // Test suites metric: 14
      expect(screen.getByText('14')).toBeInTheDocument()
      // Cron schedules metric: 5
      expect(screen.getByText('5')).toBeInTheDocument()
      // SLA pass rate metric: 98.7%
      expect(screen.getByText('98.7%')).toBeInTheDocument()
    })

    // Orchestrator status synchronized badge
    expect(screen.getByText('Synchronized')).toBeInTheDocument()
  })

  it('renders Degraded badge when control_plane_status is DOWN', async () => {
    vi.spyOn(api, 'getDashboard').mockResolvedValue({
      ...mockDashboardData,
      control_plane_status: 'DOWN',
    })

    renderWithProviders(<DashboardView />)

    await waitFor(() => {
      expect(screen.getAllByText('Degraded').length).toBeGreaterThan(0)
    })
  })

  it('renders recent run activity table and navigates to runs view', async () => {
    const onNavigate = vi.fn()
    vi.spyOn(api, 'getDashboard').mockResolvedValue(mockDashboardData)

    renderWithProviders(<DashboardView onNavigate={onNavigate} />)

    await waitFor(() => {
      expect(screen.getByText('Recent Run Activity')).toBeInTheDocument()
      expect(screen.getByText('run-recent-1')).toBeInTheDocument()
      expect(screen.getByText('run-recent-2')).toBeInTheDocument()
    })

    // Click on View All Runs or Inspect run
    const viewAllBtn = screen.getByRole('button', { name: /view all runs/i })
    fireEvent.click(viewAllBtn)
    expect(onNavigate).toHaveBeenCalledWith('runs')
  })

  it('renders empty state when there are no recent runs', async () => {
    vi.spyOn(api, 'getDashboard').mockResolvedValue({
      ...mockDashboardData,
      recent_runs: [],
    })

    renderWithProviders(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText(/no execution runs recorded yet/i)).toBeInTheDocument()
    })
  })

  it('renders error banner with retry button on API failure', async () => {
    const getDashboardSpy = vi.spyOn(api, 'getDashboard')
      .mockRejectedValueOnce(new Error('Network error: BFF gateway unreachable'))
      .mockResolvedValueOnce(mockDashboardData)

    renderWithProviders(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText(/failed to load dashboard telemetry/i)).toBeInTheDocument()
    })

    const retryBtn = screen.getByRole('button', { name: /retry/i })
    fireEvent.click(retryBtn)

    await waitFor(() => {
      expect(screen.getByText('98.7%')).toBeInTheDocument()
    })
    expect(getDashboardSpy).toHaveBeenCalledTimes(2)
  })

  it('satisfies basic accessibility contracts without critical violations', async () => {
    vi.spyOn(api, 'getDashboard').mockResolvedValue(mockDashboardData)

    const { container } = renderWithProviders(<DashboardView />)

    await waitFor(() => {
      expect(screen.getByText('Control Plane Overview')).toBeInTheDocument()
    })

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})
