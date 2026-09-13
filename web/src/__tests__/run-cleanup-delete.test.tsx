import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import * as React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CleanRuntimeDialog } from '@/components/dialogs/CleanRuntimeDialog'
import { DeleteRunDialog } from '@/components/dialogs/DeleteRunDialog'
import { LiveRunMonitor } from '@/components/runs/LiveRunMonitor'
import { RunsView } from '@/views/RunsView'
import { RecipeProvider } from '@/context/RecipeContext'
import { TooltipProvider } from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import type { HistoricalRun } from '@/types/suite'
import type { RunnerProfile } from '@/types/profile'

const mockProfile: RunnerProfile = {
  id: 'prof-smoke',
  name: 'smoke-profile',
  description: 'Smoke test runner profile',
  runner_image: 'alpine:3.20',
  cpu_request: '250m',
  cpu_limit: '500m',
  memory_request: '256Mi',
  memory_limit: '512Mi',
  node_selector: {},
  affinity: { node_selector_terms: [] },
  tolerations: [],
  active_deadline_seconds: 3600,
  runtime_class_name: null,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
}

const mockCompletedRun: HistoricalRun = {
  id: 'run-term-1',
  suiteId: 'suite-1',
  artifactId: 'art-1',
  runnerProfileId: 'prof-smoke',
  status: 'COMPLETED',
  k8sJobName: 'vuhive-job-term',
  k8sNamespace: 'vuhive-runners',
  durationMs: 12000,
  exitCode: 0,
  slaPassed: true,
  createdAt: new Date().toISOString(),
}

const mockFailedInitRun: HistoricalRun = {
  id: 'run-init-err',
  suiteId: 'suite-1',
  artifactId: 'art-1',
  runnerProfileId: 'prof-smoke',
  status: 'FAILED',
  k8sJobName: 'vuhive-job-init-err',
  k8sNamespace: 'vuhive-runners',
  abortReason: 'pod initContainer fetch-artifacts failed: Init:Error',
  exitCode: 1,
  durationMs: 3000,
  createdAt: new Date().toISOString(),
}

const mockActiveRun: HistoricalRun = {
  id: 'run-active-1',
  suiteId: 'suite-1',
  artifactId: 'art-1',
  runnerProfileId: 'prof-smoke',
  status: 'RUNNING',
  k8sJobName: 'vuhive-job-active',
  k8sNamespace: 'vuhive-runners',
  startedAt: new Date().toISOString(),
  createdAt: new Date().toISOString(),
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <RecipeProvider>{ui}</RecipeProvider>
      </TooltipProvider>
    </QueryClientProvider>
  )
}

