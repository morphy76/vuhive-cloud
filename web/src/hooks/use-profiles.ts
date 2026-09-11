import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'
import type { RunnerProfile, CreateProfileInput, UpdateProfileInput } from '@/types/profile'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export function useProfiles(initialData?: RunnerProfile[]) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['profiles'],
      queryFn: () => api.getProfiles(),
      initialData,
    },
    client
  )
}

export function useProfile(id: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['profiles', id],
      queryFn: () => api.getProfile(id),
      enabled: Boolean(id),
    },
    client
  )
}

export function useCreateProfile() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: CreateProfileInput) => api.createProfile(data),
      onSuccess: (newProfile) => {
        queryClient.setQueryData<RunnerProfile[]>(['profiles'], (old = []) => [newProfile, ...old])
        queryClient.invalidateQueries({ queryKey: ['profiles'] })
      },
    },
    queryClient
  )
}

export function useUpdateProfile(id: string) {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: UpdateProfileInput) => api.updateProfile(id, data),
      onSuccess: (updatedProfile) => {
        queryClient.setQueryData<RunnerProfile>(['profiles', id], updatedProfile)
        queryClient.invalidateQueries({ queryKey: ['profiles'] })
      },
    },
    queryClient
  )
}

export function useDeleteProfile() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (id: string) => api.deleteProfile(id),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['profiles'] })
      },
    },
    queryClient
  )
}
