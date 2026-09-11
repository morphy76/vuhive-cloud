import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'

export interface BuildStatusChangedEvent {
  artifact_id: string
  suite_id: string
  platform: string
  status: 'PENDING' | 'BUILDING' | 'READY' | 'FAILED'
  previous_status?: string
  s3_binary_key?: string
  sha256_checksum?: string
  error_message?: string
  timestamp: string
}

export interface RunStatusChangedEvent {
  run_id: string
  suite_id: string
  status: 'QUEUED' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'ABORTED'
  previous_status?: string
  started_at?: string
  finished_at?: string
  duration_ms?: number
  exit_code?: number
  sla_passed?: boolean
  k8s_job_name?: string
  metrics?: {
    total_iterations?: number
    total_requests?: number
    avg_tps?: number
    p50_duration_ms?: number
    p90_duration_ms?: number
    p95_duration_ms?: number
    p99_duration_ms?: number
    error_rate_pct?: number
  }
  timestamp: string
}

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

const SSE_PREFIXES = ['/api/bff/v1/events', '/api/v1/bff/events']

/**
 * Hook for subscribing to live build status updates via SSE.
 */
export function useBuildEvents(
  suiteId?: string,
  onBuildStatusChange?: (event: BuildStatusChangedEvent) => void
) {
  const queryClient = useSafeQueryClient()
  const onBuildStatusChangeRef = useRef(onBuildStatusChange)
  onBuildStatusChangeRef.current = onBuildStatusChange

  useEffect(() => {
    if (typeof window === 'undefined' || typeof EventSource === 'undefined') {
      return
    }

    let es: EventSource | null = null
    let activeIndex = 0

    const connect = () => {
      const targetUrl = SSE_PREFIXES[activeIndex]
      try {
        es = new EventSource(targetUrl)

        es.addEventListener('build_status_changed', (e: MessageEvent) => {
          try {
            const data: BuildStatusChangedEvent = JSON.parse(e.data)
            if (!suiteId || data.suite_id === suiteId) {
              if (onBuildStatusChangeRef.current) {
                onBuildStatusChangeRef.current(data)
              }
              // Invalidate related queries so UI updates reactively
              queryClient.invalidateQueries({
                queryKey: ['suites', data.suite_id, 'artifacts'],
              })
              queryClient.invalidateQueries({
                queryKey: ['suites'],
              })
            }
          } catch (err) {
            console.warn('Failed parsing build_status_changed SSE payload:', err)
          }
        })

        es.onerror = () => {
          if (es) {
            es.close()
            es = null
          }
          // Try alternate endpoint if primary fails
          activeIndex = (activeIndex + 1) % SSE_PREFIXES.length
        }
      } catch (err) {
        console.warn('Unable to connect to SSE events stream:', err)
      }
    }

    connect()

    return () => {
      if (es) {
        es.close()
        es = null
      }
    }
  }, [suiteId, queryClient])
}

/**
 * Hook for subscribing to live test run status updates via SSE.
 */
export function useRunEvents(
  runId?: string,
  onRunStatusChange?: (event: RunStatusChangedEvent) => void
) {
  const queryClient = useSafeQueryClient()
  const onRunStatusChangeRef = useRef(onRunStatusChange)
  onRunStatusChangeRef.current = onRunStatusChange

  useEffect(() => {
    if (typeof window === 'undefined' || typeof EventSource === 'undefined') {
      return
    }

    let es: EventSource | null = null
    let activeIndex = 0

    const connect = () => {
      const targetUrl = SSE_PREFIXES[activeIndex]
      try {
        es = new EventSource(targetUrl)

        es.addEventListener('run_status_changed', (e: MessageEvent) => {
          try {
            const data: RunStatusChangedEvent = JSON.parse(e.data)
            if (!runId || data.run_id === runId) {
              if (onRunStatusChangeRef.current) {
                onRunStatusChangeRef.current(data)
              }
              // Invalidate related queries so UI updates reactively
              queryClient.invalidateQueries({
                queryKey: ['runs'],
              })
              queryClient.invalidateQueries({
                queryKey: ['runs', data.run_id],
              })
              if (data.suite_id) {
                queryClient.invalidateQueries({
                  queryKey: ['suites', data.suite_id, 'runs'],
                })
              }
            }
          } catch (err) {
            console.warn('Failed parsing run_status_changed SSE payload:', err)
          }
        })

        es.onerror = () => {
          if (es) {
            es.close()
            es = null
          }
          activeIndex = (activeIndex + 1) % SSE_PREFIXES.length
        }
      } catch (err) {
        console.warn('Unable to connect to SSE events stream:', err)
      }
    }

    connect()

    return () => {
      if (es) {
        es.close()
        es = null
      }
    }
  }, [runId, queryClient])
}
