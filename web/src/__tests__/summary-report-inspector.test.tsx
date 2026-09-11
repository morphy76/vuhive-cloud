import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SummaryReportInspector } from '@/components/runs/SummaryReportInspector'
import * as exportUtils from '@/lib/export-utils'
import { api } from '@/lib/api'

vi.mock('@/lib/api', () => ({
  api: {
    getRunReport: vi.fn(),
  },
}))

describe('SummaryReportInspector', () => {
  let queryClient: QueryClient
  let downloadJsonSpy: any
  let exportCsvSpy: any

  const mockReport = {
    suite_name: 'Checkout Suite',
    scenario: 'e2e_checkout',
    version: '1.0.0',
    passed: true,
    sla_passed: true,
    total_iterations: 500,
    total_requests: 2500,
    avg_tps: 50.0,
    p50_duration_ms: 20.0,
    p95_duration_ms: 45.0,
    p99_duration_ms: 80.0,
    error_rate_pct: 0.1,
    steps: [
      {
        name: 'GET /cart',
        requests: 1000,
        tps: 20.0,
        p50_ms: 15.0,
        p90_ms: 25.0,
        p95_ms: 35.0,
        p99_ms: 50.0,
        failed_requests: 0,
        error_rate_pct: 0.0,
        status: 'PASS',
      },
      {
        name: 'POST /checkout',
        requests: 1500,
        tps: 30.0,
        p50_ms: 25.0,
        p90_ms: 40.0,
        p95_ms: 55.0,
        p99_ms: 90.0,
        failed_requests: 3,
        error_rate_pct: 0.2,
        status: 'PASS',
      },
    ],
    metrics: [
      { name: 'vuhive.http.reqs', type: 'counter', count: 2500 },
      { name: 'vuhive.http.req_duration', type: 'duration', mean: 22.5, p95: 45.0 },
    ],
    thresholds: [
      { metric: 'vuhive.http.req_duration', stat: 'p95', operator: '<=', target: '50ms', actual: '45ms', passed: true },
    ],
  }

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })
    vi.clearAllMocks()

    downloadJsonSpy = vi.spyOn(exportUtils, 'downloadJsonFile').mockImplementation(() => {})
    exportCsvSpy = vi.spyOn(exportUtils, 'exportToCsvFile').mockImplementation(() => {})

    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  function renderInspector(runId = 'run-123') {
    return render(
      <QueryClientProvider client={queryClient}>
        <SummaryReportInspector runId={runId} />
      </QueryClientProvider>
    )
  }

  it('renders loading state while fetching report', () => {
    vi.mocked(api.getRunReport).mockReturnValue(new Promise(() => {}))
    renderInspector()

    expect(screen.getByText(/loading telemetry report/i)).toBeInTheDocument()
  })

  it('renders tabular overview with scenario steps by default', async () => {
    vi.mocked(api.getRunReport).mockResolvedValue(mockReport)
    renderInspector()

    await waitFor(() => {
      expect(screen.getByText('GET /cart')).toBeInTheDocument()
      expect(screen.getByText('POST /checkout')).toBeInTheDocument()
    })

    // Check table headers
    expect(screen.getByText(/step \/ endpoint name/i)).toBeInTheDocument()
    expect(screen.getByText(/error rate/i)).toBeInTheDocument()
  })

  it('switches between tabular tabs (Steps, Metrics, SLA Thresholds)', async () => {
    vi.mocked(api.getRunReport).mockResolvedValue(mockReport)
    renderInspector()

    await waitFor(() => {
      expect(screen.getByText('GET /cart')).toBeInTheDocument()
    })

    // Switch to Metrics
    fireEvent.click(screen.getByRole('button', { name: /telemetry metrics/i }))
    expect(screen.getByText('vuhive.http.reqs')).toBeInTheDocument()

    // Switch to Thresholds
    fireEvent.click(screen.getByRole('button', { name: /sla thresholds/i }))
    expect(screen.getByText('45ms')).toBeInTheDocument()
    expect(screen.getByText('PASSED')).toBeInTheDocument()
  })

  it('switches between Tabular View and Collapsible JSON Tree Explorer', async () => {
    vi.mocked(api.getRunReport).mockResolvedValue(mockReport)
    renderInspector()

    await waitFor(() => {
      expect(screen.getByText('GET /cart')).toBeInTheDocument()
    })

    // Switch to JSON Tree Explorer
    const jsonViewBtn = screen.getByRole('button', { name: /json tree/i })
    fireEvent.click(jsonViewBtn)

    // JSON tree should now be visible
    expect(screen.getByText(/suite_name/)).toBeInTheDocument()
    expect(screen.getByText(/"Checkout Suite"/)).toBeInTheDocument()

    // Switch back to Tabular View
    const tableViewBtn = screen.getByRole('button', { name: /tabular view/i })
    fireEvent.click(tableViewBtn)
    expect(screen.getByText('GET /cart')).toBeInTheDocument()
  })

  it('triggers Download JSON action', async () => {
    vi.mocked(api.getRunReport).mockResolvedValue(mockReport)
    renderInspector('run-456')

    await waitFor(() => {
      expect(screen.getByText('GET /cart')).toBeInTheDocument()
    })

    const downloadBtn = screen.getByRole('button', { name: /download json/i })
    fireEvent.click(downloadBtn)

    expect(downloadJsonSpy).toHaveBeenCalledWith(mockReport, 'summary-run-456.json')
  })

  it('triggers Export CSV action from tabular view', async () => {
    vi.mocked(api.getRunReport).mockResolvedValue(mockReport)
    renderInspector('run-789')

    await waitFor(() => {
      expect(screen.getByText('GET /cart')).toBeInTheDocument()
    })

    const exportCsvBtn = screen.getByRole('button', { name: /export csv/i })
    fireEvent.click(exportCsvBtn)

    expect(exportCsvSpy).toHaveBeenCalled()
    expect(exportCsvSpy.mock.calls[0][2]).toBe('summary-run-789-steps.csv')
  })

  it('handles error state gracefully', async () => {
    vi.mocked(api.getRunReport).mockRejectedValue(new Error('Report not found in storage'))
    renderInspector('run-err')

    await waitFor(() => {
      expect(screen.getByText(/unable to load telemetry report/i)).toBeInTheDocument()
    })
  })
})
