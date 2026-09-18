import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listSuiteSecrets,
  createSuiteSecret,
  updateSuiteSecret,
  deleteSuiteSecret,
} from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'
import type { SuiteSecret, CreateSecretPayload, UpdateSecretPayload } from '@/types/secret'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export function useSuiteSecrets(suiteId: string) {
  const client = useSafeQueryClient()
  return useQuery<SuiteSecret[]>(
    {
      queryKey: ['suites', suiteId, 'secrets'],
      queryFn: () => listSuiteSecrets(suiteId),
      enabled: Boolean(suiteId),
    },
    client
  )
}

export function useCreateSuiteSecret(suiteId: string) {
  const queryClient = useSafeQueryClient()
  return useMutation(
    {
      mutationFn: (payload: CreateSecretPayload) => createSuiteSecret(suiteId, payload),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites', suiteId, 'secrets'] })
      },
    },
    queryClient
  )
}

export function useUpdateSuiteSecret(suiteId: string) {
  const queryClient = useSafeQueryClient()
  return useMutation(
    {
      mutationFn: ({
        secretId,
        payload,
      }: {
        secretId: string
        payload: UpdateSecretPayload
      }) => updateSuiteSecret(suiteId, secretId, payload),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites', suiteId, 'secrets'] })
      },
    },
    queryClient
  )
}

export function useDeleteSuiteSecret(suiteId: string) {
  const queryClient = useSafeQueryClient()
  return useMutation(
    {
      mutationFn: (secretId: string) => deleteSuiteSecret(suiteId, secretId),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['suites', suiteId, 'secrets'] })
      },
    },
    queryClient
  )
}
