import * as React from 'react'
import { HelpCircle } from 'lucide-react'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

export interface HelpTooltipProps {
  text: React.ReactNode
  label?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
  className?: string
  iconClassName?: string
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
}

export const HelpTooltip: React.FC<HelpTooltipProps> = ({
  text,
  label = 'More information',
  side = 'top',
  align = 'center',
  className,
  iconClassName,
  open,
  defaultOpen,
  onOpenChange,
}) => {
  return (
    <Tooltip open={open} defaultOpen={defaultOpen} onOpenChange={onOpenChange}>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          className={cn(
            'inline-flex items-center justify-center rounded-full text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-1 transition-colors p-0.5 min-w-[20px] min-h-[20px]',
            className
          )}
        >
          <HelpCircle className={cn('w-3.5 h-3.5 flex-shrink-0', iconClassName)} />
        </button>
      </TooltipTrigger>
      <TooltipContent
        side={side}
        align={align}
        className="max-w-xs text-xs font-normal leading-relaxed text-slate-100 bg-slate-900 dark:bg-slate-800 dark:text-slate-100 border border-slate-700/50 shadow-lg p-2.5 z-50 rounded-lg"
      >
        {text}
      </TooltipContent>
    </Tooltip>
  )
}
