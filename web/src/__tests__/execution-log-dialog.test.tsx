import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ExecutionLogDialog } from '../components/dialogs/ExecutionLogDialog'
import { api } from '../lib/api'
import type { HistoricalRun } from '../types/suite'

const mockRun: HistoricalRun = {
  id: 'run-999',
  suiteId: 'suite-1',
  status: 'COMPLETED',
  createdAt: '2026-09-11T20:00:00Z',
  k8sJobName: 'vuhive-run-999-job',
}

describe('ExecutionLogDialog', () => {
  let queryClient: QueryClient

  beforeEach(() => {
    vi.clearAllMocks()
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })
    Object.defineProperty(HTMLElement.prototype, 'offsetHeight', { configurable: true, value: 600 })
    Object.defineProperty(HTMLElement.prototype, 'offsetWidth', { configurable: true, value: 800 })
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', { configurable: true, value: 600 })
    Object.defineProperty(HTMLElement.prototype, 'clientWidth', { configurable: true, value: 800 })
  })

  it('renders nothing when run is null', () => {
    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <ExecutionLogDialog open={true} onOpenChange={() => {}} run={null} />
      </QueryClientProvider>
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders dialog content with virtualized viewer when open', async () => {
    vi.spyOn(api, 'getRunLogs').mockResolvedValue(
      '\u001b[32m[INFO]\u001b[0m Container startup initialized\n\u001b[32m[INFO]\u001b[0m Execution done'
    )

    const onOpenChange = vi.fn()
    render(
      <QueryClientProvider client={queryClient}>
        <ExecutionLogDialog open={true} onOpenChange={onOpenChange} run={mockRun} />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByText('run-999')).toBeInTheDocument()
      expect(screen.getByText(/Container Execution Logs/i)).toBeInTheDocument()
      expect(screen.getByText(/Container startup initialized/i)).toBeInTheDocument()
    })

    const closeBtn = screen.getByLabelText(/close log viewer/i)
    fireEvent.click(closeBtn)
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
