import * as React from 'react'
import { AlertTriangle, Loader2 } from 'lucide-react'
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

export interface AbortConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  run?: HistoricalRun | null
  onConfirm: (reason: string) => Promise<void> | void
  isAborting?: boolean
}

export const AbortConfirmationDialog: React.FC<AbortConfirmationDialogProps> = ({
  open,
  onOpenChange,
  run,
  onConfirm,
  isAborting = false,
}) => {
  const [reason, setReason] = React.useState('')

  const handleConfirm = async () => {
    await onConfirm(reason)
    setReason('')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <div className="flex items-center gap-2.5 text-rose-600 dark:text-rose-400 mb-1">
            <div className="p-2 rounded-xl bg-rose-50 dark:bg-rose-950/50 border border-rose-200 dark:border-rose-900">
              <AlertTriangle className="w-5 h-5" aria-hidden="true" />
            </div>
            <DialogTitle className="text-lg font-bold text-slate-900 dark:text-white">
              Abort Test Execution
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
            Confirm graceful pod teardown for active load test execution.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs">
          <div className="p-3 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800/60 text-amber-900 dark:text-amber-200 leading-relaxed">
            <p className="font-semibold mb-1">Warning: Graceful Pod Teardown</p>
            <p className="text-[11px] opacity-90">
              Aborting this test run will send an immediate termination signal to Kubernetes, abort the rendezvous
              barrier, terminate the batch/v1 Job and its worker pods, and finalize the execution status as <span className="font-bold">ABORTED</span>.
              This action cannot be undone.
            </p>
          </div>

          {run && (
            <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 space-y-1">
              <div className="text-[11px] text-slate-500 dark:text-slate-400">Target Run:</div>
              <div className="font-mono text-xs font-semibold text-slate-800 dark:text-slate-200">
                {run.id} {run.k8sJobName && `(${run.k8sJobName})`}
              </div>
            </div>
          )}

          <div className="space-y-1.5">
            <label
              htmlFor="abort-reason"
              className="text-xs font-semibold text-slate-700 dark:text-slate-300"
            >
              Cancellation Reason (Optional)
            </label>
            <textarea
              id="abort-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. High latency spike exceeding SLA thresholds"
              rows={2}
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500 resize-none"
            />
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isAborting}
            className="min-h-[40px]"
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleConfirm}
            disabled={isAborting}
            className="min-h-[40px] bg-rose-600 hover:bg-rose-700 text-white font-semibold focus-visible:ring-rose-500"
          >
            {isAborting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            Confirm Abort
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
