import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MarkdownViewer } from '../components/markdown/MarkdownViewer'

describe('MarkdownViewer', () => {
  beforeEach(() => {
    // Mock clipboard API
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('renders headings with generated IDs and accessible anchor links', () => {
    const md = `
# Getting Started

## 1. Installation & Setup
`
    render(<MarkdownViewer content={md} />)

    const h1 = screen.getByRole('heading', { level: 1, name: /getting started/i })
    expect(h1).toHaveAttribute('id', 'getting-started')

    const h2 = screen.getByRole('heading', { level: 2, name: /1\. installation & setup/i })
    expect(h2).toHaveAttribute('id', '1-installation--setup')

    const anchorLinks = screen.getAllByLabelText(/link to section/i)
    expect(anchorLinks.length).toBeGreaterThan(0)
    expect(anchorLinks[0]).toHaveAttribute('href', '#getting-started')
  })

  it('renders fenced code block with copy-to-clipboard button and handles click', async () => {
    const md = `
\`\`\`bash
kubectl get pods -n vuhive-system
\`\`\`
`
    render(<MarkdownViewer content={md} />)

    const codeElement = screen.getByText('kubectl get pods -n vuhive-system')
    expect(codeElement).toBeInTheDocument()

    const copyBtn = screen.getByRole('button', { name: /copy code to clipboard/i })
    expect(copyBtn).toBeInTheDocument()

    fireEvent.click(copyBtn)

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      expect.stringContaining('kubectl get pods -n vuhive-system')
    )

    await waitFor(() => {
      expect(screen.getByText(/copied!/i)).toBeInTheDocument()
    })
  })

  it('renders inline code with styling', () => {
    const md = 'Use `vuhive.yaml` to configure test scenarios.'
    render(<MarkdownViewer content={md} />)

    const inlineCode = screen.getByText('vuhive.yaml')
    expect(inlineCode.tagName.toLowerCase()).toBe('code')
  })

  it('renders GitHub callout alerts for [!TIP] and [!CAUTION]', () => {
    const md = `
> [!TIP]
> This is a helpful tip about load testing.

> [!CAUTION]
> Insecure overrides are forbidden.
`
    render(<MarkdownViewer content={md} />)

    expect(screen.getByText('TIP')).toBeInTheDocument()
    expect(screen.getByText('This is a helpful tip about load testing.')).toBeInTheDocument()

    expect(screen.getByText('CAUTION')).toBeInTheDocument()
    expect(screen.getByText('Insecure overrides are forbidden.')).toBeInTheDocument()
  })

  it('renders responsive GFM tables', () => {
    const md = `
| Component | Status | Port |
| :--- | :--- | :--- |
| Control Plane | Running | 8080 |
| PostgreSQL | Ready | 5432 |
`
    render(<MarkdownViewer content={md} />)

    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByText('Component')).toBeInTheDocument()
    expect(screen.getByText('Control Plane')).toBeInTheDocument()
    expect(screen.getByText('5432')).toBeInTheDocument()
  })

  it('renders GFM task list items', () => {
    const md = `
- [x] Completed task
- [ ] Pending task
`
    render(<MarkdownViewer content={md} />)

    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes).toHaveLength(2)
    expect(checkboxes[0]).toBeChecked()
    expect(checkboxes[0]).toBeDisabled()
    expect(checkboxes[1]).not.toBeChecked()
    expect(checkboxes[1]).toBeDisabled()
  })

  it('adds target="_blank" and rel="noopener noreferrer" to external links', () => {
    const md = '[External Site](https://example.com) and [Local Anchor](#section-1)'
    render(<MarkdownViewer content={md} />)

    const externalLink = screen.getByRole('link', { name: 'External Site' })
    expect(externalLink).toHaveAttribute('href', 'https://example.com')
    expect(externalLink).toHaveAttribute('target', '_blank')
    expect(externalLink).toHaveAttribute('rel', 'noopener noreferrer')

    const localLink = screen.getByRole('link', { name: 'Local Anchor' })
    expect(localLink).toHaveAttribute('href', '#section-1')
    expect(localLink).not.toHaveAttribute('target')
  })
})
