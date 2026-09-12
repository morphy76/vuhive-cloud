import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { axe } from 'vitest-axe'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { UploadBuildDialog } from '@/components/dialogs/UploadBuildDialog'
import { BuildStatusStepper } from '@/components/build/BuildStatusStepper'
import { BuildLogViewer } from '@/components/build/BuildLogViewer'
import { TooltipProvider } from '@/components/ui/tooltip'
import { api } from '@/lib/api'

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
        <TooltipProvider>{children}</TooltipProvider>
      </QueryClientProvider>
    )
  }
}

describe('UploadBuildDialog & Drag-and-Drop Build Workflow', () => {
  const mockOnOpenChange = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders dropzone, platform options, and security switch', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    expect(screen.getByText(/Upload Source & Trigger Build/i)).toBeInTheDocument()
    expect(screen.getByText(/Drag & drop your Go scenario archive/i)).toBeInTheDocument()
    expect(screen.getByText(/linux\/amd64/i)).toBeInTheDocument()
    expect(screen.getByText(/linux\/arm64/i)).toBeInTheDocument()
    expect(screen.getByText(/Allow Insecure Imports/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).toBeDisabled()
  })

  it('accepts valid .tar.gz file via file input and enables submit button', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['scenario content'], 'scenario.tar.gz', {
      type: 'application/gzip',
    })
    const input = screen.getByTestId('source-file-input') as HTMLInputElement

    fireEvent.change(input, { target: { files: [file] } })

    expect(screen.getByText('scenario.tar.gz')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).not.toBeDisabled()
  })

  it('accepts valid .zip file via drag and drop', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const dropZone = screen.getByTestId('dropzone')
    const file = new File(['zip content'], 'loadtest.zip', {
      type: 'application/zip',
    })

    fireEvent.dragEnter(dropZone)
    fireEvent.dragOver(dropZone)
    fireEvent.drop(dropZone, {
      dataTransfer: { files: [file] },
    })

    expect(screen.getByText('loadtest.zip')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).not.toBeDisabled()
  })

  it('accepts valid .tar.bz2 file via file input and enables submit button', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['bzip2 content'], 'scenario.tar.bz2', {
      type: 'application/x-bzip2',
    })
    const input = screen.getByTestId('source-file-input') as HTMLInputElement

    fireEvent.change(input, { target: { files: [file] } })

    expect(screen.getByText('scenario.tar.bz2')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).not.toBeDisabled()
  })

  it('accepts valid .tbz2 file via drag and drop', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const dropZone = screen.getByTestId('dropzone')
    const file = new File(['tbz2 content'], 'loadtest.tbz2', {
      type: 'application/x-bzip2',
    })

    fireEvent.dragEnter(dropZone)
    fireEvent.dragOver(dropZone)
    fireEvent.drop(dropZone, {
      dataTransfer: { files: [file] },
    })

    expect(screen.getByText('loadtest.tbz2')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).not.toBeDisabled()
  })

  it('rejects unsupported file formats with validation message', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const dropZone = screen.getByTestId('dropzone')
    const file = new File(['invalid binary'], 'program.exe', {
      type: 'application/octet-stream',
    })

    fireEvent.drop(dropZone, {
      dataTransfer: { files: [file] },
    })

    expect(
      screen.getByText(
        /Unsupported archive format\. Please upload a \.tar\.gz, \.tar\.bz2, or \.zip archive\./i
      )
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).toBeDisabled()
  })

  it('rejects files larger than 50MB with validation error', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const dropZone = screen.getByTestId('dropzone')
    // 55MB file
    const largeFile = new File(['x'.repeat(100)], 'huge-scenario.tar.gz', {
      type: 'application/gzip',
    })
    Object.defineProperty(largeFile, 'size', { value: 55 * 1024 * 1024 })

    fireEvent.drop(dropZone, {
      dataTransfer: { files: [largeFile] },
    })

    expect(screen.getByText(/File size exceeds 50MB limit/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).toBeDisabled()
  })

  it('allows selecting architecture platform', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const arm64Btn = screen.getByRole('button', { name: 'linux/arm64' })
    fireEvent.click(arm64Btn)
    expect(arm64Btn).toHaveAttribute('aria-pressed', 'true')
  })

  it('allows removing selected file', () => {
    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['content'], 'scenario.tar.gz', { type: 'application/gzip' })
    const input = screen.getByTestId('source-file-input')
    fireEvent.change(input, { target: { files: [file] } })

    expect(screen.getByText('scenario.tar.gz')).toBeInTheDocument()
    const removeBtn = screen.getByRole('button', { name: /Remove file/i })
    fireEvent.click(removeBtn)

    expect(screen.queryByText('scenario.tar.gz')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Upload & Build/i })).toBeDisabled()
  })

  it('handles AST rejection error and displays AST error callout with remediation tips', async () => {
    const uploadSpy = vi.spyOn(api, 'uploadSuiteBuild').mockRejectedValueOnce(
      new Error('forbidden package import: os/exec is prohibited')
    )

    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['scenario content'], 'scenario.tar.gz', { type: 'application/gzip' })
    const input = screen.getByTestId('source-file-input')
    fireEvent.change(input, { target: { files: [file] } })

    const submitBtn = screen.getByRole('button', { name: /Upload & Build/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/AST Static Analysis Failed/i)).toBeInTheDocument()
    })

    expect(screen.getByText(/forbidden package import: os\/exec is prohibited/i)).toBeInTheDocument()
    uploadSpy.mockRestore()
  })

  it('transitions to live status stepper upon successful upload', async () => {
    const uploadSpy = vi.spyOn(api, 'uploadSuiteBuild').mockResolvedValueOnce({
      message: 'build triggered successfully',
      artifacts: [
        {
          id: 'art-12345',
          suite_id: 'suite-test-123',
          platform: 'linux/amd64',
          status: 'PENDING',
          created_at: new Date().toISOString(),
        },
      ],
    })

    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['scenario content'], 'scenario.tar.gz', { type: 'application/gzip' })
    const input = screen.getByTestId('source-file-input')
    fireEvent.change(input, { target: { files: [file] } })

    const submitBtn = screen.getByRole('button', { name: /Upload & Build/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByText(/Build Progress/i)).toBeInTheDocument()
    })

    expect(screen.getByText('Queued')).toBeInTheDocument()
    expect(screen.getByText('art-12345')).toBeInTheDocument()
    uploadSpy.mockRestore()
  })

  it('allows cancelling an active build from the dialog', async () => {
    const uploadSpy = vi.spyOn(api, 'uploadSuiteBuild').mockResolvedValueOnce({
      message: 'build triggered successfully',
      artifacts: [
        {
          id: 'art-12345',
          suite_id: 'suite-test-123',
          platform: 'linux/amd64',
          status: 'PENDING',
          created_at: new Date().toISOString(),
        },
      ],
    })
    const cancelSpy = vi.spyOn(api, 'cancelSuiteBuild').mockResolvedValueOnce({
      id: 'art-12345',
      suiteId: 'suite-test-123',
      platform: 'linux/amd64',
      status: 'CANCELLED',
      createdAt: new Date().toISOString(),
    })

    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['scenario content'], 'scenario.tar.gz', { type: 'application/gzip' })
    const input = screen.getByTestId('source-file-input')
    fireEvent.change(input, { target: { files: [file] } })

    const submitBtn = screen.getByRole('button', { name: /Upload & Build/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Cancel Build/i })).toBeInTheDocument()
    })

    const cancelBtn = screen.getByRole('button', { name: /Cancel Build/i })
    fireEvent.click(cancelBtn)

    await waitFor(() => {
      expect(cancelSpy).toHaveBeenCalledWith('suite-test-123', 'art-12345', expect.any(String))
      expect(screen.getAllByText(/Build Cancelled/i).length).toBeGreaterThanOrEqual(1)
    })

    uploadSpy.mockRestore()
    cancelSpy.mockRestore()
  })

  it('allows retrying a failed build from the dialog', async () => {
    const uploadSpy = vi.spyOn(api, 'uploadSuiteBuild').mockResolvedValueOnce({
      message: 'build triggered successfully',
      artifacts: [
        {
          id: 'art-fail-1',
          suite_id: 'suite-test-123',
          platform: 'linux/amd64',
          status: 'FAILED',
          created_at: new Date().toISOString(),
        },
      ],
    })
    const retrySpy = vi.spyOn(api, 'retrySuiteBuild').mockResolvedValueOnce({
      id: 'art-fail-1',
      suiteId: 'suite-test-123',
      platform: 'linux/amd64',
      status: 'BUILDING',
      createdAt: new Date().toISOString(),
    })

    render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const file = new File(['scenario content'], 'scenario.tar.gz', { type: 'application/gzip' })
    const input = screen.getByTestId('source-file-input')
    fireEvent.change(input, { target: { files: [file] } })

    const submitBtn = screen.getByRole('button', { name: /Upload & Build/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Retry Build/i })).toBeInTheDocument()
    })

    const retryBtn = screen.getByRole('button', { name: /Retry Build/i })
    fireEvent.click(retryBtn)

    await waitFor(() => {
      expect(retrySpy).toHaveBeenCalledWith('suite-test-123', 'art-fail-1')
    })

    uploadSpy.mockRestore()
    retrySpy.mockRestore()
  })

  it('has no accessibility violations in initial state', async () => {
    const { container } = render(
      <UploadBuildDialog
        suiteId="suite-test-123"
        open={true}
        onOpenChange={mockOnOpenChange}
      />,
      { wrapper: createTestWrapper() }
    )

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})

