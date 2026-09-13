import * as React from 'react'
import { Loader2, Sparkles } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import type { HistoricalRun } from '@/types/suite'

export interface CleanRuntimeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  run?: HistoricalRun | null
  onConfirm: () => Promise<void> | void
  isCleaning?: boolean
}

export const CleanRuntimeDialog: React.FC<CleanRuntimeDialogProps> = ({
  open,
  onOpenChange,
  run,
  onConfirm,
  isCleaning = false,
}) => {
  if (!run) return null

  const handleConfirm = async () => {
    await onConfirm()
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <div className="flex items-center gap-2.5 text-amber-600 dark:text-amber-400 mb-1">
            <div className="p-2 rounded-xl bg-amber-50 dark:bg-amber-950/50 border border-amber-200 dark:border-amber-900">
              <Sparkles className="w-5 h-5" aria-hidden="true" />
            </div>
            <DialogTitle className="text-lg font-bold text-slate-900 dark:text-white">
              Clean Runtime Resources
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
            Purge lingering Kubernetes Job and Pod resources for this test run.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs">
          <div className="p-3 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800/60 text-amber-900 dark:text-amber-200 leading-relaxed">
            <p className="font-semibold mb-1">Cluster Runtime Cleanup</p>
            <p className="text-[11px] opacity-90">
              This action deletes the underlying Kubernetes <code className="font-mono font-bold">batch/v1 Job</code> and
              any associated Pods (including containers stuck in <code className="font-mono font-bold">Init:Error</code> or CrashLoop).
              {run.status === 'RUNNING' || run.status === 'QUEUED' ? (
                <span> Since this run is currently active, its status will be finalized as <span className="font-bold">ABORTED</span>.</span>
              ) : (
                <span> Existing reports and logs stored in object storage will remain intact.</span>
              )}
            </p>
          </div>

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 space-y-1">
            <div className="text-[11px] text-slate-500 dark:text-slate-400">Target Run:</div>
            <div className="font-mono text-xs font-semibold text-slate-800 dark:text-slate-200">
              {run.id}
            </div>
            {run.k8sJobName && (
              <div className="text-[11px] text-slate-500 dark:text-slate-400 font-mono">
                Job: {run.k8sJobName} {run.k8sNamespace ? `in ${run.k8sNamespace}` : ''}
              </div>
            )}
            <div className="text-[11px] text-slate-500 dark:text-slate-400">
              Status: <span className="font-semibold text-slate-700 dark:text-slate-300">{run.status}</span>
            </div>
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isCleaning}
            className="min-h-[40px]"
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="default"
            onClick={handleConfirm}
            disabled={isCleaning}
            className="min-h-[40px] bg-amber-600 hover:bg-amber-700 text-white gap-1.5"
          >
            {isCleaning ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
                <span>Cleaning Runtime...</span>
              </>
            ) : (
              <>
                <Sparkles className="w-4 h-4" aria-hidden="true" />
                <span>Clean Runtime</span>
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
