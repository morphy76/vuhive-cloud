import React from 'react'
import { Database } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { useOnlineStatus } from '@/hooks/use-online-status'

export interface OfflinePreviewBadgeProps {
  className?: string
}

/**
 * OfflinePreviewBadge displays an "Offline Preview" indicator
 * attached to cards or tables when operating in disconnected/cached mode.
 */
export const OfflinePreviewBadge: React.FC<OfflinePreviewBadgeProps> = ({ className = '' }) => {
  const { isOnline } = useOnlineStatus()

  if (isOnline) {
    return null
  }

  return (
    <Badge
      variant="warning"
      className={`text-[11px] font-medium tracking-wide uppercase px-2 py-0.5 gap-1.5 ${className}`}
    >
      <Database className="w-3 h-3" aria-hidden="true" />
      <span>Offline Preview</span>
    </Badge>
  )
}
