import React, { useState } from 'react'
import { Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface CodeBlockProps {
  children: string
  language?: string
  className?: string
}

export const CodeBlock: React.FC<CodeBlockProps> = ({
  children,
  language,
  className,
}) => {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(children)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy code to clipboard', err)
    }
  }

  return (
    <div
      className={cn(
        'group relative my-4 rounded-xl border border-slate-800 bg-slate-950 shadow-sm overflow-hidden text-slate-100',
        className
      )}
    >
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-800/80 bg-slate-900/60 text-xs text-slate-400">
        <span className="font-mono lowercase">{language || 'text'}</span>
        <Button
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          aria-label="Copy code to clipboard"
          className="h-7 px-2.5 text-xs text-slate-400 hover:text-white hover:bg-slate-800 transition-colors gap-1.5 min-h-[28px]"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              <span className="text-emerald-400 font-medium">Copied!</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span>Copy</span>
            </>
          )}
        </Button>
      </div>

      <pre className="overflow-x-auto p-4 text-xs font-mono leading-relaxed">
        <code>{children}</code>
      </pre>

      <span className="sr-only" aria-live="polite">
        {copied ? 'Copied code to clipboard' : ''}
      </span>
    </div>
  )
}
