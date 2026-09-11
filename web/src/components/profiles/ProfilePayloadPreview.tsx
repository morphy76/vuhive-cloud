import React, { useState } from 'react'
import { Check, Copy, Code2 } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface ProfilePayloadPreviewProps {
  payload: Record<string, any>
}

export const ProfilePayloadPreview: React.FC<ProfilePayloadPreviewProps> = ({ payload }) => {
  const [copied, setCopied] = useState(false)

  const jsonString = JSON.stringify(payload, null, 2)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(jsonString)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback
    }
  }

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-900 text-slate-100 overflow-hidden text-xs">
      <div className="flex items-center justify-between px-3.5 py-2 bg-slate-950 border-b border-slate-800">
        <div className="flex items-center gap-2">
          <Code2 className="w-3.5 h-3.5 text-brand-400" />
          <span className="font-mono text-slate-400">Kubernetes Profile Payload (JSON)</span>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          aria-label="Copy Kubernetes JSON payload"
          className="h-7 px-2 text-slate-300 hover:text-white hover:bg-slate-800 gap-1.5"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              <span className="text-emerald-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span>Copy JSON</span>
            </>
          )}
        </Button>
      </div>

      <pre className="p-3.5 font-mono text-[11px] overflow-x-auto max-h-60 leading-relaxed text-slate-300">
        {jsonString}
      </pre>
    </div>
  )
}
