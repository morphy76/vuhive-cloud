import * as React from 'react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { JsonTreeViewer } from '@/components/runs/JsonTreeViewer'

describe('JsonTreeViewer', () => {
  const sampleData = {
    suite_name: 'Checkout Suite',
    passed: true,
    total_requests: 6000,
    metrics: [
      { name: 'vuhive.http.req_duration', type: 'duration', p95: 35.0 },
      { name: 'vuhive.http.req_failed', type: 'rate', rate: 0.005 },
    ],
    thresholds: [
      { metric: 'vuhive.http.req_duration', passed: true },
    ],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders root keys and formatted primitive values', () => {
    render(<JsonTreeViewer data={sampleData} />)

    expect(screen.getByText(/suite_name/)).toBeInTheDocument()
    expect(screen.getByText(/"Checkout Suite"/)).toBeInTheDocument()
    expect(screen.getByText(/passed/)).toBeInTheDocument()
    expect(screen.getByText(/true/)).toBeInTheDocument()
    expect(screen.getByText(/6000/)).toBeInTheDocument()
  })

  it('collapses and expands nodes when clicking chevron or key', () => {
    render(<JsonTreeViewer data={sampleData} initialExpandedDepth={3} />)

    // Initially 2 instances (one in metrics, one in thresholds)
    expect(screen.getAllByText(/vuhive\.http\.req_duration/).length).toBe(2)

    // Find collapse button for metrics
    const collapseMetricsBtn = screen.getByRole('button', { name: /^collapse metrics$/i })
    expect(collapseMetricsBtn).toBeInTheDocument()

    // Toggle collapse
    fireEvent.click(collapseMetricsBtn)
    expect(screen.getAllByText(/vuhive\.http\.req_duration/).length).toBe(1)

    // Find expand button for metrics
    const expandMetricsBtn = screen.getByRole('button', { name: /^expand metrics$/i })
    expect(expandMetricsBtn).toBeInTheDocument()

    // Toggle expand again
    fireEvent.click(expandMetricsBtn)
    expect(screen.getAllByText(/vuhive\.http\.req_duration/).length).toBe(2)
  })

  it('supports "Expand All" and "Collapse All" global controls', () => {
    render(<JsonTreeViewer data={sampleData} initialExpandedDepth={0} />)

    const expandAllBtn = screen.getByRole('button', { name: /expand all/i })
    fireEvent.click(expandAllBtn)

    expect(screen.getAllByText(/vuhive\.http\.req_duration/).length).toBeGreaterThan(0)

    const collapseAllBtn = screen.getByRole('button', { name: /collapse all/i })
    fireEvent.click(collapseAllBtn)

    expect(screen.queryByText(/vuhive\.http\.req_duration/)).not.toBeInTheDocument()
  })

  it('filters and highlights matching search query', () => {
    render(<JsonTreeViewer data={sampleData} />)

    const searchInput = screen.getByPlaceholderText(/search json keys or values/i)
    fireEvent.change(searchInput, { target: { value: 'req_duration' } })

    // Matches should be found and highlighted
    expect(screen.getAllByText(/req_duration/).length).toBeGreaterThan(0)
    // Match counter badge
    expect(screen.getByText(/match/i)).toBeInTheDocument()
  })

  it('copies JSON path to clipboard', async () => {
    render(<JsonTreeViewer data={sampleData} initialExpandedDepth={3} />)

    const copyPathBtns = screen.getAllByRole('button', { name: /copy path/i })
    expect(copyPathBtns.length).toBeGreaterThan(0)

    await React.act(async () => {
      fireEvent.click(copyPathBtns[0])
    })
    expect(navigator.clipboard.writeText).toHaveBeenCalled()
  })

  it('copies full JSON payload to clipboard', async () => {
    render(<JsonTreeViewer data={sampleData} />)

    const copyAllBtn = screen.getByRole('button', { name: /copy json/i })
    await React.act(async () => {
      fireEvent.click(copyAllBtn)
    })

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      JSON.stringify(sampleData, null, 2)
    )
  })
})
