import React from 'react'
import { Ban, CheckCircle2, Circle, Loader2, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

export type StepperStage = 'QUEUED' | 'BUILDING' | 'READY' | 'FAILED' | 'CANCELLED'

export interface BuildStatusStepperProps {
  currentStatus: string // PENDING, QUEUED, BUILDING, READY, FAILED, CANCELLED
  platform?: string
  artifactId?: string
  className?: string
}

interface StepItem {
  id: StepperStage
  label: string
  description: string
}

const STEPS: StepItem[] = [
  {
    id: 'QUEUED',
    label: 'Queued',
    description: 'Awaiting container slot',
  },
  {
    id: 'BUILDING',
    label: 'Compiling',
    description: 'Ephemeral K8s build Job',
  },
  {
    id: 'READY',
    label: 'Ready',
    description: 'Binary verified in S3',
  },
]

export const BuildStatusStepper: React.FC<BuildStatusStepperProps> = ({
  currentStatus,
  platform,
  artifactId,
  className,
}) => {
  const normalizedStatus =
    currentStatus.toUpperCase() === 'PENDING' ? 'QUEUED' : currentStatus.toUpperCase()

  const isFailed = normalizedStatus === 'FAILED'
  const isCancelled = normalizedStatus === 'CANCELLED'

  const getStepState = (stepIndex: number) => {
    if (isCancelled) {
      if (stepIndex === 0) return 'completed'
      if (stepIndex === 1) return 'cancelled'
      return 'pending'
    }

    if (isFailed) {
      if (stepIndex === 0) return 'completed'
      if (stepIndex === 1) return 'failed'
      return 'pending'
    }

    if (normalizedStatus === 'READY') {
      return 'completed'
    }

    if (normalizedStatus === 'BUILDING') {
      if (stepIndex === 0) return 'completed'
      if (stepIndex === 1) return 'active'
      return 'pending'
    }

    // Default: QUEUED
    if (stepIndex === 0) return 'active'
    return 'pending'
  }

  return (
    <div
      className={cn(
        'rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 p-4 space-y-4',
        className
      )}
      aria-label="Build compilation progress stepper"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
            Build Progress
          </span>
          {platform && (
            <span className="text-xs font-mono font-medium px-2 py-0.5 rounded bg-brand-50 text-brand-700 dark:bg-brand-950/80 dark:text-brand-300">
              {platform}
            </span>
          )}
        </div>
        {artifactId && (
          <span className="text-xs font-mono text-slate-400 truncate max-w-[140px]">
            {artifactId}
          </span>
        )}
      </div>

      <ol className="grid grid-cols-3 gap-2 relative">
        {STEPS.map((step, idx) => {
          const state = getStepState(idx)

          return (
            <li
              key={step.id}
              className="flex flex-col items-center text-center space-y-1.5 relative"
            >
              {/* Step indicator */}
              <div
                className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center transition-all duration-300 z-10',
                  state === 'completed' &&
                    'bg-emerald-100 text-emerald-600 dark:bg-emerald-950/80 dark:text-emerald-400',
                  state === 'active' &&
                    'bg-brand-100 text-brand-600 ring-2 ring-brand-500/30 dark:bg-brand-950 dark:text-brand-400',
                  state === 'failed' &&
                    'bg-red-100 text-red-600 ring-2 ring-red-500/30 dark:bg-red-950 dark:text-red-400',
                  state === 'cancelled' &&
                    'bg-amber-100 text-amber-600 ring-2 ring-amber-500/30 dark:bg-amber-950 dark:text-amber-400',
                  state === 'pending' &&
                    'bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-500'
                )}
              >
                {state === 'completed' && <CheckCircle2 className="w-4 h-4" />}
                {state === 'active' && <Loader2 className="w-4 h-4 animate-spin" />}
                {state === 'failed' && <XCircle className="w-4 h-4" />}
                {state === 'cancelled' && <Ban className="w-4 h-4" />}
                {state === 'pending' && <Circle className="w-3.5 h-3.5" />}
              </div>

              {/* Step label & desc */}
              <div className="space-y-0.5">
                <div
                  className={cn(
                    'text-xs font-semibold',
                    state === 'active' && 'text-brand-600 dark:text-brand-400',
                    state === 'failed' && 'text-red-600 dark:text-red-400',
                    state === 'cancelled' && 'text-amber-600 dark:text-amber-400',
                    state === 'completed' && 'text-slate-900 dark:text-white',
                    state === 'pending' && 'text-slate-400 dark:text-slate-500'
                  )}
                >
                  {isFailed && idx === 2 ? 'Failed' : isCancelled && idx === 2 ? 'Cancelled' : step.label}
                </div>
                <div className="text-[10px] text-slate-400 dark:text-slate-500 hidden sm:block">
                  {isFailed && idx === 2 ? 'Compilation error' : isCancelled && idx === 2 ? 'Build cancelled' : step.description}
                </div>
              </div>
            </li>
          )
        })}
      </ol>
    </div>
  )
}
