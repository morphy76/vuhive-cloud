import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { axe } from 'vitest-axe'
import { DeleteSuiteDialog } from '@/components/dialogs/DeleteSuiteDialog'
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
