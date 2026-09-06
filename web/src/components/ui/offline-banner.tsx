import React from 'react'
import { WifiOff } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/use-online-status'

export interface OfflineBannerProps {
  className?: string
}

/**
 * OfflineBanner displays a non-intrusive notification banner
 * when browser network connectivity is lost.
 */
export const OfflineBanner: React.FC<OfflineBannerProps> = ({ className = '' }) => {
  const { isOnline } = useOnlineStatus()

  if (isOnline) {
    return null
  }

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
