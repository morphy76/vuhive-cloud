import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { axe } from 'vitest-axe'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { DeleteSuiteDialog } from '@/components/dialogs/DeleteSuiteDialog'
import { useDeleteSuite, useCreateSuite } from '@/hooks/use-suites'
import { api } from '@/lib/api'
import type { TestSuite } from '@/types/suite'

const mockDraftSuite: TestSuite = {
  id: 'suite-draft-1',
  name: 'Authentication Load Test',
  description: 'OAuth2 token refresh load simulation',
  state: 'DRAFT',
  buildStatus: 'READY',
  platforms: ['linux/amd64'],
  createdAt: '2026-03-01T10:00:00Z',
  updatedAt: '2026-03-10T12:00:00Z',
  runCount: 0,
}

const mockActiveSuite: TestSuite = {
  id: 'suite-active-1',
  name: 'Payment Gateway Stress Test',
  description: 'Simulated high concurrency card checkout',
  state: 'ACTIVE',
  buildStatus: 'READY',
  platforms: ['linux/amd64', 'linux/arm64'],
  createdAt: '2026-03-01T10:00:00Z',
  updatedAt: '2026-03-10T12:00:00Z',
  runCount: 14,
}

describe('DeleteSuiteDialog', () => {
  it('renders destructive warning, cascading deletion details, and S3 retention notice', () => {
    render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockDraftSuite}
        onConfirm={vi.fn()}
      />
    )

    expect(screen.getByRole('heading', { name: /delete test suite/i })).toBeInTheDocument()
    expect(screen.getAllByText(/Authentication Load Test/i).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/suite-draft-1/i)).toBeInTheDocument()

    // Irreversible warning
    expect(screen.getByText(/cannot be undone/i)).toBeInTheDocument()

    // Cascading deletion warning
    expect(screen.getByText(/configurations, artifacts, schedules, and historical runs/i)).toBeInTheDocument()

    // S3 object retention notice
    expect(screen.getByText(/s3\/minio/i)).toBeInTheDocument()
  })

  it('allows immediate deletion for DRAFT suites without requiring name confirmation', () => {
    const onConfirm = vi.fn()
    render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockDraftSuite}
        onConfirm={onConfirm}
      />
    )

    const deleteBtn = screen.getByRole('button', { name: /delete test suite/i })
    expect(deleteBtn).toBeEnabled()

    fireEvent.click(deleteBtn)
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('requires typing exact suite name to confirm deletion for ACTIVE suites', () => {
    const onConfirm = vi.fn()
    render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockActiveSuite}
        onConfirm={onConfirm}
      />
    )

    const deleteBtn = screen.getByRole('button', { name: /delete test suite/i })
    expect(deleteBtn).toBeDisabled()

    const input = screen.getByPlaceholderText(mockActiveSuite.name)
    expect(input).toBeInTheDocument()

    // Wrong name
    fireEvent.change(input, { target: { value: 'Wrong Name' } })
    expect(deleteBtn).toBeDisabled()

    // Correct name
    fireEvent.change(input, { target: { value: mockActiveSuite.name } })
    expect(deleteBtn).toBeEnabled()

    fireEvent.click(deleteBtn)
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('blocks deletion when hasActiveBuilds is true and displays informative warning', () => {
    render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockDraftSuite}
        onConfirm={vi.fn()}
        hasActiveBuilds={true}
      />
    )

    expect(screen.getByText(/cannot delete suite while builds are in progress/i)).toBeInTheDocument()
    const deleteBtn = screen.getByRole('button', { name: /delete test suite/i })
    expect(deleteBtn).toBeDisabled()
  })

  it('blocks deletion when hasActiveRuns is true and displays informative warning', () => {
    render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockDraftSuite}
        onConfirm={vi.fn()}
        hasActiveRuns={true}
      />
    )

    expect(screen.getByText(/cannot delete suite while test runs are actively executing/i)).toBeInTheDocument()
    const deleteBtn = screen.getByRole('button', { name: /delete test suite/i })
    expect(deleteBtn).toBeDisabled()
  })

  it('displays loading spinner and disables buttons during deletion', () => {
    render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockDraftSuite}
        onConfirm={vi.fn()}
        isDeleting={true}
      />
    )

    const deleteBtn = screen.getByRole('button', { name: /deleting\.\.\./i })
    expect(deleteBtn).toBeDisabled()
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled()
  })

  it('has no accessibility violations', async () => {
    const { container } = render(
      <DeleteSuiteDialog
        open={true}
        onOpenChange={vi.fn()}
        suite={mockActiveSuite}
        onConfirm={vi.fn()}
      />
    )

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})

describe('useDeleteSuite & useCreateSuite cache invalidation', () => {
  it('invalidates both suites and dashboard queries upon successful suite deletion', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const deleteSpy = vi.spyOn(api, 'deleteSuite').mockResolvedValue(undefined)

    function TestComponent() {
      const deleteMutation = useDeleteSuite()
      return (
        <button onClick={() => deleteMutation.mutate('suite-to-delete')}>
          Confirm Delete
        </button>
      )
    }

    render(
      <QueryClientProvider client={queryClient}>
        <TestComponent />
      </QueryClientProvider>
    )

    fireEvent.click(screen.getByRole('button', { name: /confirm delete/i }))

    await waitFor(() => {
      expect(deleteSpy).toHaveBeenCalledWith('suite-to-delete')
    })

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['suites'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['dashboard'] })

    deleteSpy.mockRestore()
  })

  it('invalidates dashboard queries upon successful suite creation', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const createSpy = vi.spyOn(api, 'createSuite').mockResolvedValue(mockDraftSuite)

    function TestComponent() {
      const createMutation = useCreateSuite()
      return (
        <button onClick={() => createMutation.mutate({ name: 'New Test Suite' })}>
          Create Suite
        </button>
      )
    }

    render(
      <QueryClientProvider client={queryClient}>
        <TestComponent />
      </QueryClientProvider>
    )

    fireEvent.click(screen.getByRole('button', { name: /create suite/i }))

    await waitFor(() => {
      expect(createSpy).toHaveBeenCalledWith({ name: 'New Test Suite' })
    })

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['suites'] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['dashboard'] })

    createSpy.mockRestore()
  })
})