describe('BuildStatusStepper', () => {
  it('renders queued status correctly', () => {
    render(<BuildStatusStepper currentStatus="QUEUED" platform="linux/amd64" artifactId="art-123" />)
    expect(screen.getByText('Build Progress')).toBeInTheDocument()
    expect(screen.getByText('linux/amd64')).toBeInTheDocument()
    expect(screen.getByText('art-123')).toBeInTheDocument()
    expect(screen.getByText('Queued')).toBeInTheDocument()
  })

  it('renders building status correctly', () => {
    render(<BuildStatusStepper currentStatus="BUILDING" platform="linux/arm64" />)
    expect(screen.getByText('Compiling')).toBeInTheDocument()
  })

  it('renders ready status correctly', () => {
    render(<BuildStatusStepper currentStatus="READY" platform="linux/amd64" />)
    expect(screen.getByText('Ready')).toBeInTheDocument()
  })

  it('renders failed status with failure indication', () => {
    render(<BuildStatusStepper currentStatus="FAILED" platform="linux/amd64" />)
    expect(screen.getByText('Failed')).toBeInTheDocument()
    expect(screen.getByText('Compilation error')).toBeInTheDocument()
  })

  it('renders cancelled status with cancellation indication', () => {
    render(<BuildStatusStepper currentStatus="CANCELLED" platform="linux/amd64" />)
    expect(screen.getByText('Cancelled')).toBeInTheDocument()
    expect(screen.getByText('Build cancelled')).toBeInTheDocument()
  })
})

