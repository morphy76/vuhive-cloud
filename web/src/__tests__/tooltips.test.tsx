import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import * as React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { ProfilesView } from '@/views/ProfilesView'
import { SuitesView } from '@/views/SuitesView'
import { SchedulesView } from '@/views/SchedulesView'
import { SuiteDetailView } from '@/views/SuiteDetailView'
import { LiveRunMonitor } from '@/components/runs/LiveRunMonitor'
import { RunSummaryDashboard } from '@/components/runs/RunSummaryDashboard'
import { SummaryReportInspector } from '@/components/runs/SummaryReportInspector'
import { VirtualizedLogViewer } from '@/components/logs/VirtualizedLogViewer'
import { Sidebar } from '@/components/layout/Sidebar'
import { RecipeProvider } from '@/context/RecipeContext'
import { ThemeProvider } from '@/context/ThemeContext'
import { api, FALLBACK_PROFILES } from '@/lib/api'
import type { TestSuite, SuiteConfiguration, CompiledArtifact, HistoricalRun } from '@/types/suite'
import type { Schedule } from '@/types/schedule'

const mockSuites: TestSuite[] = [
  {
    id: 'suite-smoke-1',
    name: 'smoke-suite',
    description: 'Smoke test suite',
    state: 'ACTIVE',
    buildStatus: 'READY',
    platforms: ['linux/amd64'],
    createdAt: '2026-03-01T10:00:00Z',
    updatedAt: '2026-03-01T10:00:00Z',
  },
]

const mockConfigs: SuiteConfiguration[] = [
  {
    id: 'cfg-smoke-1',
    suiteId: 'suite-smoke-1',
    name: 'Default Config',
    contentYaml: 'scenarios: []',
    isDefault: true,
    createdAt: '2026-03-01T10:00:00Z',
  },
  {
    id: 'cfg-smoke-2',
    suiteId: 'suite-smoke-1',
    name: 'Stress Config',
    contentYaml: 'scenarios: []',
    isDefault: false,
    createdAt: '2026-03-01T10:00:00Z',
  },
]

const mockArtifacts: CompiledArtifact[] = [
  {
    id: 'art-smoke-1',
    suiteId: 'suite-smoke-1',
    platform: 'linux/amd64',
    status: 'READY',
    sha256Checksum: 'abcdef1234567890',
    buildLogsS3Key: 'logs/art-1.log',
    createdAt: '2026-03-01T10:00:00Z',
  },
]

const mockSchedules: Schedule[] = [
  {
    id: 'sched-smoke-1',
    name: 'Nightly Run',
    suiteId: 'suite-smoke-1',
    artifactId: 'art-smoke-1',
    configurationId: 'cfg-smoke-1',
    runnerProfileId: 'standard-single-node',
    cronExpression: '0 2 * * *',
    isActive: true,
    k8sCronJobName: 'vuhive-sched-1',
    createdAt: '2026-03-01T10:00:00Z',
    updatedAt: '2026-03-01T10:00:00Z',
  },
]

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={0}>
        <ThemeProvider>
          <RecipeProvider>{ui}</RecipeProvider>
        </ThemeProvider>
      </TooltipProvider>
    </QueryClientProvider>
  )
}

