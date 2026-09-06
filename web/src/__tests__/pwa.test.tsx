import React from 'react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, act, fireEvent } from '@testing-library/react'
import { axe } from 'vitest-axe'
import { useOnlineStatus } from '@/hooks/use-online-status'
import { useInstallPrompt } from '@/hooks/use-install-prompt'
import { OfflineBanner } from '@/components/ui/offline-banner'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import { InstallButton } from '@/components/ui/install-button'

// Helper component for testing useOnlineStatus
const OnlineStatusTestComponent: React.FC = () => {
  const { isOnline } = useOnlineStatus()
  return (
    <div>
      <span data-testid="status">{isOnline ? 'ONLINE' : 'OFFLINE'}</span>
      <OfflineBanner />
      <OfflinePreviewBadge />
    </div>
  )
}

// Helper component for testing useInstallPrompt
const InstallPromptTestComponent: React.FC = () => {
  const { canInstall, isInstalled } = useInstallPrompt()
  return (
    <div>
      <span data-testid="can-install">{canInstall ? 'CAN_INSTALL' : 'NO_INSTALL'}</span>
      <span data-testid="is-installed">{isInstalled ? 'INSTALLED' : 'NOT_INSTALLED'}</span>
      <InstallButton />
    </div>
  )
}

describe('PWA & Offline Shell Capabilities', () => {
  let originalOnLine: boolean

  beforeEach(() => {
    originalOnLine = navigator.onLine
    vi.restoreAllMocks()
  })

  afterEach(() => {
    Object.defineProperty(navigator, 'onLine', {
      value: originalOnLine,
      writable: true,
      configurable: true,
    })
  })

  describe('useOnlineStatus & Offline Indicators', () => {
    it('detects online state initially when navigator.onLine is true', () => {
      Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
      render(<OnlineStatusTestComponent />)

      expect(screen.getByTestId('status')).toHaveTextContent('ONLINE')
      expect(screen.queryByRole('status')).not.toBeInTheDocument()
      expect(screen.queryByText(/Offline Preview/i)).not.toBeInTheDocument()
    })

    it('reacts to window offline and online events and renders banner', () => {
      Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
      render(<OnlineStatusTestComponent />)

      expect(screen.getByTestId('status')).toHaveTextContent('ONLINE')

      // Trigger offline event
      act(() => {
        Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })
        window.dispatchEvent(new Event('offline'))
      })

      expect(screen.getByTestId('status')).toHaveTextContent('OFFLINE')
      expect(screen.getByRole('status')).toBeInTheDocument()
      expect(screen.getByText(/Offline Mode/i)).toBeInTheDocument()
      expect(screen.getByText(/Offline Preview/i)).toBeInTheDocument()

      // Trigger online event
      act(() => {
        Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
        window.dispatchEvent(new Event('online'))
      })

      expect(screen.getByTestId('status')).toHaveTextContent('ONLINE')
      expect(screen.queryByRole('status')).not.toBeInTheDocument()
      expect(screen.queryByText(/Offline Preview/i)).not.toBeInTheDocument()
    })

    it('conforms to WCAG accessibility guidelines when offline banner is displayed', async () => {
      Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })
      const { container } = render(
        <main>
          <OfflineBanner />
          <OfflinePreviewBadge />
        </main>
      )

      const results = await axe(container)
      expect(results).toHaveNoViolations()
    })
  })

  describe('useInstallPrompt & InstallButton', () => {
    it('remains hidden when beforeinstallprompt has not fired', () => {
      render(<InstallPromptTestComponent />)

      expect(screen.getByTestId('can-install')).toHaveTextContent('NO_INSTALL')
      expect(screen.queryByRole('button', { name: /install/i })).not.toBeInTheDocument()
    })

    it('reveals install button when beforeinstallprompt is triggered and handles prompt call', async () => {
      render(<InstallPromptTestComponent />)

      const promptMock = vi.fn().mockResolvedValue(undefined)
      const userChoicePromise = Promise.resolve({ outcome: 'accepted', platform: 'web' })

      const installEvent = new Event('beforeinstallprompt') as any
      installEvent.prompt = promptMock
      installEvent.userChoice = userChoicePromise
      installEvent.preventDefault = vi.fn()

      act(() => {
        window.dispatchEvent(installEvent)
      })

      expect(screen.getByTestId('can-install')).toHaveTextContent('CAN_INSTALL')
      const button = screen.getByRole('button', { name: /install/i })
      expect(button).toBeInTheDocument()

      // Click install button
      await act(async () => {
        fireEvent.click(button)
      })

      expect(promptMock).toHaveBeenCalledTimes(1)
    })

    it('hides install button when appinstalled event fires', () => {
      render(<InstallPromptTestComponent />)

      const installEvent = new Event('beforeinstallprompt') as any
      installEvent.prompt = vi.fn().mockResolvedValue(undefined)
      installEvent.userChoice = Promise.resolve({ outcome: 'accepted' })
      installEvent.preventDefault = vi.fn()

      act(() => {
        window.dispatchEvent(installEvent)
      })

      expect(screen.getByRole('button', { name: /install/i })).toBeInTheDocument()

      act(() => {
        window.dispatchEvent(new Event('appinstalled'))
      })

      expect(screen.getByTestId('is-installed')).toHaveTextContent('INSTALLED')
      expect(screen.queryByRole('button', { name: /install/i })).not.toBeInTheDocument()
    })

    it('conforms to WCAG accessibility guidelines when install button is present', async () => {
      const { container } = render(<InstallPromptTestComponent />)

      const installEvent = new Event('beforeinstallprompt') as any
      installEvent.prompt = vi.fn().mockResolvedValue(undefined)
      installEvent.userChoice = Promise.resolve({ outcome: 'accepted' })
      installEvent.preventDefault = vi.fn()

      act(() => {
        window.dispatchEvent(installEvent)
      })

      const results = await axe(container)
      expect(results).toHaveNoViolations()
    })
  })

  describe('TanStack Query Offline Persistence', () => {
    it('initializes queryClient with offlineFirst networkMode and long gcTime', async () => {
      const { queryClient, createIDBPersister } = await import('@/lib/query-client')
      const defaultOptions = queryClient.getDefaultOptions()

      expect(defaultOptions.queries?.networkMode).toBe('offlineFirst')
      expect(defaultOptions.queries?.gcTime).toBeGreaterThanOrEqual(1000 * 60 * 60 * 24)
      expect(defaultOptions.queries?.staleTime).toBeGreaterThanOrEqual(1000 * 60 * 5)

      const persister = createIDBPersister('test-key')
      const dummyClient: any = {
        timestamp: Date.now(),
        buster: 'v1',
        clientState: { queries: [], mutations: [] },
      }

      await persister.persistClient(dummyClient)
      const restored = await persister.restoreClient()
      expect(restored).toEqual(dummyClient)

      await persister.removeClient()
      const afterRemove = await persister.restoreClient()
      expect(afterRemove).toBeUndefined()
    })
  })
})

