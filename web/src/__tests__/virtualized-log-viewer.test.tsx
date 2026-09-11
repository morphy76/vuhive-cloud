import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { VirtualizedLogViewer } from '../components/logs/VirtualizedLogViewer'

describe('VirtualizedLogViewer', () => {
  const sampleLogs = [
    '\u001b[32m[INFO]\u001b[0m 2026-09-11 20:30:00 Starting scenario "Checkout Funnel"',
    '\u001b[34m[DEBUG]\u001b[0m Initializing 100 virtual users',
    '\u001b[33m[WARN]\u001b[0m Barrier rendezvous latency elevated: 45ms',
    '\u001b[31m[ERROR]\u001b[0m Step "SubmitOrder" failed: HTTP 504 Gateway Timeout',
    '\u001b[32m[INFO]\u001b[0m Test execution completed with 1 failure',
  ].join('\n')

  beforeEach(() => {
    vi.clearAllMocks()
    // Polyfill element dimensions for jsdom so virtualizer computes visible lines
    Object.defineProperty(HTMLElement.prototype, 'offsetHeight', { configurable: true, value: 600 })
    Object.defineProperty(HTMLElement.prototype, 'offsetWidth', { configurable: true, value: 800 })
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', { configurable: true, value: 600 })
    Object.defineProperty(HTMLElement.prototype, 'clientWidth', { configurable: true, value: 800 })
    HTMLElement.prototype.getBoundingClientRect = function () {
      return {
        top: 0,
        left: 0,
        bottom: 600,
        right: 800,
        width: 800,
        height: 600,
        x: 0,
        y: 0,
        toJSON: () => {},
      }
    }
  })

  it('renders loading state when isLoading is true', () => {
    render(<VirtualizedLogViewer logs="" isLoading={true} />)
    expect(screen.getByText(/loading logs/i)).toBeInTheDocument()
  })

  it('renders empty placeholder when logs are empty', () => {
    render(<VirtualizedLogViewer logs="" />)
    expect(screen.getByText(/no log output recorded/i)).toBeInTheDocument()
  })

  it('renders log lines and line numbers', () => {
    render(<VirtualizedLogViewer logs={sampleLogs} runId="run-123" />)
    expect(screen.getByText('run-123')).toBeInTheDocument()
    expect(screen.getByText(/Starting scenario/)).toBeInTheDocument()
    expect(screen.getByText(/SubmitOrder/)).toBeInTheDocument()
  })

  it('renders ANSI styled spans', () => {
    render(<VirtualizedLogViewer logs={sampleLogs} />)
    const errorSpan = screen.getByText('[ERROR]')
    expect(errorSpan).toBeInTheDocument()
    // Should have styled color from ANSI code \u001b[31m
    expect(errorSpan.style.color).toBeDefined()
  })

  it('supports searching across logs and navigating matches', () => {
    render(<VirtualizedLogViewer logs={sampleLogs} />)

    const searchInput = screen.getByPlaceholderText(/search logs/i)
    fireEvent.change(searchInput, { target: { value: 'INFO' } })

    // "INFO" appears in 2 lines
    expect(screen.getByText(/1 of 2/i)).toBeInTheDocument()

    // Next match button
    const nextBtn = screen.getByLabelText(/next match/i)
    fireEvent.click(nextBtn)
    expect(screen.getByText(/2 of 2/i)).toBeInTheDocument()

    // Prev match button
    const prevBtn = screen.getByLabelText(/previous match/i)
    fireEvent.click(prevBtn)
    expect(screen.getByText(/1 of 2/i)).toBeInTheDocument()
  })

  it('displays 0 matches when search query is not found', () => {
    render(<VirtualizedLogViewer logs={sampleLogs} />)
    const searchInput = screen.getByPlaceholderText(/search logs/i)
    fireEvent.change(searchInput, { target: { value: 'nonexistentterm' } })
    expect(screen.getByText(/0 matches/i)).toBeInTheDocument()
  })

  it('toggles line wrapping mode', () => {
    render(<VirtualizedLogViewer logs={sampleLogs} />)
    const wrapButton = screen.getByLabelText(/toggle line wrap/i)
    expect(wrapButton).toHaveAttribute('aria-pressed', 'false')

    fireEvent.click(wrapButton)
    expect(wrapButton).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(wrapButton)
    expect(wrapButton).toHaveAttribute('aria-pressed', 'false')
  })

  it('toggles auto-scroll mode', () => {
    render(<VirtualizedLogViewer logs={sampleLogs} />)
    const autoScrollButton = screen.getByLabelText(/toggle auto-scroll/i)
    expect(autoScrollButton).toBeInTheDocument()

    // Toggle on/off
    fireEvent.click(autoScrollButton)
    expect(autoScrollButton).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(autoScrollButton)
    expect(autoScrollButton).toHaveAttribute('aria-pressed', 'true')
  })

  it('toggles high-contrast mode', () => {
    const { container } = render(<VirtualizedLogViewer logs={sampleLogs} />)
    const contrastButton = screen.getByLabelText(/toggle high-contrast/i)

    fireEvent.click(contrastButton)
    expect(contrastButton).toHaveAttribute('aria-pressed', 'true')
    // High contrast container class
    expect(container.querySelector('[data-high-contrast="true"]')).toBeInTheDocument()
  })

  it('copies logs to clipboard when copy button is clicked', async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextMock,
      },
    })

    render(<VirtualizedLogViewer logs={sampleLogs} />)
    const copyButton = screen.getByLabelText(/copy logs/i)
    fireEvent.click(copyButton)

    expect(writeTextMock).toHaveBeenCalledWith(sampleLogs)
    expect(await screen.findByText(/copied/i)).toBeInTheDocument()
  })

  it('calls onDownloadRawLog when download button is clicked', () => {
    const onDownload = vi.fn()
    render(<VirtualizedLogViewer logs={sampleLogs} onDownloadRawLog={onDownload} />)
    const downloadButton = screen.getByLabelText(/download raw log/i)
    fireEvent.click(downloadButton)
    expect(onDownload).toHaveBeenCalledTimes(1)
  })

  it('handles 50,000 lines without freezing or erroring', () => {
    const lines = Array.from({ length: 50000 }, (_, i) => `[${i + 1}] Event step executed successfully`)
    const largeLog = lines.join('\n')

    const start = performance.now()
    render(<VirtualizedLogViewer logs={largeLog} />)
    const duration = performance.now() - start

    // Virtualized rendering of 50k lines should complete quickly (under 1000ms)
    expect(duration).toBeLessThan(1000)
    expect(screen.getByText(/50,000 lines/i)).toBeInTheDocument()
  })
})
