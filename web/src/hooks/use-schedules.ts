import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'
import type { Schedule, CreateScheduleInput, UpdateScheduleInput } from '@/types/schedule'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export function useSchedules() {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['schedules'],
      queryFn: () => api.getSchedules(),
    },
    client
  )
}

export function useSchedule(id?: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['schedules', id],
      queryFn: () => api.getSchedule(id!),
      enabled: Boolean(id),
    },
    client
  )
}

export function useCreateSchedule() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: CreateScheduleInput) => api.createSchedule(data),
      onSuccess: (newSchedule) => {
        queryClient.setQueryData<Schedule[]>(['schedules'], (old = []) => [...old, newSchedule])
        queryClient.setQueryData<Schedule>(['schedules', newSchedule.id], newSchedule)
        queryClient.invalidateQueries({ queryKey: ['schedules'] })
      },
    },
    queryClient
  )
}

export function useUpdateSchedule() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: ({ id, data }: { id: string; data: UpdateScheduleInput }) =>
        api.updateSchedule(id, data),
      onSuccess: (updated) => {
        queryClient.setQueryData<Schedule>(['schedules', updated.id], updated)
        queryClient.setQueryData<Schedule[]>(['schedules'], (old = []) =>
          old.map((s) => (s.id === updated.id ? updated : s))
        )
        queryClient.invalidateQueries({ queryKey: ['schedules'] })
      },
    },
    queryClient
  )
}

export function useDeleteSchedule() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (id: string) => api.deleteSchedule(id),
      onSuccess: (_, id) => {
        queryClient.setQueryData<Schedule[]>(['schedules'], (old = []) =>
          old.filter((s) => s.id !== id)
        )
        queryClient.removeQueries({ queryKey: ['schedules', id] })
        queryClient.invalidateQueries({ queryKey: ['schedules'] })
      },
    },
    queryClient
  )
}
