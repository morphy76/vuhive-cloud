import * as React from 'react'
import { Info, AlertTriangle, AlertCircle, CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'

export type InfoBadgeVariant = 'info' | 'warning' | 'error' | 'success' | 'default'

export interface InfoBadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: InfoBadgeVariant
  title: string
  children?: React.ReactNode
}

const variantStyles: Record<
  InfoBadgeVariant,
  { container: string; title: string; icon: React.FC<{ className?: string }> }
> = {
  info: {
    container: 'bg-brand-50/80 border-brand-200 text-brand-900 dark:bg-brand-950/40 dark:border-brand-800 dark:text-brand-200',
    title: 'text-brand-900 dark:text-brand-100',
    icon: Info,
  },
  warning: {
    container: 'bg-amber-50/80 border-amber-200 text-amber-900 dark:bg-amber-950/40 dark:border-amber-800 dark:text-amber-200',
    title: 'text-amber-900 dark:text-amber-100',
    icon: AlertTriangle,
  },
  error: {
    container: 'bg-red-50/80 border-red-200 text-red-900 dark:bg-red-950/40 dark:border-red-800 dark:text-red-200',
    title: 'text-red-900 dark:text-red-100',
    icon: AlertCircle,
  },
  success: {
    container: 'bg-emerald-50/80 border-emerald-200 text-emerald-900 dark:bg-emerald-950/40 dark:border-emerald-800 dark:text-emerald-200',
    title: 'text-emerald-900 dark:text-emerald-100',
    icon: CheckCircle2,
  },
  default: {
    container: 'bg-slate-50 border-slate-200 text-slate-800 dark:bg-slate-800/60 dark:border-slate-700 dark:text-slate-200',
    title: 'text-slate-900 dark:text-slate-100',
    icon: Info,
  },
}

export const InfoBadge: React.FC<InfoBadgeProps> = ({
  variant = 'info',
  title,
  children,
  className,
  ...props
}) => {
  const config = variantStyles[variant] || variantStyles.default
  const Icon = config.icon

  return (
    <div
      role="note"
      className={cn(
        'rounded-xl border p-3.5 text-xs transition-colors flex gap-3',
        config.container,
        className
      )}
      {...props}
    >
      <Icon className="w-4 h-4 flex-shrink-0 mt-0.5" />
      <div className="space-y-1">
        <div className={cn('font-semibold text-xs', config.title)}>{title}</div>
        {children && <div className="text-xs leading-relaxed opacity-90">{children}</div>}
      </div>
    </div>
  )
}