describe('BuildLogViewer', () => {
  it('renders log lines and header with line count', () => {
    const logs = 'Line 1: Compiling\nLine 2: Warning\nLine 3: Finished'
    render(<BuildLogViewer logs={logs} title="Build Logs" />)

    expect(screen.getByText('Build Logs')).toBeInTheDocument()
    expect(screen.getByText('(3 lines)')).toBeInTheDocument()
    expect(screen.getByText('Line 1: Compiling')).toBeInTheDocument()
    expect(screen.getByText('Line 3: Finished')).toBeInTheDocument()
  })

  it('toggles log visibility on header click', () => {
    const logs = 'Line 1: Starting compilation...'
    render(<BuildLogViewer logs={logs} defaultExpanded={true} />)

    expect(screen.getByText('Line 1: Starting compilation...')).toBeInTheDocument()
    const toggleBtn = screen.getByRole('button', { name: /Compilation Logs/i })
    fireEvent.click(toggleBtn)

    expect(screen.queryByText('Line 1: Starting compilation...')).not.toBeInTheDocument()
    fireEvent.click(toggleBtn)
    expect(screen.getByText('Line 1: Starting compilation...')).toBeInTheDocument()
  })

  it('copies log content to clipboard', async () => {
    const logs = 'Sample build log lines'
    const writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextMock,
      },
    })

    render(<BuildLogViewer logs={logs} />)

    const copyBtn = screen.getByRole('button', { name: /Copy logs to clipboard/i })
    fireEvent.click(copyBtn)

    expect(writeTextMock).toHaveBeenCalledWith(logs)
    await waitFor(() => {
      expect(screen.getByText('Copied')).toBeInTheDocument()
    })
  })
})

