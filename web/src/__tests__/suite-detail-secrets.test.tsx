import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SuiteDetailView } from '@/views/SuiteDetailView'
import { TooltipProvider } from '@/components/ui/tooltip'
import type { TestSuite } from '@/types/suite'

const mockSuite: TestSuite = {
  id: 'suite-secrets-test',
  name: 'Secrets Test Suite',
  description: 'Suite with secrets management',
  state: 'ACTIVE',
  buildStatus: 'READY',
  platforms: ['linux/amd64'],
  createdAt: '2026-03-01T10:00:00Z',
  updatedAt: '2026-03-01T10:00:00Z',
  runCount: 0,
}

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>{children}</TooltipProvider>
      </QueryClientProvider>
    )
  }
}

describe('SuiteDetailView Secrets Tab', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('renders Secrets tab trigger alongside Configurations, Artifacts, and Runs', async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/secrets')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            secrets: [
              {
                id: 'sec-1',
                suite_id: 'suite-secrets-test',
                key: 'API_KEY',
                created_at: '2026-03-01T10:00:00Z',
                updated_at: '2026-03-01T10:00:00Z',
              },
            ],
            count: 1,
          }),
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ configs: [], artifacts: [], runs: [] }),
      })
    })

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <SuiteDetailView suite={mockSuite} onBack={vi.fn()} />
      </Wrapper>
    )

    const secretsTab = await screen.findByRole('tab', { name: /secrets/i })
    expect(secretsTab).toBeInTheDocument()
  })

  it('displays secrets list with key names, masked values, and encryption indicators', async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/secrets')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            secrets: [
              {
                id: 'sec-1',
                suite_id: 'suite-secrets-test',
                key: 'DATABASE_URL',
                created_at: '2026-03-01T10:00:00Z',
                updated_at: '2026-03-01T10:00:00Z',
              },
            ],
            count: 1,
          }),
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ configs: [], artifacts: [], runs: [] }),
      })
    })

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <SuiteDetailView suite={mockSuite} onBack={vi.fn()} />
      </Wrapper>
    )

    const secretsTab = await screen.findByRole('tab', { name: /secrets/i })
    fireEvent.mouseDown(secretsTab, { button: 0 })
    fireEvent.click(secretsTab)

    expect(await screen.findByText('DATABASE_URL')).toBeInTheDocument()
    expect(screen.getByText('••••••••')).toBeInTheDocument()
    expect(screen.getByText(/AES-256-GCM/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /add suite secret/i })).toBeInTheDocument()
  })

  it('displays empty state when no secrets exist for the suite', async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/secrets')) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({ secrets: [], count: 0 }),
        })
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ configs: [], artifacts: [], runs: [] }),
      })
    })

    const Wrapper = createWrapper()
    render(
      <Wrapper>
        <SuiteDetailView suite={mockSuite} onBack={vi.fn()} />
      </Wrapper>
    )

    const secretsTab = await screen.findByRole('tab', { name: /secrets/i })
    fireEvent.mouseDown(secretsTab, { button: 0 })
    fireEvent.click(secretsTab)

    expect(await screen.findByText(/no suite secrets defined/i)).toBeInTheDocument()
    expect(screen.getAllByText(/\$\{secrets\.KEY\}/i).length).toBeGreaterThanOrEqual(1)
  })
})
