import { render, screen, fireEvent, within, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { axe } from 'vitest-axe'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SuitesView } from '@/views/SuitesView'
import { SuiteDetailView } from '@/views/SuiteDetailView'
import { CreateSuiteDialog } from '@/components/dialogs/CreateSuiteDialog'
import { RecipeProvider } from '@/context/RecipeContext'
import { TooltipProvider } from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import type { TestSuite } from '@/types/suite'

const createTestWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <RecipeProvider>
          <TooltipProvider>{children}</TooltipProvider>
        </RecipeProvider>
      </QueryClientProvider>
    )
  }
}

const mockSuites: TestSuite[] = [
  {
    id: 'suite-checkout',
    name: 'Checkout & Payment Stress Test',
    description: 'E2E checkout funnel under heavy synthetic concurrency',
    state: 'ACTIVE',
    buildStatus: 'READY',
    platforms: ['linux/amd64', 'linux/arm64'],
    createdAt: '2026-03-01T10:00:00Z',
    updatedAt: '2026-03-10T12:00:00Z',
    runCount: 12,
  },
  {
    id: 'suite-catalog',
    name: 'Product Catalog High Throughput',
    description: 'Read-heavy product search and filtering benchmark',
    state: 'ACTIVE',
    buildStatus: 'BUILDING',
    platforms: ['linux/arm64'],
    createdAt: '2026-03-05T09:00:00Z',
    updatedAt: '2026-03-11T08:00:00Z',
    runCount: 5,
  },
  {
    id: 'suite-auth',
    name: 'OAuth2 Token Grant Barrier Test',
    description: 'Distributed synchronization barrier during token refresh bursts',
    state: 'DRAFT',
    buildStatus: 'READY',
    platforms: ['linux/amd64'],
    createdAt: '2026-02-15T14:30:00Z',
    updatedAt: '2026-02-20T16:45:00Z',
    runCount: 0,
  },
]

