import * as React from 'react'
import { CalendarClock, CheckCircle2, AlertCircle } from 'lucide-react'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import {
  validateCron,
  describeCron,
  getNextCronRun,
  CRON_PRESETS,
} from '@/lib/cron-utils'

export interface CronBuilderProps {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  className?: string
}

export const CronBuilder: React.FC<CronBuilderProps> = ({
  value,
  onChange,
  disabled = false,
  className = '',
}) => {
  const validation = React.useMemo(() => validateCron(value), [value])
  const naturalDescription = React.useMemo(() => describeCron(value), [value])
  const nextRun = React.useMemo(() => getNextCronRun(value), [value])

  const formattedNextRun = React.useMemo(() => {
    if (!nextRun) return null
    return nextRun.toUTCString().replace('GMT', 'UTC')
  }, [nextRun])

  const isPresetSelected = (presetExpr: string) => value.trim() === presetExpr.trim()

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Input Field */}
      <div className="space-y-1.5">
        <div className="flex items-center gap-1.5">
          <label
            htmlFor="cron-expression"
            className="text-xs font-semibold text-slate-700 dark:text-slate-300"
          >
            Cron Expression
          </label>
          <HelpTooltip
            text="Standard 5-field CRON expression (minute, hour, day-of-month, month, day-of-week) executed in UTC timezone by native Kubernetes CronJobs."
            label="Help for cron expression"
          />
        </div>
        <input
          id="cron-expression"
          type="text"
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
          placeholder="e.g. 0 2 * * *"
          aria-label="Cron Expression"
          className={`w-full rounded-xl border px-3.5 py-2 text-sm font-mono transition-colors focus-visible:outline-none focus-visible:ring-2 ${
            validation.isValid
              ? 'border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 focus-visible:ring-brand-500'
              : 'border-red-300 dark:border-red-800 bg-red-50/30 dark:bg-red-950/20 text-red-900 dark:text-red-200 focus-visible:ring-red-500'
          }`}
        />
        {!validation.isValid && validation.error && (
          <div className="flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400 mt-1">
            <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
            <span>{validation.error}</span>
          </div>
        )}
      </div>

      {/* Quick Presets */}
      <div className="space-y-1.5">
        <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
          Quick Presets
        </span>
        <div className="flex gap-2 flex-wrap">
          {CRON_PRESETS.map((p) => {
            const isSelected = isPresetSelected(p.expr)
            return (
              <button
                key={p.label}
                type="button"
                aria-label={`Preset ${p.label}`}
                disabled={disabled}
                onClick={() => onChange(p.expr)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors border ${
                  isSelected
                    ? 'bg-brand-50 border-brand-300 text-brand-700 dark:bg-brand-950/60 dark:border-brand-800 dark:text-brand-300 font-semibold'
                    : 'bg-slate-50 border-slate-200 text-slate-600 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
                }`}
              >
                {p.label} ({p.expr})
              </button>
            )
          })}
        </div>
      </div>

      {/* Natural Language Preview & Next Trigger Banner */}
      <div
        className={`p-3.5 rounded-xl border transition-colors ${
          validation.isValid
            ? 'bg-slate-50 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700/60'
            : 'bg-red-50/50 dark:bg-red-950/20 border-red-200 dark:border-red-900/50'
        }`}
      >
        <div className="flex items-start gap-3">
          {validation.isValid ? (
            <CalendarClock className="w-4 h-4 text-brand-600 dark:text-brand-400 mt-0.5 flex-shrink-0" />
          ) : (
            <AlertCircle className="w-4 h-4 text-red-500 dark:text-red-400 mt-0.5 flex-shrink-0" />
          )}

          <div className="space-y-1 text-xs">
            <div className="font-medium text-slate-900 dark:text-slate-100 flex items-center gap-1.5">
              <span>{naturalDescription}</span>
              {validation.isValid && (
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 inline flex-shrink-0" />
              )}
            </div>

            {validation.isValid && formattedNextRun && (
              <div className="text-[11px] text-slate-500 dark:text-slate-400">
                <span className="font-medium">Next scheduled run:</span>{' '}
                <span className="font-mono text-slate-700 dark:text-slate-300">
                  {formattedNextRun}
                </span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
