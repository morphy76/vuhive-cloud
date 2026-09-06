import { QueryClient } from '@tanstack/react-query'
import type { PersistedClient, Persister } from '@tanstack/react-query-persist-client'
import { get, set, del } from 'idb-keyval'

const IDB_CACHE_KEY = 'vuhive_cloud_query_cache'

// Memory store fallback for test or restricted environments without IndexedDB
const memoryFallbackStore = new Map<string, PersistedClient>()

/**
 * Creates an IndexedDB persister for TanStack Query utilizing idb-keyval.
 * Gracefully falls back to an in-memory store in non-browser or test runtimes.
 */
export function createIDBPersister(key: string = IDB_CACHE_KEY): Persister {
  const isIndexedDBSupported =
    typeof window !== 'undefined' &&
    typeof window.indexedDB !== 'undefined' &&
    window.indexedDB !== null

  return {
    persistClient: async (persistedClient: PersistedClient): Promise<void> => {
      if (!isIndexedDBSupported) {
        memoryFallbackStore.set(key, persistedClient)
        return
      }
      try {
        await set(key, persistedClient)
      } catch (error) {
        console.warn('Failed persisting query client cache to IndexedDB:', error)
      }
    },
    restoreClient: () => {
      if (!isIndexedDBSupported) {
        return memoryFallbackStore.get(key)
      }
      return get<PersistedClient>(key).catch((error) => {
        console.warn('Failed restoring query client cache from IndexedDB:', error)
        return undefined
      })
    },
    removeClient: async (): Promise<void> => {
      if (!isIndexedDBSupported) {
        memoryFallbackStore.delete(key)
        return
      }
      try {
        await del(key)
      } catch (error) {
        console.warn('Failed removing query client cache from IndexedDB:', error)
      }
    },
  }
}

/**
 * Standard QueryClient configured for offline-first resilience.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Keep cached data in storage for 24 hours
      gcTime: 1000 * 60 * 60 * 24,
      // Consider query data fresh for 5 minutes
      staleTime: 1000 * 60 * 5,
      // Attempt requests immediately, but fallback to cache seamlessly when offline
      networkMode: 'offlineFirst',
      // Retry transient network errors once
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

export const idbPersister = createIDBPersister()