describe('Action Button Tooltips and Touch Targets', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(api, 'getProfiles').mockResolvedValue([...FALLBACK_PROFILES])
    vi.spyOn(api, 'getSuites').mockResolvedValue([...mockSuites])
    vi.spyOn(api, 'getSuiteConfigs').mockResolvedValue([...mockConfigs])
    vi.spyOn(api, 'getSuiteArtifacts').mockResolvedValue([...mockArtifacts])
    vi.spyOn(api, 'getSuiteRuns').mockResolvedValue([])
    vi.spyOn(api, 'getSchedules').mockResolvedValue([...mockSchedules])
  })

  afterEach(() => {
    cleanup()
  })

  describe('ProfilesView', () => {
    it('wraps Duplicate, Edit, and Delete action buttons with tooltips and uses h-9 w-9 touch target', async () => {
      renderWithProviders(<ProfilesView />)

      await waitFor(() => {
        expect(screen.getByText('standard-single-node')).toBeInTheDocument()
      })

      const duplicateBtn = screen.getByRole('button', { name: /duplicate profile standard-single-node/i })
      const editBtn = screen.getByRole('button', { name: /edit profile standard-single-node/i })
      const deleteBtn = screen.getByRole('button', { name: /delete profile standard-single-node/i })

      expect(duplicateBtn).toBeInTheDocument()
      expect(editBtn).toBeInTheDocument()
      expect(deleteBtn).toBeInTheDocument()

      // WCAG touch target check: should be at least h-9 w-9
      expect(duplicateBtn.className).toContain('h-9')
      expect(duplicateBtn.className).toContain('w-9')
      expect(editBtn.className).toContain('h-9')
      expect(editBtn.className).toContain('w-9')
      expect(deleteBtn.className).toContain('h-9')
      expect(deleteBtn.className).toContain('w-9')

      // Assert Radix Tooltip trigger attachment
      expect(duplicateBtn).toHaveAttribute('data-state')
      expect(editBtn).toHaveAttribute('data-state')
      expect(deleteBtn).toHaveAttribute('data-state')
    })
  })

  describe('SuitesView', () => {
    it('wraps Inspect and Delete suite action buttons with tooltips', async () => {
      renderWithProviders(<SuitesView initialSuites={mockSuites} />)

      const inspectBtn = screen.getByRole('button', { name: /view details for smoke-suite/i })
      const deleteBtns = screen.getAllByRole('button', { name: /delete suite smoke-suite/i })

      expect(inspectBtn).toBeInTheDocument()
      expect(deleteBtns.length).toBeGreaterThan(0)

      expect(inspectBtn).toHaveAttribute('data-state')
      deleteBtns.forEach((btn) => expect(btn).toHaveAttribute('data-state'))
    })
  })

  describe('SchedulesView', () => {
    it('wraps Run Now, Pause/Resume, History, and Delete schedule buttons with tooltips', async () => {
      renderWithProviders(<SchedulesView />)

      await waitFor(() => {
        expect(screen.getByText('Nightly Run')).toBeInTheDocument()
      })

      const runNowBtn = screen.getByRole('button', { name: /run now for schedule nightly run/i })
      const pauseBtn = screen.getByRole('button', { name: /pause schedule nightly run/i })
      const historyBtn = screen.getByRole('button', { name: /view execution history for nightly run/i })
      const deleteBtn = screen.getByRole('button', { name: /delete schedule nightly run/i })

      expect(runNowBtn).toBeInTheDocument()
      expect(pauseBtn).toBeInTheDocument()
      expect(historyBtn).toBeInTheDocument()
      expect(deleteBtn).toBeInTheDocument()

      expect(runNowBtn).toHaveAttribute('data-state')
      expect(pauseBtn).toHaveAttribute('data-state')
      expect(historyBtn).toHaveAttribute('data-state')
      expect(deleteBtn).toHaveAttribute('data-state')
    })
  })

  describe('SuiteDetailView', () => {
    it('wraps Config row actions (Edit, Diff, Delete) with tooltips', async () => {
      renderWithProviders(<SuiteDetailView suite={mockSuites[0]} onBack={vi.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Default Config')).toBeInTheDocument()
      })

      // Configurations tab is active by default
      const editCfgBtn = screen.getByRole('button', { name: /edit and view yaml for default config/i })
      const diffCfgBtn = screen.getByRole('button', { name: /compare configuration default config/i })
      const deleteCfgBtn = screen.getByRole('button', { name: /delete configuration default config/i })

      expect(editCfgBtn).toBeInTheDocument()
      expect(diffCfgBtn).toBeInTheDocument()
      expect(deleteCfgBtn).toBeInTheDocument()

      expect(editCfgBtn).toHaveAttribute('data-state')
      expect(diffCfgBtn).toHaveAttribute('data-state')
      expect(deleteCfgBtn).toHaveAttribute('data-state')
    })
  })

  describe('LiveRunMonitor', () => {
    it('wraps close button with tooltip and uses h-9 w-9 touch target', () => {
      const mockRun: HistoricalRun = {
        id: 'run-live-1',
        suiteId: 'suite-smoke-1',
        status: 'RUNNING',
        createdAt: '2026-03-01T10:00:00Z',
      }
      renderWithProviders(<LiveRunMonitor run={mockRun} onClose={vi.fn()} />)

      const closeBtn = screen.getByRole('button', { name: /close monitor/i })
      expect(closeBtn).toBeInTheDocument()
      expect(closeBtn.className).toContain('h-9')
      expect(closeBtn.className).toContain('w-9')
      expect(closeBtn).toHaveAttribute('data-state')
    })
  })

  describe('RunSummaryDashboard', () => {
    it('wraps close button and copy checksum button with tooltips', () => {
      const mockRun: HistoricalRun = {
        id: 'run-summary-1',
        suiteId: 'suite-smoke-1',
        artifactId: 'art-smoke-1',
        status: 'COMPLETED',
        createdAt: '2026-03-01T10:00:00Z',
      }
      renderWithProviders(<RunSummaryDashboard run={mockRun} onClose={vi.fn()} />)

      const closeBtn = screen.getByRole('button', { name: /close summary dashboard/i })
      expect(closeBtn).toBeInTheDocument()
      expect(closeBtn.className).toContain('h-9')
      expect(closeBtn.className).toContain('w-9')
      expect(closeBtn).toHaveAttribute('data-state')

      const copyChecksumBtn = screen.getByRole('button', { name: /copy checksum/i })
      expect(copyChecksumBtn).toBeInTheDocument()
      expect(copyChecksumBtn).toHaveAttribute('data-state')
    })
  })

  describe('SummaryReportInspector', () => {
    it('wraps close button with tooltip and uses h-9 w-9 touch target', () => {
      renderWithProviders(
        <SummaryReportInspector
          runId="run-summary-1"
          report={{ status: 'PASS', passed: true, suite_name: 'Smoke', scenario: 'smoke' }}
          onClose={vi.fn()}
        />
      )

      const closeBtn = screen.getByRole('button', { name: /close inspector/i })
      expect(closeBtn).toBeInTheDocument()
      expect(closeBtn.className).toContain('h-9')
      expect(closeBtn.className).toContain('w-9')
      expect(closeBtn).toHaveAttribute('data-state')
    })
  })

  describe('VirtualizedLogViewer', () => {
    it('wraps high-contrast toggle and close button with tooltips', () => {
      renderWithProviders(
        <VirtualizedLogViewer logs="Sample log text" runId="run-summary-1" onClose={vi.fn()} />
      )

      const highContrastBtn = screen.getByRole('button', { name: /toggle high-contrast mode/i })
      const closeBtn = screen.getByRole('button', { name: /close log viewer/i })

      expect(highContrastBtn).toBeInTheDocument()
      expect(highContrastBtn).toHaveAttribute('data-state')

      expect(closeBtn).toBeInTheDocument()
      expect(closeBtn.className).toContain('h-9')
      expect(closeBtn.className).toContain('w-9')
      expect(closeBtn).toHaveAttribute('data-state')
    })
  })

  describe('Sidebar', () => {
    it('wraps sidebar collapse button with tooltip and h-9 w-9 target', () => {
      renderWithProviders(
        <Sidebar
          currentRoute="suites"
          onSelectRoute={vi.fn()}
          isCollapsed={false}
          onToggleCollapse={vi.fn()}
        />
      )

      const toggleBtn = screen.getByRole('button', { name: /collapse sidebar/i })
      expect(toggleBtn).toBeInTheDocument()
      expect(toggleBtn.className).toContain('h-9')
      expect(toggleBtn.className).toContain('w-9')
      expect(toggleBtn).toHaveAttribute('data-state')
    })
  })
})
