import React from 'react'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { parseCPUTomilli, formatMillitoCPU, parseMemoryToMiB, formatMiBToMemory } from '@/lib/k8s-resources'
import { cn } from '@/lib/utils'

interface ResourceSliderInputProps {
  label: string
  helpText: string
  type: 'cpu' | 'memory'
  value: string
  onChange: (value: string) => void
  min?: number
  max?: number
  step?: number
  idPrefix: string
}

export const ResourceSliderInput: React.FC<ResourceSliderInputProps> = ({
  label,
  helpText,
  type,
  value,
  onChange,
  min,
  max,
  step,
  idPrefix,
}) => {
  if (type === 'cpu') {
    // Current value in millicores
    const currentMilli = parseCPUTomilli(value)
    const sliderMin = min ?? 100 // 100m
    const sliderMax = max ?? 16000 // 16 cores
    const sliderStep = step ?? 100

    const quickPresets = [
      { label: '250m', value: '250m' },
      { label: '500m', value: '500m' },
      { label: '1 Core', value: '1000m' },
      { label: '2 Cores', value: '2000m' },
      { label: '4 Cores', value: '4000m' },
      { label: '8 Cores', value: '8000m' },
    ]

    const handleSliderChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const milli = Number(e.target.value)
      onChange(formatMillitoCPU(milli, false))
    }

    const handleTextChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange(e.target.value)
    }

    const coresDisplay = (currentMilli / 1000).toFixed(currentMilli % 1000 === 0 ? 0 : 2)

    return (
      <div className="space-y-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 p-3.5 transition-colors">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <label
              htmlFor={`${idPrefix}-input`}
              className="text-xs font-semibold text-slate-700 dark:text-slate-300"
            >
              {label}
            </label>
            <HelpTooltip text={helpText} label={`Help for ${label}`} />
          </div>
          <span className="text-xs font-mono font-medium text-brand-600 dark:text-brand-400">
            {coresDisplay} vCPU ({currentMilli}m)
          </span>
        </div>

        <div className="flex items-center gap-3">
          <input
            id={`${idPrefix}-slider`}
            type="range"
            min={sliderMin}
            max={sliderMax}
            step={sliderStep}
            value={currentMilli || sliderMin}
            onChange={handleSliderChange}
            aria-label={`${label} slider`}
            className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-lg appearance-none cursor-pointer accent-brand-600 dark:accent-brand-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
          />
          <input
            id={`${idPrefix}-input`}
            type="text"
            value={value}
            onChange={handleTextChange}
            placeholder="1000m"
            className="w-24 text-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 px-2 py-1 text-xs font-mono font-semibold text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
          />
        </div>

        <div className="flex flex-wrap gap-1.5 pt-1">
          {quickPresets.map((preset) => {
            const isSelected = parseCPUTomilli(preset.value) === currentMilli
            return (
              <button
                key={preset.label}
                type="button"
                onClick={() => onChange(preset.value)}
                className={cn(
                  'rounded-md px-2 py-0.5 text-[11px] font-medium transition-colors',
                  isSelected
                    ? 'bg-brand-600 text-white dark:bg-brand-500'
                    : 'bg-slate-200/70 hover:bg-slate-200 text-slate-700 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300'
                )}
              >
                {preset.label}
              </button>
            )
          })}
        </div>
      </div>
    )
  }

  // Memory input
  const currentMiB = parseMemoryToMiB(value)
  const sliderMin = min ?? 128 // 128Mi
  const sliderMax = max ?? 32768 // 32Gi
  const sliderStep = step ?? 128

  const quickPresets = [
    { label: '256Mi', value: '256Mi' },
    { label: '512Mi', value: '512Mi' },
    { label: '1Gi', value: '1Gi' },
    { label: '2Gi', value: '2Gi' },
    { label: '4Gi', value: '4Gi' },
    { label: '8Gi', value: '8Gi' },
    { label: '16Gi', value: '16Gi' },
  ]

  const handleSliderChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const mib = Number(e.target.value)
    onChange(formatMiBToMemory(mib))
  }

  const handleTextChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value)
  }

  const gibDisplay = (currentMiB / 1024).toFixed(currentMiB % 1024 === 0 ? 0 : 2)

  return (
    <div className="space-y-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 p-3.5 transition-colors">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <label
            htmlFor={`${idPrefix}-input`}
            className="text-xs font-semibold text-slate-700 dark:text-slate-300"
          >
            {label}
          </label>
          <HelpTooltip text={helpText} label={`Help for ${label}`} />
        </div>
        <span className="text-xs font-mono font-medium text-brand-600 dark:text-brand-400">
          {gibDisplay} GiB ({currentMiB}Mi)
        </span>
      </div>

      <div className="flex items-center gap-3">
        <input
          id={`${idPrefix}-slider`}
          type="range"
          min={sliderMin}
          max={sliderMax}
          step={sliderStep}
          value={currentMiB || sliderMin}
          onChange={handleSliderChange}
          aria-label={`${label} slider`}
          className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-lg appearance-none cursor-pointer accent-brand-600 dark:accent-brand-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
        />
        <input
          id={`${idPrefix}-input`}
          type="text"
          value={value}
          onChange={handleTextChange}
          placeholder="1Gi"
          className="w-24 text-center rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 px-2 py-1 text-xs font-mono font-semibold text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
        />
      </div>

      <div className="flex flex-wrap gap-1.5 pt-1">
        {quickPresets.map((preset) => {
          const isSelected = parseMemoryToMiB(preset.value) === currentMiB
          return (
            <button
              key={preset.label}
              type="button"
              onClick={() => onChange(preset.value)}
              className={cn(
                'rounded-md px-2 py-0.5 text-[11px] font-medium transition-colors',
                isSelected
                  ? 'bg-brand-600 text-white dark:bg-brand-500'
                  : 'bg-slate-200/70 hover:bg-slate-200 text-slate-700 dark:bg-slate-800 dark:hover:bg-slate-700 dark:text-slate-300'
              )}
            >
              {preset.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}
