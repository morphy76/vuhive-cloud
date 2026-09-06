import React from 'react'
import { Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'
import { useInstallPrompt } from '@/hooks/use-install-prompt'

export interface InstallButtonProps {
  className?: string
  showText?: boolean
}

/**
 * InstallButton displays an install action when the browser detects
 * that vuhive-cloud satisfies PWA installability requirements.
 */
export const InstallButton: React.FC<InstallButtonProps> = ({
  className = '',
  showText = true,
}) => {
  const { canInstall, promptInstall } = useInstallPrompt()

  if (!canInstall) {
    return null
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void promptInstall()}
            className={`gap-1.5 border-brand-500/30 text-brand-600 dark:text-brand-400 hover:bg-brand-50 dark:hover:bg-brand-950/40 min-h-[40px] px-3 ${className}`}
            aria-label="Install vuhive-cloud application"
          >
            <Download className="w-4 h-4 text-brand-500" aria-hidden="true" />
            {showText && <span className="font-medium text-xs">Install</span>}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          Install vuhive-cloud to your home screen or desktop for offline access
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
