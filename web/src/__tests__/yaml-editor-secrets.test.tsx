import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { YamlEditor } from '@/components/editor/YamlEditor'
import type { SuiteSecret } from '@/types/secret'

const mockSecrets: SuiteSecret[] = [
  {
    id: 'sec-1',
    suiteId: 'suite-1',
    key: 'API_KEY',
    createdAt: '2026-03-01T10:00:00Z',
    updatedAt: '2026-03-01T10:00:00Z',
  },
  {
    id: 'sec-2',
    suiteId: 'suite-1',
    key: 'STRIPE_SECRET',
    createdAt: '2026-03-01T10:00:00Z',
    updatedAt: '2026-03-01T10:00:00Z',
  },
]

describe('YamlEditor with Secret Insertion', () => {
  it('renders Insert Secret dropdown when availableSecrets are provided', () => {
    render(
      <YamlEditor
        value={"version: \"1.0\""}
        onChange={vi.fn()}
        availableSecrets={mockSecrets}
      />
    )

    expect(screen.getByRole('button', { name: /insert secret/i })).toBeInTheDocument()
  })

  it('displays available secret keys when dropdown is opened', () => {
    render(
      <YamlEditor
        value={"version: \"1.0\""}
        onChange={vi.fn()}
        availableSecrets={mockSecrets}
      />
    )

    const insertBtn = screen.getByRole('button', { name: /insert secret/i })
    fireEvent.click(insertBtn)

    expect(screen.getByText('API_KEY')).toBeInTheDocument()
    expect(screen.getByText('STRIPE_SECRET')).toBeInTheDocument()
  })

  it('inserts ${secrets.KEY} placeholder into editor on selection', () => {
    const onChange = vi.fn()
    render(
      <YamlEditor
        value="vus: 10"
        onChange={onChange}
        availableSecrets={mockSecrets}
      />
    )

    const insertBtn = screen.getByRole('button', { name: /insert secret/i })
    fireEvent.click(insertBtn)

    const apiKeyOption = screen.getByText('API_KEY')
    fireEvent.click(apiKeyOption)

    expect(onChange).toHaveBeenCalled()
    // The change should include ${secrets.API_KEY}
    const lastCallArg = onChange.mock.calls[onChange.mock.calls.length - 1][0]
    expect(lastCallArg).toContain('${secrets.API_KEY}')
  })
})
