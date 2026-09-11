import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'
import type { TestSuite } from '@/types/suite'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export function useSuites(initialData?: TestSuite[]) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['suites'],
      queryFn: () => api.getSuites(),
      initialData,
    },
    client
  )
}

export function useSuite(id: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['suites', id],
      queryFn: () => api.getSuite(id),
      enabled: Boolean(id),
    },
    client
  )
}

export function useCreateSuite() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: { name: string; description?: string }) => api.createSuite(data),
      onSuccess: (newSuite) => {
        queryClient.setQueryData<TestSuite[]>(['suites'], (old = []) => [newSuite, ...old])
        queryClient.invalidateQueries({ queryKey: ['suites'] })
      },
    },
    queryClient
  )
}

export function useUpdateSuite(id: string) {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: { name: string; description?: string; state?: string }) =>
        api.updateSuite(id, data),
      onSuccess: (updatedSuite) => {
        queryClient.setQueryData<TestSuite>(['suites', id], updatedSuite)
        queryClient.invalidateQueries({ queryKey: ['suites'] })
      },
    },
    queryClient
  )
}

export function useDeleteSuite() {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (id: string) => api.deleteSuite(id),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites'] })
      },
    },
    queryClient
  )
}

export function useSuiteConfigs(suiteId: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['suites', suiteId, 'configs'],
      queryFn: () => api.getSuiteConfigs(suiteId),
      enabled: Boolean(suiteId),
    },
    client
  )
}

export function useCreateSuiteConfig(suiteId: string) {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (data: { name: string; content_yaml: string; is_default?: boolean }) =>
        api.createSuiteConfig(suiteId, data),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites', suiteId, 'configs'] })
      },
    },
    queryClient
  )
}

export function useDeleteSuiteConfig(suiteId: string) {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (configId: string) => api.deleteSuiteConfig(suiteId, configId),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites', suiteId, 'configs'] })
      },
    },
    queryClient
  )
}

export function useSuiteArtifacts(suiteId: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['suites', suiteId, 'artifacts'],
      queryFn: () => api.getSuiteArtifacts(suiteId),
      enabled: Boolean(suiteId),
    },
    client
  )
}

export function useUploadSuiteBuild(suiteId: string) {
  const queryClient = useSafeQueryClient()

  return useMutation(
    {
      mutationFn: (formData: FormData) => api.uploadSuiteBuild(suiteId, formData),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites', suiteId, 'artifacts'] })
      },
    },
    queryClient
  )
}

export function useSuiteRuns(suiteId: string) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['suites', suiteId, 'runs'],
      queryFn: () => api.getSuiteRuns(suiteId),
      enabled: Boolean(suiteId),
    },
    client
  )
}
