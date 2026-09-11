import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'
import type { HistoricalRun, TriggerRunInput } from '@/types/suite'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export function useRuns(filter?: { suiteId?: string; status?: string; scheduleId?: string }) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['runs', filter],
      queryFn: () => api.getRuns(filter),
    },
    client
  )
}

export function useRun(id?: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['runs', id],
      queryFn: () => api.getRun(id!),
      enabled: Boolean(id),
    },
    client
  )
}

export function useTriggerRun() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: TriggerRunInput) => api.triggerRun(data),
      onSuccess: (newRun) => {
        queryClient.setQueryData<HistoricalRun[]>(['runs', undefined], (old = []) => [newRun, ...old])
        queryClient.setQueryData<HistoricalRun>(['runs', newRun.id], newRun)
        queryClient.invalidateQueries({ queryKey: ['runs'] })
        if (newRun.suiteId) {
          queryClient.invalidateQueries({ queryKey: ['suites', newRun.suiteId, 'runs'] })
        }
      },
    },
    queryClient
  )
}

export function useAbortRun() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
        api.abortRun(id, reason),
      onSuccess: (abortedRun) => {
        queryClient.setQueryData<HistoricalRun>(['runs', abortedRun.id], abortedRun)
        queryClient.setQueryData<HistoricalRun[]>(['runs', undefined], (old = []) =>
          old.map((r) => (r.id === abortedRun.id ? abortedRun : r))
        )
        queryClient.invalidateQueries({ queryKey: ['runs'] })
      },
    },
    queryClient
  )
}