describe('SuitesView Catalog', () => {
  it('renders test suite catalog heading and create button', () => {
    const Wrapper = createTestWrapper()
    render(<SuitesView initialSuites={mockSuites} />, { wrapper: Wrapper })

    expect(screen.getByRole('heading', { level: 1, name: /test suites/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /new suite/i })).toBeInTheDocument()
    expect(screen.getByRole('searchbox', { name: /search test suites/i })).toBeInTheDocument()
  })

  it('renders table on desktop and card list for mobile devices', () => {
    const Wrapper = createTestWrapper()
    render(<SuitesView initialSuites={mockSuites} />, { wrapper: Wrapper })

    // Desktop table exists
    const table = screen.getByRole('table')
    expect(table).toBeInTheDocument()
    expect(within(table).getByText('Checkout & Payment Stress Test')).toBeInTheDocument()

    // Mobile card list items exist
    const mobileCards = screen.getAllByTestId('suite-mobile-card')
    expect(mobileCards.length).toBe(mockSuites.length)
  })

  it('filters suites in real-time by search query matching name or description', () => {
    const Wrapper = createTestWrapper()
    render(<SuitesView initialSuites={mockSuites} />, { wrapper: Wrapper })

    const searchInput = screen.getByRole('searchbox', { name: /search test suites/i })
    fireEvent.change(searchInput, { target: { value: 'Product Catalog' } })

    // Matches in both desktop table and mobile card
    expect(screen.getAllByText('Product Catalog High Throughput').length).toBeGreaterThanOrEqual(1)
    expect(screen.queryByText('Checkout & Payment Stress Test')).not.toBeInTheDocument()

    // Filter by description
    fireEvent.change(searchInput, { target: { value: 'synthetic concurrency' } })
    expect(screen.getAllByText('Checkout & Payment Stress Test').length).toBeGreaterThanOrEqual(1)
    expect(screen.queryByText('Product Catalog High Throughput')).not.toBeInTheDocument()
  })

  it('sorts test suites by name and creation date', () => {
    const Wrapper = createTestWrapper()
    render(<SuitesView initialSuites={mockSuites} />, { wrapper: Wrapper })

    const sortSelect = screen.getByLabelText(/sort test suites/i)
    fireEvent.change(sortSelect, { target: { value: 'name-asc' } })

    const cards = screen.getAllByTestId('suite-mobile-card')
    expect(within(cards[0]).getByText('Checkout & Payment Stress Test')).toBeInTheDocument()
    expect(within(cards[1]).getByText('OAuth2 Token Grant Barrier Test')).toBeInTheDocument()
    expect(within(cards[2]).getByText('Product Catalog High Throughput')).toBeInTheDocument()
  })

  it('displays empty state when no suites match the search filter', () => {
    const Wrapper = createTestWrapper()
    render(<SuitesView initialSuites={mockSuites} />, { wrapper: Wrapper })

    const searchInput = screen.getByRole('searchbox', { name: /search test suites/i })
    fireEvent.change(searchInput, { target: { value: 'nonexistent-suite-query-xyz' } })

    expect(screen.getByText(/no test suites found/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Clear Search' })).toBeInTheDocument()
  })
})

describe('CreateSuiteDialog Name Uniqueness Validation', () => {
  it('validates name uniqueness in real-time and disables submission on duplicate', async () => {
    const Wrapper = createTestWrapper()
    const handleOpenChange = vi.fn()

    render(
      <CreateSuiteDialog
        open={true}
        onOpenChange={handleOpenChange}
        existingSuites={mockSuites}
      />,
      { wrapper: Wrapper }
    )

    const nameInput = screen.getByRole('textbox', { name: 'Suite Name' })
    const submitBtn = screen.getByRole('button', { name: /create suite/i })

    // Initially empty
    expect(nameInput).toHaveValue('')

    // Type duplicate name (case-insensitive)
    fireEvent.change(nameInput, { target: { value: 'checkout & payment stress test' } })
    expect(await screen.findByText(/a suite with this name already exists/i)).toBeInTheDocument()
    expect(nameInput).toHaveAttribute('aria-invalid', 'true')
    expect(submitBtn).toBeDisabled()

    // Change to unique name
    fireEvent.change(nameInput, { target: { value: 'New Unique Scenario Suite' } })
    expect(screen.queryByText(/a suite with this name already exists/i)).not.toBeInTheDocument()
    expect(nameInput).toHaveAttribute('aria-invalid', 'false')
    expect(submitBtn).toBeEnabled()
  })
})

describe('SuiteDetailView & Tab Navigation', () => {
  it('renders suite header metadata, action buttons, and sub-resource tabs', async () => {
    const Wrapper = createTestWrapper()
    const onBack = vi.fn()

    render(
      <SuiteDetailView
        suite={mockSuites[0]}
        onBack={onBack}
      />,
      { wrapper: Wrapper }
    )

    // Header metadata
    expect(screen.getByRole('heading', { level: 1, name: mockSuites[0].name })).toBeInTheDocument()
    expect(screen.getByText(mockSuites[0].description)).toBeInTheDocument()
    expect(screen.getByText(mockSuites[0].id)).toBeInTheDocument()

    // Quick action buttons
    expect(screen.getByRole('button', { name: /trigger run/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /upload build/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /attach config/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /delete suite/i })).toBeInTheDocument()

    // Back button
    const backBtn = screen.getByRole('button', { name: /back to suites/i })
    fireEvent.click(backBtn)
    expect(onBack).toHaveBeenCalledTimes(1)

    // Radix Tabs navigation
    const configTab = screen.getByRole('tab', { name: /configurations/i })
    const artifactsTab = screen.getByRole('tab', { name: /artifacts/i })
    const runsTab = screen.getByRole('tab', { name: /runs/i })

    expect(configTab).toBeInTheDocument()
    expect(artifactsTab).toBeInTheDocument()
    expect(runsTab).toBeInTheDocument()

    // Default tab is Configurations
    expect(configTab).toHaveAttribute('data-state', 'active')
  })

  it('opens delete confirmation dialog when Delete Suite is clicked in detail view', async () => {
    const Wrapper = createTestWrapper()
    render(
      <SuiteDetailView
        suite={mockSuites[0]}
        onBack={() => {}}
      />,
      { wrapper: Wrapper }
    )

    const deleteBtn = screen.getByRole('button', { name: /delete suite/i })
    fireEvent.click(deleteBtn)

    expect(await screen.findByRole('heading', { name: /delete test suite/i })).toBeInTheDocument()
  })

  it('has no accessibility violations in SuiteDetailView', async () => {
    const Wrapper = createTestWrapper()
    const { container } = render(
      <SuiteDetailView
        suite={mockSuites[0]}
        onBack={() => {}}
      />,
      { wrapper: Wrapper }
    )

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})

describe('SuitesView Delete Action', () => {
  it('opens delete confirmation dialog when clicking row delete action', async () => {
    const Wrapper = createTestWrapper()
    render(<SuitesView initialSuites={mockSuites} />, { wrapper: Wrapper })

    const deleteButtons = screen.getAllByRole('button', { name: /delete suite/i })
    expect(deleteButtons.length).toBeGreaterThanOrEqual(1)

    fireEvent.click(deleteButtons[0])
    expect(await screen.findByRole('heading', { name: /delete test suite/i })).toBeInTheDocument()
  })
})

describe('SuiteDetailView Artifact Actions (Cancel, Retry, Delete)', () => {
  it('allows cancelling building artifact, retrying failed artifact, and deleting artifact', async () => {
    const mockArtifacts = [
      {
        id: 'art-building-1',
        suiteId: mockSuites[0].id,
        platform: 'linux/amd64',
        status: 'BUILDING' as const,
        createdAt: '2026-03-10T10:00:00Z',
      },
      {
        id: 'art-failed-2',
        suiteId: mockSuites[0].id,
        platform: 'linux/arm64',
        status: 'FAILED' as const,
        errorMessage: 'exit code 1',
        createdAt: '2026-03-10T11:00:00Z',
      },
      {
        id: 'art-cancelled-3',
        suiteId: mockSuites[0].id,
        platform: 'linux/amd64',
        status: 'CANCELLED' as const,
        errorMessage: 'Cancelled by user',
        createdAt: '2026-03-10T12:00:00Z',
      },
    ]

    const getArtifactsSpy = vi.spyOn(api, 'getSuiteArtifacts').mockResolvedValue(mockArtifacts)
    const cancelSpy = vi.spyOn(api, 'cancelSuiteBuild').mockResolvedValue({
      id: 'art-building-1',
      suiteId: mockSuites[0].id,
      platform: 'linux/amd64',
      status: 'CANCELLED',
      createdAt: '2026-03-10T10:00:00Z',
    })
    const retrySpy = vi.spyOn(api, 'retrySuiteBuild').mockResolvedValue({
      id: 'art-failed-2',
      suiteId: mockSuites[0].id,
      platform: 'linux/arm64',
      status: 'BUILDING',
      createdAt: '2026-03-10T11:00:00Z',
    })
    const deleteSpy = vi.spyOn(api, 'deleteSuiteArtifact').mockResolvedValue(undefined)

    const Wrapper = createTestWrapper()
    render(<SuiteDetailView suite={mockSuites[0]} onBack={() => {}} />, { wrapper: Wrapper })

    // Switch to Artifacts tab
    const artifactsTab = screen.getByRole('tab', { name: /artifacts/i })
    fireEvent.mouseDown(artifactsTab, { button: 0 })
    fireEvent.click(artifactsTab)

    // Verify artifact rows rendered
    expect(await screen.findByText('art-building-1')).toBeInTheDocument()
    expect(screen.getByText('art-failed-2')).toBeInTheDocument()
    expect(screen.getByText('art-cancelled-3')).toBeInTheDocument()

    // Test Cancel on building artifact
    const cancelBtn = screen.getByRole('button', { name: /cancel build for art-building-1/i })
    fireEvent.click(cancelBtn)
    await waitFor(() => {
      expect(cancelSpy).toHaveBeenCalledWith(mockSuites[0].id, 'art-building-1', expect.any(String))
    })

    // Test Retry on failed artifact
    const retryBtn = screen.getByRole('button', { name: /retry build for art-failed-2/i })
    fireEvent.click(retryBtn)
    await waitFor(() => {
      expect(retrySpy).toHaveBeenCalledWith(mockSuites[0].id, 'art-failed-2')
    })

    // Test Delete on cancelled artifact opens DeleteArtifactDialog
    const deleteBtn = screen.getByRole('button', { name: /delete artifact art-cancelled-3/i })
    fireEvent.click(deleteBtn)

    expect(await screen.findByRole('heading', { name: /delete artifact/i })).toBeInTheDocument()
    const confirmDeleteBtn = screen.getByRole('button', { name: 'Delete Artifact' })
    fireEvent.click(confirmDeleteBtn)
    await waitFor(() => {
      expect(deleteSpy).toHaveBeenCalledWith(mockSuites[0].id, 'art-cancelled-3')
    })

    getArtifactsSpy.mockRestore()
    cancelSpy.mockRestore()
    retrySpy.mockRestore()
    deleteSpy.mockRestore()
  })
})
