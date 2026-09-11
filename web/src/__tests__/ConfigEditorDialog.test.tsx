import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ConfigEditorDialog } from '@/components/dialogs/ConfigEditorDialog'
import { api } from '@/lib/api'
import type { SuiteConfiguration } from '@/types/suite'

import { TooltipProvider } from '@/components/ui/tooltip'

vi.mock('@/lib/api', () => ({
  api: {
    createSuiteConfig: vi.fn().mockResolvedValue({
      id: 'cfg-new',
      suiteId: 's-1',
      name: 'new-profile.yaml',
      contentYaml: 'vus: 50\n',
      isDefault: false,
      createdAt: new Date().toISOString(),
    }),
    updateSuiteConfig: vi.fn().mockResolvedValue({
      id: 'cfg-1',
      suiteId: 's-1',
      name: 'updated-profile.yaml',
      contentYaml: 'vus: 100\n',
      isDefault: true,
      createdAt: new Date().toISOString(),
    }),
  },
}))

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={0}>{ui}</TooltipProvider>
    </QueryClientProvider>
  )
}

describe('ConfigEditorDialog', () => {
  const existingConfig: SuiteConfiguration = {
    id: 'cfg-1',
    suiteId: 's-1',
    name: 'existing.yaml',
    contentYaml: 'version: "1.0"\nexecution:\n  vus: 50\n',
    isDefault: false,
    createdAt: new Date().toISOString(),
  }

  it('renders in create mode with empty/default fields and template support', () => {
    renderWithClient(
      <ConfigEditorDialog
        suiteId="s-1"
        open={true}
        onOpenChange={vi.fn()}
      />
    )

    expect(screen.getByRole('heading', { name: /attach scenario configuration/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/configuration name/i)).toBeInTheDocument()
    expect(screen.getByText(/load template/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /attach configuration/i })).toBeInTheDocument()
  })

  it('renders in edit mode with populated fields and diff tab', () => {
    renderWithClient(
      <ConfigEditorDialog
        suiteId="s-1"
        open={true}
        onOpenChange={vi.fn()}
        initialConfig={existingConfig}
      />
    )

    expect(screen.getByRole('heading', { name: /edit configuration/i })).toBeInTheDocument()
    expect(screen.getByDisplayValue('existing.yaml')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /editor/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /diff/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument()
  })

  it('submits update mutation when saving changes in edit mode', async () => {
    const onOpenChange = vi.fn()
    renderWithClient(
      <ConfigEditorDialog
        suiteId="s-1"
        open={true}
        onOpenChange={onOpenChange}
        initialConfig={existingConfig}
      />
    )

    const saveBtn = screen.getByRole('button', { name: /save changes/i })
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(api.updateSuiteConfig).toHaveBeenCalledWith(
        's-1',
        'cfg-1',
        expect.objectContaining({
          name: 'existing.yaml',
        })
      )
      expect(onOpenChange).toHaveBeenCalledWith(false)
    })
  })
})
