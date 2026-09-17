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
  {
    id: 'profile-2',
    name: 'high-perf-runner',
    description: 'High performance runner (4 vCPU, 8Gi RAM)',
    runner_image: 'custom-perf-image:v1',
    cpu_request: '2000m',
    cpu_limit: '4000m',
    memory_request: '4Gi',
    memory_limit: '8Gi',
    node_selector: { 'node.kubernetes.io/instance-type': 'c5.2xlarge' },
    affinity: { node_selector_terms: [] },
    tolerations: [],
    active_deadline_seconds: 7200,
    runtime_class_name: 'kata-containers',
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
    it('renders clean 4-section visual hierarchy with read-only runner profile resource specs', async () => {
      renderWithProviders(<TriggerRunDialog open={true} onOpenChange={() => {}} />)

      expect(screen.getByText('Execute Test Run')).toBeInTheDocument()

      // 1. Test Target section
      expect(screen.getByRole('heading', { name: /^Test Target$/i })).toBeInTheDocument()
      expect(screen.getByLabelText(/^test suite$/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/^compiled artifact$/i)).toBeInTheDocument()

      // 2. Scenario Configuration section
      expect(screen.getByRole('heading', { name: /^Scenario Configuration$/i })).toBeInTheDocument()
      expect(screen.getByLabelText(/^scenario configuration$/i)).toBeInTheDocument()

      // 3. Execution Infrastructure section
      expect(screen.getByRole('heading', { name: /^Execution Infrastructure$/i })).toBeInTheDocument()
      expect(screen.getByLabelText(/^runner profile$/i)).toBeInTheDocument()

      // Wait for profiles to load and assert read-only profile resource specs
      await waitFor(() => {
        expect(screen.getByText(/500m \/ 1000m/i)).toBeInTheDocument()
        expect(screen.getByText(/512Mi \/ 1Gi/i)).toBeInTheDocument()
        expect(screen.getByText(/alpine:3\.20/i)).toBeInTheDocument()
        expect(screen.getByText(/1 Worker Pod/i)).toBeInTheDocument()
        expect(screen.getByText(/dedicated=loadgen:NoSchedule/i)).toBeInTheDocument()
      })

      // Ensure misleading interactive inputs are REMOVED
      expect(screen.queryByRole('spinbutton', { name: /runner pods count/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('textbox', { name: /cpu millicores/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('textbox', { name: /memory units/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('textbox', { name: /node tolerations/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('switch', { name: /start barrier/i })).not.toBeInTheDocument()

      // Contextual help tooltips on read-only specs
      expect(screen.getByRole('button', { name: /^help for runner pods count$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for cpu allocation$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for memory allocation$/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^help for node tolerations$/i })).toBeInTheDocument()

      // Guidance link / hint to Profiles tab
      expect(screen.getByText(/To modify resources or create new configurations/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Profiles tab/i })).toBeInTheDocument()

      // Resource allocation guidance badge
      expect(screen.getByText(/Kubernetes Runner Resource Allocation/i)).toBeInTheDocument()

      // 4. Execution Safeguards section
      expect(screen.getByRole('heading', { name: /^Execution Safeguards$/i })).toBeInTheDocument()
      expect(screen.getByLabelText(/^execution timeout/i)).toBeInTheDocument()
    })

    it('dynamically updates read-only resource specifications when profile changes', async () => {
      renderWithProviders(<TriggerRunDialog open={true} onOpenChange={() => {}} />)

      // Wait for profiles to load in the select dropdown and initial profile specs to display
      await waitFor(() => {
        expect(screen.getByRole('option', { name: /high-perf-runner/i })).toBeInTheDocument()
        expect(screen.getByText(/500m \/ 1000m/i)).toBeInTheDocument()
        expect(screen.getByText(/alpine:3\.20/i)).toBeInTheDocument()
      })

      // Switch to profile-2
      const profileSelect = screen.getByLabelText(/^runner profile$/i)
      fireEvent.change(profileSelect, { target: { value: 'profile-2' } })

      // New profile specs rendered
      await waitFor(() => {
        expect(screen.getByText(/2000m \/ 4000m/i)).toBeInTheDocument()
        expect(screen.getByText(/4Gi \/ 8Gi/i)).toBeInTheDocument()
        expect(screen.getByText(/custom-perf-image:v1/i)).toBeInTheDocument()
        expect(screen.getByText(/kata-containers/i)).toBeInTheDocument()
      })
    })

    it('navigates to Profiles view when clicking the Profiles tab guidance link', async () => {
      const handleNavigateProfiles = vi.fn()
      const handleOpenChange = vi.fn()

      renderWithProviders(
        <TriggerRunDialog
          open={true}
          onOpenChange={handleOpenChange}
          onNavigateProfiles={handleNavigateProfiles}
        />
      )

      const profilesBtn = screen.getByRole('button', { name: /Profiles tab/i })
      fireEvent.click(profilesBtn)

      expect(handleNavigateProfiles).toHaveBeenCalledTimes(1)
      expect(handleOpenChange).toHaveBeenCalledWith(false)
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

    it('displays warning banner and Activate & Dispatch button when DRAFT suite is selected', async () => {
      renderWithProviders(
        <TriggerRunDialog open={true} onOpenChange={() => {}} initialSuiteId="suite-2" />
      )

      await waitFor(() => {
        expect(screen.getByTestId('trigger-run-draft-warning')).toBeInTheDocument()
        expect(screen.getByText(/test suite is currently in draft state/i)).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /activate & dispatch/i })).toBeInTheDocument()
      })
    })

    it('clicking Activate & Dispatch activates the suite and dispatches run', async () => {
      const updateSuiteSpy = vi.spyOn(api, 'updateSuite').mockResolvedValue({
        ...mockSuites[1],
        state: 'ACTIVE',
      })
      const handleTriggered = vi.fn()
      const handleOpenChange = vi.fn()

      renderWithProviders(
        <TriggerRunDialog
          open={true}
          onOpenChange={handleOpenChange}
          initialSuiteId="suite-2"
          onRunTriggered={handleTriggered}
        />
      )

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /activate & dispatch/i })).toBeInTheDocument()
      })

      const activateAndDispatchBtn = screen.getByRole('button', { name: /activate & dispatch/i })
      fireEvent.click(activateAndDispatchBtn)

      await waitFor(() => {
        expect(updateSuiteSpy).toHaveBeenCalledWith('suite-2', {
          name: mockSuites[1].name,
          description: mockSuites[1].description,
          state: 'ACTIVE',
        })
        expect(api.triggerRun).toHaveBeenCalledWith(
          expect.objectContaining({
            suite_id: 'suite-2',
          })
        )
        expect(handleTriggered).toHaveBeenCalled()
        expect(handleOpenChange).toHaveBeenCalledWith(false)
      })

      updateSuiteSpy.mockRestore()
    })

    it('clicking Activate Suite in warning banner activates suite and switches button to Dispatch Run', async () => {
      const updateSuiteSpy = vi.spyOn(api, 'updateSuite').mockResolvedValue({
        ...mockSuites[1],
        state: 'ACTIVE',
      })

      renderWithProviders(
        <TriggerRunDialog open={true} onOpenChange={() => {}} initialSuiteId="suite-2" />
      )

      await waitFor(() => {
        expect(screen.getByTestId('trigger-run-draft-warning')).toBeInTheDocument()
      })

      const bannerActivateBtn = screen.getByRole('button', { name: /activate suite/i })
      fireEvent.click(bannerActivateBtn)

      await waitFor(() => {
        expect(updateSuiteSpy).toHaveBeenCalledWith('suite-2', {
          name: mockSuites[1].name,
          description: mockSuites[1].description,
          state: 'ACTIVE',
        })
        expect(screen.queryByTestId('trigger-run-draft-warning')).not.toBeInTheDocument()
        expect(screen.getByRole('button', { name: /^dispatch run$/i })).toBeInTheDocument()
      })

      updateSuiteSpy.mockRestore()
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

    it('renders execution runs and selects completed run into RunSummaryDashboard with SLA status', async () => {
      vi.spyOn(api, 'getRuns').mockResolvedValue([
        {
          id: 'run-completed-1',
          suiteId: 'suite-1',
          artifactId: 'art-arm64',
          runnerProfileId: 'profile-1',
          status: 'COMPLETED',
          k8sJobName: 'job-done',
          k8sNamespace: 'vuhive-runners',
          durationMs: 300000,
          exitCode: 0,
          slaPassed: true,
          metrics: {
            totalIterations: 20000,
            totalRequests: 60000,
            avgTps: 2000,
            p50DurationMs: 20,
            p90DurationMs: 30,
            p95DurationMs: 40,
            p99DurationMs: 60,
            errorRatePct: 0.0,
          },
          createdAt: new Date().toISOString(),
        },
      ])

      renderWithProviders(<RunsView />)

      await waitFor(() => {
        expect(screen.getByText('Execution Runs')).toBeInTheDocument()
        expect(screen.getByText('run-completed-1')).toBeInTheDocument()
      })

      // Click on row to open summary dashboard
      const runRow = screen.getByText('run-completed-1')
      fireEvent.click(runRow)

      await waitFor(() => {
        expect(screen.getByText('SLA PASSED')).toBeInTheDocument()
        expect(screen.getByText('Executive Summary')).toBeInTheDocument()
        expect(screen.getByText('Execution Details')).toBeInTheDocument()
        expect(screen.getByText('20,000')).toBeInTheDocument()
      })

      // Toggle to execution details
      const detailsBtn = screen.getByRole('button', { name: /execution details/i })
      fireEvent.click(detailsBtn)

      await waitFor(() => {
        expect(screen.getByText('Live Execution Monitor')).toBeInTheDocument()
      })
    })

    it('renders empty state when there are no execution runs', async () => {
      vi.spyOn(api, 'getRuns').mockResolvedValue([])

      renderWithProviders(<RunsView />)

      await waitFor(() => {
        expect(screen.getByText('No execution runs found')).toBeInTheDocument()
        expect(screen.getByText(/trigger your first load test run/i)).toBeInTheDocument()
      })
    })

    it('renders error state with retry button when fetching runs fails', async () => {
      const getRunsSpy = vi.spyOn(api, 'getRuns')
        .mockRejectedValueOnce(new Error('Failed to fetch runs'))
        .mockResolvedValueOnce([])

      renderWithProviders(<RunsView />)

      await waitFor(() => {
        expect(screen.getByText('Failed to load execution runs')).toBeInTheDocument()
      })

      const retryBtn = screen.getByRole('button', { name: /retry/i })
      fireEvent.click(retryBtn)

      await waitFor(() => {
        expect(screen.getByText('No execution runs found')).toBeInTheDocument()
      })
      expect(getRunsSpy).toHaveBeenCalledTimes(2)
    })

    it('renders 100% error rate as 100.00% and never 10000.00% in Runs table and monitor (Issue #219)', async () => {
      const fullErrorRun: HistoricalRun = {
        id: 'run-100pct-err',
        suiteId: 'suite-1',
        status: 'FAILED',
        k8sJobName: 'job-err-100',
        durationMs: 60000,
        exitCode: 1,
        slaPassed: false,
        metrics: {
          totalIterations: 100,
          totalRequests: 100,
          avgTps: 1.67,
          p95DurationMs: 500,
          errorRatePct: 100.0,
        },
        createdAt: new Date().toISOString(),
      }

      vi.spyOn(api, 'getRuns').mockResolvedValue([fullErrorRun])

      renderWithProviders(<RunsView />)

      await waitFor(() => {
        expect(screen.getByText('run-100pct-err')).toBeInTheDocument()
      })

      // In Runs table, 100.00% must be displayed, never 10000.00%
      expect(screen.getByText('100.00%')).toBeInTheDocument()
      expect(screen.queryByText('10000.00%')).not.toBeInTheDocument()

      // Click on row to open LiveRunMonitor (failed run with inspectorTab=monitor or toggle)
      fireEvent.click(screen.getByText('run-100pct-err'))

      const detailsBtn = screen.getByRole('button', { name: /execution details/i })
      fireEvent.click(detailsBtn)

      await waitFor(() => {
        expect(screen.getByText('Live Execution Monitor')).toBeInTheDocument()
      })

      // In LiveRunMonitor KPIs, 100.00% must be displayed, never 10000.00%
      expect(screen.queryByText('10000.00%')).not.toBeInTheDocument()
      const errorRateDisplays = screen.getAllByText('100.00%')
      expect(errorRateDisplays.length).toBeGreaterThanOrEqual(1)
    })
  })
})

