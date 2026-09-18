import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  listSuiteSecrets,
  createSuiteSecret,
  updateSuiteSecret,
  deleteSuiteSecret,
  api,
} from '@/lib/api'
import {
  useSuiteSecrets,
  useCreateSuiteSecret,
  useUpdateSuiteSecret,
  useDeleteSuiteSecret,
} from '@/hooks/use-secrets'

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )

  return { queryClient, Wrapper }
}

describe('Secrets API and Hooks', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  describe('Direct API methods', () => {
    it('listSuiteSecrets fetches and maps secrets list', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          secrets: [
            {
              id: 'sec-1',
              suite_id: 'suite-1',
              key: 'API_KEY',
              created_at: '2026-03-01T10:00:00Z',
              updated_at: '2026-03-01T10:00:00Z',
            },
          ],
          count: 1,
        }),
      } as Response)

      const res = await listSuiteSecrets('suite-1')
      expect(res).toEqual([
        {
          id: 'sec-1',
          suiteId: 'suite-1',
          key: 'API_KEY',
          createdAt: '2026-03-01T10:00:00Z',
          updatedAt: '2026-03-01T10:00:00Z',
        },
      ])
      expect(api.listSuiteSecrets).toBeDefined()
    })

    it('createSuiteSecret sends POST request and returns created secret', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => ({
          id: 'sec-2',
          suite_id: 'suite-1',
          key: 'AUTH_TOKEN',
          created_at: '2026-03-02T10:00:00Z',
          updated_at: '2026-03-02T10:00:00Z',
        }),
      } as Response)

      const res = await createSuiteSecret('suite-1', {
        key: 'AUTH_TOKEN',
        value: 'super-secret',
      })
      expect(res.key).toBe('AUTH_TOKEN')
      expect(res.id).toBe('sec-2')
    })

    it('updateSuiteSecret sends PUT request and returns updated secret', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          id: 'sec-1',
          suite_id: 'suite-1',
          key: 'API_KEY',
          created_at: '2026-03-01T10:00:00Z',
          updated_at: '2026-03-03T12:00:00Z',
        }),
      } as Response)

      const res = await updateSuiteSecret('suite-1', 'sec-1', {
        value: 'new-key-value',
      })
      expect(res.id).toBe('sec-1')
      expect(res.updatedAt).toBe('2026-03-03T12:00:00Z')
    })

    it('deleteSuiteSecret sends DELETE request', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
      } as Response)

      await expect(deleteSuiteSecret('suite-1', 'sec-1')).resolves.toBeUndefined()
    })
  })

  describe('TanStack Query hooks', () => {
    it('useSuiteSecrets fetches secrets through query', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          secrets: [
            {
              id: 'sec-1',
              suite_id: 'suite-1',
              key: 'API_KEY',
              created_at: '2026-03-01T10:00:00Z',
              updated_at: '2026-03-01T10:00:00Z',
            },
          ],
          count: 1,
        }),
      } as Response)

      const { Wrapper } = createWrapper()
      const { result } = renderHook(() => useSuiteSecrets('suite-1'), {
        wrapper: Wrapper,
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))
      expect(result.current.data).toHaveLength(1)
      expect(result.current.data?.[0].key).toBe('API_KEY')
    })

    it('useCreateSuiteSecret mutates and invalidates cache', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => ({
          id: 'sec-new',
          suite_id: 'suite-1',
          key: 'NEW_KEY',
          created_at: '2026-03-01T10:00:00Z',
          updated_at: '2026-03-01T10:00:00Z',
        }),
      } as Response)

      const { queryClient, Wrapper } = createWrapper()
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

      const { result } = renderHook(() => useCreateSuiteSecret('suite-1'), {
        wrapper: Wrapper,
      })

      await result.current.mutateAsync({ key: 'NEW_KEY', value: 'val' })

      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ['suites', 'suite-1', 'secrets'],
      })
    })

    it('useUpdateSuiteSecret mutates and invalidates cache', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          id: 'sec-1',
          suite_id: 'suite-1',
          key: 'API_KEY',
          created_at: '2026-03-01T10:00:00Z',
          updated_at: '2026-03-01T12:00:00Z',
        }),
      } as Response)

      const { queryClient, Wrapper } = createWrapper()
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

      const { result } = renderHook(() => useUpdateSuiteSecret('suite-1'), {
        wrapper: Wrapper,
      })

      await result.current.mutateAsync({ secretId: 'sec-1', payload: { value: 'newval' } })

      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ['suites', 'suite-1', 'secrets'],
      })
    })

    it('useDeleteSuiteSecret mutates and invalidates cache', async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
      } as Response)

      const { queryClient, Wrapper } = createWrapper()
      const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')

      const { result } = renderHook(() => useDeleteSuiteSecret('suite-1'), {
        wrapper: Wrapper,
      })

      await result.current.mutateAsync('sec-1')

      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ['suites', 'suite-1', 'secrets'],
      })
    })
  })
})