describe('Run Runtime Cleanup and Deletion UI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(api, 'getProfile').mockResolvedValue(mockProfile)
    vi.spyOn(api, 'getProfiles').mockResolvedValue([mockProfile])
  })

  describe('CleanRuntimeDialog', () => {
    it('renders dialog details and triggers confirm', async () => {
      const onConfirm = vi.fn()
      const onOpenChange = vi.fn()

      renderWithProviders(
        <CleanRuntimeDialog
          open={true}
          onOpenChange={onOpenChange}
          run={mockFailedInitRun}
          onConfirm={onConfirm}
        />
      )

      expect(screen.getByText('Clean Runtime Resources')).toBeInTheDocument()
      expect(screen.getByText('run-init-err')).toBeInTheDocument()
      expect(screen.getByText(/vuhive-job-init-err/)).toBeInTheDocument()

      const cleanBtn = screen.getByRole('button', { name: /clean runtime/i })
      fireEvent.click(cleanBtn)

      expect(onConfirm).toHaveBeenCalledTimes(1)
    })
  })

  describe('DeleteRunDialog', () => {
    it('renders delete confirmation and allows deleting completed run', async () => {
      const onConfirm = vi.fn()
      const onOpenChange = vi.fn()

      renderWithProviders(
        <DeleteRunDialog
          open={true}
          onOpenChange={onOpenChange}
          run={mockCompletedRun}
          onConfirm={onConfirm}
        />
      )

      expect(screen.getByText('Delete Test Run')).toBeInTheDocument()
      expect(screen.getByText('run-term-1')).toBeInTheDocument()

      const deleteBtn = screen.getByRole('button', { name: /delete run/i })
      expect(deleteBtn).not.toBeDisabled()
      fireEvent.click(deleteBtn)

      expect(onConfirm).toHaveBeenCalledTimes(1)
    })

    it('blocks deletion when run is currently active (RUNNING)', async () => {
      const onConfirm = vi.fn()
      const onOpenChange = vi.fn()

      renderWithProviders(
        <DeleteRunDialog
          open={true}
          onOpenChange={onOpenChange}
          run={mockActiveRun}
          onConfirm={onConfirm}
        />
      )

      expect(screen.getByText(/active run cannot be deleted/i)).toBeInTheDocument()
      const deleteBtn = screen.getByRole('button', { name: /delete run/i })
      expect(deleteBtn).toBeDisabled()

      fireEvent.click(deleteBtn)
      expect(onConfirm).not.toHaveBeenCalled()
    })
  })

  describe('LiveRunMonitor Profile and Init:Error Handling', () => {
    it('clarifies runner profile allocation vs init container resources', async () => {
      renderWithProviders(<LiveRunMonitor run={mockCompletedRun} />)

      await waitFor(() => {
        expect(screen.getByText('Runner Profile')).toBeInTheDocument()
        expect(screen.getByText(/smoke-profile \(250m\/256Mi\)/)).toBeInTheDocument()
        expect(screen.getByText('Init: 50m / 64Mi')).toBeInTheDocument()
      })
    })

    it('prominently surfaces Init:Error banner and Clean Runtime button on failure', async () => {
      const cleanupSpy = vi.spyOn(api, 'cleanupRun').mockResolvedValue({
        ...mockFailedInitRun,
        status: 'ABORTED',
      })

      renderWithProviders(<LiveRunMonitor run={mockFailedInitRun} />)

      await waitFor(() => {
        expect(screen.getByText(/Runner Initialization Failure Detected/i)).toBeInTheDocument()
        expect(screen.getByText(/lightweight bootstrap init resources/i)).toBeInTheDocument()
      })

      // Clean Runtime button should be available in both banner and header
      const cleanBtns = screen.getAllByRole('button', { name: /clean runtime/i })
      expect(cleanBtns.length).toBeGreaterThan(0)

      // Open dialog and confirm
      fireEvent.click(cleanBtns[0])

      await waitFor(() => {
        expect(screen.getByText('Clean Runtime Resources')).toBeInTheDocument()
      })

      const dialogCleanBtn = screen.getAllByRole('button', { name: /clean runtime/i }).find(
        (b) => b.closest('[role="dialog"]') !== null
      )
      expect(dialogCleanBtn).toBeDefined()
      fireEvent.click(dialogCleanBtn!)

      await waitFor(() => {
        expect(cleanupSpy).toHaveBeenCalledWith('run-init-err')
      })
    })
  })

  describe('RunsView Actions Column and Integration', () => {
    it('renders Actions column and triggers cleanup and delete workflows', async () => {
      vi.spyOn(api, 'getRuns').mockResolvedValue([mockFailedInitRun, mockCompletedRun])
      vi.spyOn(api, 'getSuites').mockResolvedValue([
        {
          id: 'suite-1',
          name: 'Smoke Test Suite',
          description: 'Smoke test suite description',
          state: 'ACTIVE',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ])

      const cleanupSpy = vi.spyOn(api, 'cleanupRun').mockResolvedValue({
        ...mockFailedInitRun,
        status: 'ABORTED',
      })
      const deleteSpy = vi.spyOn(api, 'deleteRun').mockResolvedValue()

      renderWithProviders(<RunsView />)

      await waitFor(() => {
        expect(screen.getByText('Actions')).toBeInTheDocument()
        expect(screen.getByText('run-init-err')).toBeInTheDocument()
        expect(screen.getByText('run-term-1')).toBeInTheDocument()
      })

      // Click clean runtime on first row
      const cleanButtons = screen.getAllByRole('button', { name: /clean runtime/i })
      expect(cleanButtons.length).toBeGreaterThanOrEqual(2)
      fireEvent.click(cleanButtons[0])

      await waitFor(() => {
        expect(screen.getByText('Clean Runtime Resources')).toBeInTheDocument()
      })

      // Confirm clean
      const confirmCleanBtn = screen.getAllByRole('button', { name: /clean runtime/i }).find(
        (b) => b.closest('[role="dialog"]') !== null
      )
      fireEvent.click(confirmCleanBtn!)

      await waitFor(() => {
        expect(cleanupSpy).toHaveBeenCalledWith('run-init-err')
      })

      // Click delete run on second row
      const deleteButtons = screen.getAllByRole('button', { name: /delete run/i })
      expect(deleteButtons.length).toBeGreaterThanOrEqual(2)
      fireEvent.click(deleteButtons[1])

      await waitFor(() => {
        expect(screen.getByText('Delete Test Run')).toBeInTheDocument()
      })

      // Confirm delete
      const confirmDeleteBtn = screen.getAllByRole('button', { name: /delete run/i }).find(
        (b) => b.closest('[role="dialog"]') !== null
      )
      fireEvent.click(confirmDeleteBtn!)

      await waitFor(() => {
        expect(deleteSpy).toHaveBeenCalledWith('run-term-1')
      })
    })
  })
})
