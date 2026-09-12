import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { axe } from 'vitest-axe'
import { DeleteArtifactDialog } from '@/components/dialogs/DeleteArtifactDialog'
import type { CompiledArtifact } from '@/types/suite'

const mockArtifact: CompiledArtifact = {
  id: 'art-checkout-amd64',
  suiteId: 'suite-1',
  platform: 'linux/amd64',
  status: 'READY',
  s3BinaryKey: 'suites/suite-1/artifacts/art-checkout-amd64/loadtest',
  sha256Checksum: 'abcdef1234567890',
  createdAt: '2026-03-01T10:00:00Z',
}

describe('DeleteArtifactDialog', () => {
  it('renders destructive warning, artifact details, and handles confirm', () => {
    const onConfirm = vi.fn()
    const onOpenChange = vi.fn()

    render(
      <DeleteArtifactDialog
        open={true}
        onOpenChange={onOpenChange}
        artifact={mockArtifact}
        onConfirm={onConfirm}
      />
    )

    expect(screen.getByRole('heading', { name: /delete artifact/i })).toBeInTheDocument()
    expect(screen.getAllByText(/art-checkout-amd64/i).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/linux\/amd64/i)).toBeInTheDocument()
    expect(screen.getByText(/READY/i)).toBeInTheDocument()
    expect(screen.getByText(/suites\/suite-1\/artifacts\/art-checkout-amd64\/loadtest/i)).toBeInTheDocument()

    // Irreversible warning
    expect(screen.getByText(/cannot be undone/i)).toBeInTheDocument()

    const deleteBtn = screen.getByRole('button', { name: 'Delete Artifact' })
    expect(deleteBtn).toBeEnabled()

    fireEvent.click(deleteBtn)
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('renders nothing when artifact is null', () => {
    const { container } = render(
      <DeleteArtifactDialog
        open={true}
        onOpenChange={vi.fn()}
        artifact={null}
        onConfirm={vi.fn()}
      />
    )

    expect(container).toBeEmptyDOMElement()
  })

  it('has zero accessibility violations', async () => {
    const { container } = render(
      <DeleteArtifactDialog
        open={true}
        onOpenChange={vi.fn()}
        artifact={mockArtifact}
        onConfirm={vi.fn()}
      />
    )

    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})
