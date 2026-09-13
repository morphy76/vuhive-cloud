import * as React from 'react'
import { Loader2, Trash2 } from 'lucide-react'
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

export interface DeleteRunDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  run?: HistoricalRun | null
  onConfirm: () => Promise<void> | void
  isDeleting?: boolean
}

export const DeleteRunDialog: React.FC<DeleteRunDialogProps> = ({
  open,
  onOpenChange,
  run,
  onConfirm,
  isDeleting = false,
}) => {
  if (!run) return null

  const isActive = run.status === 'RUNNING' || run.status === 'QUEUED'

  const handleConfirm = async () => {
    if (isActive) return
    await onConfirm()
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
              <Trash2 className="w-5 h-5" aria-hidden="true" />
            </div>
            <DialogTitle className="text-lg font-bold text-slate-900 dark:text-white">
              Delete Test Run
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
            Confirm permanent deletion of this test run and all stored assets.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs">
          {isActive ? (
            <div className="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-800/60 text-rose-900 dark:text-rose-200 leading-relaxed">
              <p className="font-semibold mb-1">Active Run Cannot Be Deleted</p>
              <p className="text-[11px] opacity-90">
                This run is currently in <span className="font-bold">{run.status}</span> state. You must abort or let the run complete before it can be deleted.
              </p>
            </div>
          ) : (
            <div className="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-800/60 text-rose-900 dark:text-rose-200 leading-relaxed">
              <p className="font-semibold mb-1">Warning: Permanent Deletion</p>
              <p className="text-[11px] opacity-90">
                Deleting this test run will remove its execution record from the database, purge all associated logs
                and summary reports from object storage, and delete any remaining cluster jobs. This action cannot be undone.
              </p>
            </div>
          )}

          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 space-y-1">
            <div className="text-[11px] text-slate-500 dark:text-slate-400">Target Run:</div>
            <div className="font-mono text-xs font-semibold text-slate-800 dark:text-slate-200">
              {run.id}
            </div>
            {run.suiteId && (
              <div className="text-[11px] text-slate-600 dark:text-slate-300">
                Suite ID: {run.suiteId}
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
            disabled={isDeleting}
            className="min-h-[40px]"
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleConfirm}
            disabled={isDeleting || isActive}
            className="min-h-[40px] bg-rose-600 hover:bg-rose-700 text-white gap-1.5"
          >
            {isDeleting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
                <span>Deleting Run...</span>
              </>
            ) : (
              <>
                <Trash2 className="w-4 h-4" aria-hidden="true" />
                <span>Delete Run</span>
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
