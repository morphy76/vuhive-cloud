import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type DashboardData } from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export function useDashboard(initialData?: DashboardData) {
  const client = useSafeQueryClient()
  return useQuery(
    {
      queryKey: ['dashboard'],
      queryFn: () => api.getDashboard(),
      initialData,
      refetchInterval: 10000,
    },
    client
  )
}
