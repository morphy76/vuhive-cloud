import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ConfigDiffDialog } from '@/components/dialogs/ConfigDiffDialog'
import type { SuiteConfiguration } from '@/types/suite'

describe('ConfigDiffDialog', () => {
  const configs: SuiteConfiguration[] = [
    {
      id: 'cfg-1',
      suiteId: 's-1',
      name: 'baseline.yaml',
      contentYaml: 'vus: 20\nduration: 30s\n',
      isDefault: true,
      createdAt: '2026-09-01T00:00:00Z',
    },
    {
      id: 'cfg-2',
      suiteId: 's-1',
      name: 'stress.yaml',
      contentYaml: 'vus: 100\nduration: 60s\n',
      isDefault: false,
      createdAt: '2026-09-02T00:00:00Z',
    },
  ]

  it('renders diff dialog with selectors and diff comparison', () => {
    render(
      <ConfigDiffDialog
        open={true}
        onOpenChange={vi.fn()}
        configs={configs}
        initialBaseId="cfg-1"
        initialComparisonId="cfg-2"
      />
    )

    expect(screen.getByRole('heading', { name: /compare scenario configurations/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/base configuration/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/comparison configuration/i)).toBeInTheDocument()
    expect(screen.getByText(/vus: 20/i)).toBeInTheDocument()
    expect(screen.getByText(/vus: 100/i)).toBeInTheDocument()
  })

  it('allows swapping base and comparison configurations', () => {
    render(
      <ConfigDiffDialog
        open={true}
        onOpenChange={vi.fn()}
        configs={configs}
        initialBaseId="cfg-1"
        initialComparisonId="cfg-2"
      />
    )

    const swapBtn = screen.getByRole('button', { name: /swap configurations/i })
    fireEvent.click(swapBtn)

    const baseSelect = screen.getByLabelText(/base configuration/i) as HTMLSelectElement
    const compSelect = screen.getByLabelText(/comparison configuration/i) as HTMLSelectElement
    expect(baseSelect.value).toBe('cfg-2')
    expect(compSelect.value).toBe('cfg-1')
  })
})
