import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CreateSecretDialog } from '@/components/dialogs/CreateSecretDialog'
import { EditSecretDialog } from '@/components/dialogs/EditSecretDialog'
import { DeleteSecretDialog } from '@/components/dialogs/DeleteSecretDialog'
import type { SuiteSecret } from '@/types/secret'

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

const mockSecret: SuiteSecret = {
  id: 'sec-123',
  suiteId: 'suite-abc',
  key: 'DATABASE_PASSWORD',
  createdAt: '2026-03-01T10:00:00Z',
  updatedAt: '2026-03-01T10:00:00Z',
}

describe('Secrets Interactive Dialogs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('CreateSecretDialog', () => {
    it('renders key input, secret input, and security disclaimer', () => {
      const Wrapper = createWrapper()
      render(
        <Wrapper>
          <CreateSecretDialog
            suiteId="suite-abc"
            open={true}
            onOpenChange={vi.fn()}
          />
        </Wrapper>
      )

      expect(screen.getByRole('heading', { name: /add suite secret/i })).toBeInTheDocument()
      expect(screen.getByLabelText(/secret key/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/secret value/i)).toBeInTheDocument()
      expect(screen.getByText(/AES-256-GCM/i)).toBeInTheDocument()
    })

    it('validates key format and displays error for lowercase or invalid keys', async () => {
      const Wrapper = createWrapper()
      render(
        <Wrapper>
          <CreateSecretDialog
            suiteId="suite-abc"
            open={true}
            onOpenChange={vi.fn()}
          />
        </Wrapper>
      )

      const keyInput = screen.getByLabelText(/secret key/i)
      fireEvent.change(keyInput, { target: { value: 'lowercase_key' } })

      expect(
        screen.getByText(/key must start with an uppercase letter/i)
      ).toBeInTheDocument()
    })

    it('toggles password visibility between password and text', () => {
      const Wrapper = createWrapper()
      render(
        <Wrapper>
          <CreateSecretDialog
            suiteId="suite-abc"
            open={true}
            onOpenChange={vi.fn()}
          />
        </Wrapper>
      )

      const valueInput = screen.getByLabelText(/secret value/i)
      expect(valueInput).toHaveAttribute('type', 'password')

      const toggleButton = screen.getByRole('button', { name: /toggle secret visibility/i })
      fireEvent.click(toggleButton)
      expect(valueInput).toHaveAttribute('type', 'text')

      fireEvent.click(toggleButton)
      expect(valueInput).toHaveAttribute('type', 'password')
    })
  })

  describe('EditSecretDialog', () => {
    it('renders existing key as read-only and allows entering new value', () => {
      const Wrapper = createWrapper()
      render(
        <Wrapper>
          <EditSecretDialog
            suiteId="suite-abc"
            secret={mockSecret}
            open={true}
            onOpenChange={vi.fn()}
          />
        </Wrapper>
      )

      expect(screen.getByText('DATABASE_PASSWORD')).toBeInTheDocument()
      expect(screen.getByLabelText(/new secret value/i)).toBeInTheDocument()
      expect(screen.getByText(/AES-256-GCM/i)).toBeInTheDocument()
    })
  })

  describe('DeleteSecretDialog', () => {
    it('renders warning message about ${secrets.KEY} placeholder failures and invokes onConfirm', async () => {
      const onConfirm = vi.fn().mockResolvedValue(undefined)
      const onOpenChange = vi.fn()

      render(
        <DeleteSecretDialog
          secret={mockSecret}
          open={true}
          onOpenChange={onOpenChange}
          onConfirm={onConfirm}
        />
      )

      expect(screen.getByRole('heading', { name: /delete suite secret/i })).toBeInTheDocument()
      expect(screen.getByText('DATABASE_PASSWORD')).toBeInTheDocument()
      expect(screen.getAllByText(/\$\{secrets\.DATABASE_PASSWORD\}/i).length).toBeGreaterThanOrEqual(1)

      const deleteButton = screen.getByRole('button', { name: /^delete secret$/i })
      fireEvent.click(deleteButton)

      await waitFor(() => {
        expect(onConfirm).toHaveBeenCalledTimes(1)
      })
    })
  })
})
