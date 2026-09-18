import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api, ApiError } from '@/lib/api'
import { queryClient } from '@/lib/query-client'

describe('PWA HTTP Resilience and Circuit Breaker Governance', () => {
  const originalFetch = global.fetch
  const originalOnLine = navigator.onLine

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    global.fetch = originalFetch
    Object.defineProperty(navigator, 'onLine', {
      value: originalOnLine,
      configurable: true,
    })
  })

  it('preserves status code on ApiError', () => {
    const err = new ApiError('upstream service circuit open', 503)
    expect(err.name).toBe('ApiError')
    expect(err.status).toBe(503)
    expect(err.message).toBe('upstream service circuit open')
  })

  it('blocks mutating actions when offline and throws clear offline error', async () => {
    Object.defineProperty(navigator, 'onLine', {
      value: false,
      configurable: true,
    })

    await expect(api.createSuite({ name: 'Offline Suite' })).rejects.toThrow(
      /network connection unavailable|offline/i
    )

    await expect(api.deleteSuite('suite-123')).rejects.toThrow(
      /network connection unavailable|offline/i
    )
  })

  it('times out hanging network requests after configured timeout budget', async () => {
    vi.useFakeTimers()

    global.fetch = vi.fn().mockImplementation((_url, options) => {
      return new Promise((_resolve, reject) => {
        if (options?.signal) {
          options.signal.addEventListener('abort', () => {
            reject(new Error('Request timed out after 8000ms'))
          })
        }
      })
    })

    const promise = api.getSuites()
    // Attach catch handler immediately to prevent unhandled rejection warning
    const caughtPromise = promise.catch((err) => err)

    // Advance timers past 8000ms
    await vi.advanceTimersByTimeAsync(8500)

    const err = await caughtPromise
    expect(err).toBeInstanceOf(Error)
    expect(err.message).toMatch(/timed out/i)

    vi.useRealTimers()
  })

  it('dispatches vuhive:circuit-breaker event on 503 circuit open response', async () => {
    const eventListener = vi.fn()
    window.addEventListener('vuhive:circuit-breaker', eventListener)

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      json: async () => ({ error: 'upstream service circuit open' }),
    })

    await expect(api.getSuites()).rejects.toThrow('upstream service circuit open')

    expect(eventListener).toHaveBeenCalled()
    const event = eventListener.mock.calls[0][0] as CustomEvent
    expect(event.detail.open).toBe(true)

    window.removeEventListener('vuhive:circuit-breaker', eventListener)
  })

  describe('TanStack Query Retry Policy', () => {
    it('does NOT retry client 4xx errors', () => {
      const defaultQueryOptions = queryClient.getDefaultOptions().queries
      const retryFn = defaultQueryOptions?.retry as (failureCount: number, error: any) => boolean
      expect(retryFn).toBeDefined()

      const err404 = new ApiError('HTTP error 404', 404)
      expect(retryFn(0, err404)).toBe(false)

      const err400 = new Error('HTTP error 400')
      expect(retryFn(0, err400)).toBe(false)
    })

    it('retries 5xx server errors up to 2 times with exponential backoff', () => {
      const defaultQueryOptions = queryClient.getDefaultOptions().queries
      const retryFn = defaultQueryOptions?.retry as (failureCount: number, error: any) => boolean
      const retryDelayFn = defaultQueryOptions?.retryDelay as (attemptIndex: number) => number

      expect(retryFn).toBeDefined()
      expect(retryDelayFn).toBeDefined()

      const err500 = new ApiError('HTTP error 500', 500)
      expect(retryFn(0, err500)).toBe(true)
      expect(retryFn(1, err500)).toBe(true)
      expect(retryFn(2, err500)).toBe(false)

      // Exponential backoff
      expect(retryDelayFn(0)).toBe(1000)
      expect(retryDelayFn(1)).toBe(2000)
      expect(retryDelayFn(2)).toBe(4000)
    })
  })
})
