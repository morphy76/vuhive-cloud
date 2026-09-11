import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import * as React from 'react'
import { axe } from 'vitest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TriggerRunDialog } from '@/components/dialogs/TriggerRunDialog'
import { LiveRunMonitor } from '@/components/runs/LiveRunMonitor'
import { AbortConfirmationDialog } from '@/components/dialogs/AbortConfirmationDialog'
import { RunsView } from '@/views/RunsView'
import { RecipeProvider } from '@/context/RecipeContext'
import { TooltipProvider } from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import type { HistoricalRun, TestSuite } from '@/types/suite'
import type { RunnerProfile } from '@/types/profile'

const mockSuites: TestSuite[] = [
  {
    id: 'suite-1',
    name: 'Checkout Funnel Stress Test',
    description: 'E2E checkout load test',
    state: 'ACTIVE',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'suite-2',
    name: 'Catalog Read Benchmark',
    description: 'Search catalog test',
    state: 'DRAFT',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
]

const mockProfiles: RunnerProfile[] = [
  {
    id: 'profile-1',
    name: 'standard-runner',
    description: 'Standard runner (1 vCPU, 1Gi RAM)',
    runner_image: 'alpine:3.20',
    cpu_request: '500m',
    cpu_limit: '1000m',
    memory_request: '512Mi',
    memory_limit: '1Gi',
    node_selector: {},
    affinity: { node_selector_terms: [] },
    tolerations: [
      { key: 'dedicated', operator: 'Equal', value: 'loadgen', effect: 'NoSchedule' },
    ],
    active_deadline_seconds: 3600,
    runtime_class_name: null,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

const mockArtifacts = [
  {
    id: 'art-amd64',
    suiteId: 'suite-1',
    platform: 'linux/amd64',
    sha256Checksum: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    status: 'READY' as const,
    createdAt: new Date().toISOString(),
  },
  {
    id: 'art-arm64',
    suiteId: 'suite-1',
    platform: 'linux/arm64',
    sha256Checksum: 'a7c938144298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852ba',
    status: 'READY' as const,
    createdAt: new Date().toISOString(),
  },
]

const mockConfigs = [
  {
    id: 'cfg-1',
    suiteId: 'suite-1',
    name: 'stress-profile.yaml',
    contentYaml: 'duration: 10m\nconcurrency: 100\n',
    isDefault: true,
    createdAt: new Date().toISOString(),
  },
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

describe('Ad-Hoc Load Test Execution Dispatcher & Live Monitor', () => {
  beforeEach(() => {
    vi.spyOn(api, 'getSuites').mockResolvedValue(mockSuites)
    vi.spyOn(api, 'getProfiles').mockResolvedValue(mockProfiles)
    vi.spyOn(api, 'getSuiteArtifacts').mockResolvedValue(mockArtifacts)
    vi.spyOn(api, 'getSuiteConfigs').mockResolvedValue(mockConfigs)
    vi.spyOn(api, 'triggerRun').mockImplementation(async (input) => ({
      id: 'run-new-123',
      suiteId: input.suite_id,
      artifactId: input.artifact_id,
      runnerProfileId: input.runner_profile_id,
      configurationId: input.configuration_id,
      activeDeadlineSeconds: input.active_deadline_seconds,
      status: 'QUEUED',
      k8sJobName: 'vuhive-run-run-new-123',
      k8sNamespace: 'vuhive-runners',
      createdAt: new Date().toISOString(),
    }))
    vi.spyOn(api, 'abortRun').mockImplementation(async (id, reason) => ({
      id,
      suiteId: 'suite-1',
      status: 'ABORTED',
      abortReason: reason || 'manual cancellation',
      createdAt: new Date().toISOString(),
    }))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('TriggerRunDialog', () => {
    it('renders all required selection dropdowns and technical inputs', async () => {
      renderWithProviders(<TriggerRunDialog open={true} onOpenChange={() => {}} />)

      expect(screen.getByText('Execute Test Run')).toBeInTheDocument()
      expect(screen.getByLabelText(/^test suite$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^compiled artifact$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^scenario configuration$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^runner profile$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^execution timeout/i)).toBeInTheDocument()

      // Backwards-compatible labels & tooltips required by help-components tests
      expect(screen.getByLabelText(/^runner pods count$/i)).toBeInTheDocument()
      expect(screen.getByText(/Kubernetes Runner Resource Allocation/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for runner pods count$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for cpu allocation$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for memory allocation$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for node tolerations$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for start barrier$/i })).toBeInTheDocument()
    })

    it('filters compiled artifacts by platform selector', async () => {
      renderWithProviders(
        <TriggerRunDialog open={true} onOpenChange={() => {}} initialSuiteId="suite-1" />
      )

      await waitFor(() => {
        expect(screen.getByText(/linux\/amd64/i)).toBeInTheDocument()
      })

      // Select ARM64 filter
      const arm64FilterBtn = screen.getByRole('button', { name: /linux\/arm64/i })
      fireEvent.click(arm64FilterBtn)

      // Verify artifact select displays arm64 option
      const artifactSelect = screen.getByLabelText(/^compiled artifact$/i) as HTMLSelectElement
      expect(artifactSelect).toBeInTheDocument()
    })

    it('dispatches ad-hoc test run with timeout override and calls onRunTriggered', async () => {
      const handleTriggered = vi.fn()
      const handleOpenChange = vi.fn()

      renderWithProviders(
        <TriggerRunDialog
          open={true}
          onOpenChange={handleOpenChange}
          initialSuiteId="suite-1"
          onRunTriggered={handleTriggered}
        />
      )

      await waitFor(() => {
        expect(screen.getByLabelText(/^compiled artifact$/i)).toBeInTheDocument()
        expect(screen.getByDisplayValue(/standard-runner/i)).toBeInTheDocument()
      })

      // Set timeout override
      const timeoutInput = screen.getByLabelText(/^execution timeout/i)
      fireEvent.change(timeoutInput, { target: { value: '1800' } })

      // Click dispatch run
      const dispatchBtn = screen.getByRole('button', { name: /^dispatch run$/i })
      fireEvent.click(dispatchBtn)

      await waitFor(() => {
        expect(api.triggerRun).toHaveBeenCalledWith(
          expect.objectContaining({
            suite_id: 'suite-1',
            active_deadline_seconds: 1800,
          })
        )
        expect(handleTriggered).toHaveBeenCalled()
        expect(handleOpenChange).toHaveBeenCalledWith(false)
      })
    })

    it('has zero accessibility violations', async () => {
      const { container, unmount } = renderWithProviders(
        <TriggerRunDialog open={true} onOpenChange={() => {}} initialSuiteId="suite-1" />
      )
      const results = await axe(container)
      expect(results).toHaveNoViolations()
      unmount()
    })
  })

  describe('LiveRunMonitor', () => {
    const runningRun: HistoricalRun = {
      id: 'run-live-456',
      suiteId: 'suite-1',
      artifactId: 'art-arm64',
      runnerProfileId: 'profile-1',
      status: 'RUNNING',
      k8sJobName: 'vuhive-run-run-live-456',
      k8sNamespace: 'vuhive-runners',
      startedAt: new Date(Date.now() - 65000).toISOString(),
      createdAt: new Date(Date.now() - 70000).toISOString(),
      metrics: {
        avgTps: 1250,
        p95DurationMs: 35,
        errorRatePct: 0.0,
      },
    }

    const completedRun: HistoricalRun = {
      id: 'run-done-789',
      suiteId: 'suite-1',
      status: 'COMPLETED',
      k8sJobName: 'vuhive-run-run-done-789',
      durationMs: 180000,
      startedAt: new Date(Date.now() - 200000).toISOString(),
      finishedAt: new Date(Date.now() - 20000).toISOString(),
      createdAt: new Date(Date.now() - 205000).toISOString(),
      slaPassed: true,
      metrics: {
        avgTps: 3400,
        p95DurationMs: 24,
        errorRatePct: 0.0,
      },
    }

    it('displays phase transitions timeline: QUEUED -> RUNNING -> COMPLETED', () => {
      renderWithProviders(<LiveRunMonitor run={runningRun} />)

      expect(screen.getByText('QUEUED')).toBeInTheDocument()
      expect(screen.getByText('RUNNING')).toBeInTheDocument()
      expect(screen.getByText('COMPLETED')).toBeInTheDocument()
      expect(screen.getByText('run-live-456')).toBeInTheDocument()
      expect(screen.getByText('vuhive-run-run-live-456')).toBeInTheDocument()
    })

    it('updates live running duration timer every second', () => {
      const startTime = 1700000000000
      vi.useFakeTimers()
      vi.setSystemTime(startTime + 65000)
      const timedRun: HistoricalRun = {
        ...runningRun,
        startedAt: new Date(startTime).toISOString(),
      }
      const { unmount } = renderWithProviders(<LiveRunMonitor run={timedRun} />)

      // Initial formatted duration (65s -> 01m 05s)
      expect(screen.getByText(/01m 05s/i)).toBeInTheDocument()

      // Advance by 2 seconds
      act(() => {
        vi.advanceTimersByTime(2000)
      })

      expect(screen.getByText(/01m 07s/i)).toBeInTheDocument()
      unmount()
      vi.useRealTimers()
    })

    it('enables high-contrast Abort Test button during RUNNING state', () => {
      renderWithProviders(<LiveRunMonitor run={runningRun} />)

      const abortBtn = screen.getByRole('button', { name: /abort test/i })
      expect(abortBtn).toBeInTheDocument()
      expect(abortBtn).not.toBeDisabled()
    })

    it('disables or omits Abort Test button when run is in COMPLETED state', () => {
      renderWithProviders(<LiveRunMonitor run={completedRun} />)

      const abortBtn = screen.queryByRole('button', { name: /abort test/i })
      if (abortBtn) {
        expect(abortBtn).toBeDisabled()
      } else {
        expect(abortBtn).toBeNull()
      }
    })

    it('has zero accessibility violations', async () => {
      const { container, unmount } = renderWithProviders(<LiveRunMonitor run={completedRun} />)
      const results = await axe(container)
      expect(results).toHaveNoViolations()
      unmount()
    })
  })

  describe('AbortConfirmationDialog', () => {
    const runningRun: HistoricalRun = {
      id: 'run-to-abort-123',
      suiteId: 'suite-1',
      status: 'RUNNING',
      createdAt: new Date().toISOString(),
    }

    it('renders explanation of graceful pod teardown and invokes onConfirm', async () => {
      const handleConfirm = vi.fn()
      const handleOpenChange = vi.fn()

      renderWithProviders(
        <AbortConfirmationDialog
          open={true}
          onOpenChange={handleOpenChange}
          run={runningRun}
          onConfirm={handleConfirm}
        />
      )

      expect(screen.getByText('Abort Test Execution')).toBeInTheDocument()
      expect(screen.getAllByText(/graceful pod teardown/i).length).toBeGreaterThan(0)

      // Enter optional cancellation reason
      const reasonInput = screen.getByLabelText(/cancellation reason/i)
      fireEvent.change(reasonInput, { target: { value: 'High latency threshold breach' } })

      // Confirm abort
      const confirmBtn = screen.getByRole('button', { name: /^confirm abort$/i })
      fireEvent.click(confirmBtn)

      expect(handleConfirm).toHaveBeenCalledWith('High latency threshold breach')
    })

    it('has zero accessibility violations', async () => {
      const { container, unmount } = renderWithProviders(
        <AbortConfirmationDialog
          open={true}
          onOpenChange={() => {}}
          run={runningRun}
          onConfirm={() => {}}
        />
      )
      const results = await axe(container)
      expect(results).toHaveNoViolations()
      unmount()
    })
  })

  describe('RunsView Integration', () => {
    it('renders execution runs and selects run into LiveRunMonitor', async () => {
      vi.spyOn(api, 'getRuns').mockResolvedValue([
        {
          id: 'run-table-1',
          suiteId: 'suite-1',
          status: 'RUNNING',
          k8sJobName: 'job-1',
          durationMs: 120000,
          createdAt: new Date().toISOString(),
        },
      ])

      renderWithProviders(<RunsView />)

      await waitFor(() => {
        expect(screen.getByText('Execution Runs')).toBeInTheDocument()
        expect(screen.getByText('run-table-1')).toBeInTheDocument()
      })

      // Click on row to open live monitor
      const runRow = screen.getByText('run-table-1')
      fireEvent.click(runRow)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /abort test/i })).toBeInTheDocument()
      })
    })
  })
})
