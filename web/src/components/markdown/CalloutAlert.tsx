import React from 'react'
import { Info, Lightbulb, AlertCircle, AlertTriangle, ShieldAlert } from 'lucide-react'
import { cn } from '@/lib/utils'

export type AlertType = 'note' | 'tip' | 'important' | 'warning' | 'caution'

interface CalloutConfig {
  label: string
  icon: React.ElementType
  containerClass: string
  iconClass: string
  titleClass: string
}

const ALERT_CONFIGS: Record<AlertType, CalloutConfig> = {
  note: {
    label: 'NOTE',
    icon: Info,
    containerClass: 'bg-blue-50/80 dark:bg-blue-950/30 border-blue-300 dark:border-blue-800 text-blue-950 dark:text-blue-100',
    iconClass: 'text-blue-600 dark:text-blue-400',
    titleClass: 'text-blue-800 dark:text-blue-300',
  },
  tip: {
    label: 'TIP',
    icon: Lightbulb,
    containerClass: 'bg-emerald-50/80 dark:bg-emerald-950/30 border-emerald-300 dark:border-emerald-800 text-emerald-950 dark:text-emerald-100',
    iconClass: 'text-emerald-600 dark:text-emerald-400',
    titleClass: 'text-emerald-800 dark:text-emerald-300',
  },
  important: {
    label: 'IMPORTANT',
    icon: AlertCircle,
    containerClass: 'bg-indigo-50/80 dark:bg-indigo-950/30 border-indigo-300 dark:border-indigo-800 text-indigo-950 dark:text-indigo-100',
    iconClass: 'text-indigo-600 dark:text-indigo-400',
    titleClass: 'text-indigo-800 dark:text-indigo-300',
  },
  warning: {
    label: 'WARNING',
    icon: AlertTriangle,
    containerClass: 'bg-amber-50/80 dark:bg-amber-950/30 border-amber-300 dark:border-amber-800 text-amber-950 dark:text-amber-100',
    iconClass: 'text-amber-600 dark:text-amber-400',
    titleClass: 'text-amber-800 dark:text-amber-300',
  },
  caution: {
    label: 'CAUTION',
    icon: ShieldAlert,
    containerClass: 'bg-rose-50/80 dark:bg-rose-950/30 border-rose-300 dark:border-rose-800 text-rose-950 dark:text-rose-100',
    iconClass: 'text-rose-600 dark:text-rose-400',
    titleClass: 'text-rose-800 dark:text-rose-300',
  },
}

interface CalloutAlertProps {
  children: React.ReactNode
}

/**
 * Extracts non-whitespace text from React children to check for GitHub callout patterns.
 */
function extractFirstText(node: React.ReactNode): string {
  if (typeof node === 'string') {
    const trimmed = node.trim()
    return trimmed.length > 0 ? trimmed : ''
  }
  if (Array.isArray(node)) {
    for (const child of node) {
      const text = extractFirstText(child)
      if (text) return text
    }
  }
  if (React.isValidElement(node) && (node.props as any)?.children) {
    return extractFirstText((node.props as any).children)
  }
  return ''
}

/**
 * Recursively strips the leading `[!TYPE]` token from the first text node encountered.
 */
function stripAlertHeader(children: React.ReactNode, alertType: string): React.ReactNode {
  const pattern = new RegExp(`^\\s*\\[!${alertType}\\]\\s*`, 'i')
  let stripped = false

  function cleanNode(node: React.ReactNode): React.ReactNode {
    if (stripped) return node

    if (typeof node === 'string') {
      if (pattern.test(node)) {
        stripped = true
        return node.replace(pattern, '')
      }
      return node
    }

    if (Array.isArray(node)) {
      return node.map(cleanNode)
    }

    if (React.isValidElement(node)) {
      const childProps = node.props as any
      if (childProps?.children) {
        return React.cloneElement(node, {
          ...childProps,
          children: cleanNode(childProps.children),
        })
      }
    }

    return node
  }

  return cleanNode(children)
}

export const CalloutAlert: React.FC<CalloutAlertProps> = ({ children }) => {
  const firstText = extractFirstText(children)
  const match = firstText.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]/i)

  if (!match) {
    return (
      <blockquote className="my-4 border-l-4 border-slate-300 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/30 pl-4 py-2 rounded-r-xl text-slate-700 dark:text-slate-300 italic">
        {children}
      </blockquote>
    )
  }

  const alertType = match[1].toLowerCase() as AlertType
  const config = ALERT_CONFIGS[alertType] || ALERT_CONFIGS.note
  const Icon = config.icon
  const cleanChildren = stripAlertHeader(children, match[1])

  return (
    <div
      role="note"
      aria-label={`${config.label} callout`}
      className={cn(
        'my-4 p-4 rounded-xl border-l-4 border shadow-xs transition-colors',
        config.containerClass
      )}
    >
      <div className="flex items-center gap-2 font-semibold text-xs tracking-wide uppercase mb-1.5">
        <Icon className={cn('w-4 h-4 flex-shrink-0', config.iconClass)} />
        <span className={config.titleClass}>{config.label}</span>
      </div>
      <div className="text-sm leading-relaxed space-y-2 [&>p]:my-1">
        {cleanChildren}
      </div>
    </div>
  )
}
