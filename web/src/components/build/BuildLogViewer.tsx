import React, { useState } from 'react'
import { ChevronDown, ChevronRight, Copy, Check, Terminal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface BuildLogViewerProps {
  logs?: string
  title?: string
  defaultExpanded?: boolean
  className?: string
}

export const BuildLogViewer: React.FC<BuildLogViewerProps> = ({
  logs,
  title = 'Compilation Logs',
  defaultExpanded = true,
  className,
}) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded)
  const [copied, setCopied] = useState(false)

  if (!logs) return null

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(logs)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // ignore clipboard error
    }
  }

  const logLines = logs.split('\n')

  return (
    <div
      className={cn(
        'rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-900 text-slate-100 overflow-hidden shadow-inner',
        className
      )}
    >
      {/* Header bar */}
      <div className="w-full flex items-center justify-between px-3.5 py-2 bg-slate-800/80 border-b border-slate-800">
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-2 flex-1 text-left cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-brand-500 rounded py-1"
          aria-expanded={isExpanded}
          aria-controls="build-logs-content"
        >
          {isExpanded ? (
            <ChevronDown className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronRight className="w-4 h-4 text-slate-400" />
          )}
          <Terminal className="w-4 h-4 text-brand-400" />
          <span className="text-xs font-semibold text-slate-200">{title}</span>
          <span className="text-[11px] font-mono text-slate-400">
            ({logLines.length} {logLines.length === 1 ? 'line' : 'lines'})
          </span>
        </button>

        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          className="h-7 px-2 text-xs text-slate-300 hover:text-white hover:bg-slate-700/60 gap-1.5"
          aria-label="Copy logs to clipboard"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              <span className="text-[11px] text-emerald-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span className="text-[11px]">Copy</span>
            </>
          )}
        </Button>
      </div>

      {/* Log output viewport */}
      {isExpanded && (
        <div
          id="build-logs-content"
          className="p-3 max-h-64 overflow-y-auto font-mono text-xs leading-relaxed divide-y divide-slate-800/40 select-text"
          tabIndex={0}
          aria-label="Build compilation terminal output"
        >
          {logLines.map((line, idx) => (
            <div key={idx} className="flex items-start gap-3 py-0.5 font-mono">
              <span className="select-none text-slate-600 dark:text-slate-500 text-[10px] w-6 text-right flex-shrink-0">
                {idx + 1}
              </span>
              <span
                className={cn(
                  'whitespace-pre-wrap break-all',
                  line.toLowerCase().includes('error') || line.toLowerCase().includes('fail')
                    ? 'text-red-400 font-medium'
                    : line.toLowerCase().includes('warn')
                    ? 'text-amber-400'
                    : 'text-slate-300'
                )}
              >
                {line || ' '}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
