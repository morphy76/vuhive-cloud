import React, { useState, useEffect } from 'react'
import { WifiOff, AlertTriangle } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/use-online-status'

export interface OfflineBannerProps {
  className?: string
}

/**
 * OfflineBanner displays a non-intrusive notification banner
 * when browser network connectivity is lost or upstream control plane circuit breaker is open.
 */
export const OfflineBanner: React.FC<OfflineBannerProps> = ({ className = '' }) => {
  const { isOnline } = useOnlineStatus()
  const [isCircuitOpen, setIsCircuitOpen] = useState<boolean>(false)

  useEffect(() => {
    if (typeof window === 'undefined') return

    const handleCircuitBreaker = (event: Event) => {
      const customEvent = event as CustomEvent<{ open?: boolean }>
      if (customEvent.detail && typeof customEvent.detail.open === 'boolean') {
        setIsCircuitOpen(customEvent.detail.open)
      }
    }

    window.addEventListener('vuhive:circuit-breaker', handleCircuitBreaker)
    return () => {
      window.removeEventListener('vuhive:circuit-breaker', handleCircuitBreaker)
    }
  }, [])

  if (isOnline && !isCircuitOpen) {
    return null
  }

  if (!isOnline) {
    return (
      <div
        role="status"
        aria-live="polite"
        className={`bg-amber-500/10 border-b border-amber-500/20 px-4 py-2 text-amber-800 dark:text-amber-300 text-xs font-medium flex items-center justify-center gap-2 transition-all ${className}`}
      >
        <WifiOff className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400 shrink-0" aria-hidden="true" />
        <span>
          <strong>Offline Mode</strong> — Network connection unavailable. Displaying cached test data.
        </span>
      </div>
    )
  }

  return (
    <div
      role="status"
      aria-live="polite"
      className={`bg-rose-500/10 border-b border-rose-500/20 px-4 py-2 text-rose-800 dark:text-rose-300 text-xs font-medium flex items-center justify-center gap-2 transition-all ${className}`}
    >
      <AlertTriangle className="w-3.5 h-3.5 text-rose-600 dark:text-rose-400 shrink-0" aria-hidden="true" />
      <span>
        <strong>Degraded Mode</strong> — Upstream control plane circuit breaker is open. Displaying cached test data.
      </span>
    </div>
  )
}
