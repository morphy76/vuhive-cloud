import * as React from 'react'
import { AlertTriangle, Loader2, Trash2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import type { TestSuite } from '@/types/suite'

export interface DeleteSuiteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  suite: TestSuite | null
  onConfirm: () => Promise<void> | void
  isDeleting?: boolean
  hasActiveBuilds?: boolean
  hasActiveRuns?: boolean
}

export const DeleteSuiteDialog: React.FC<DeleteSuiteDialogProps> = ({
  open,
  onOpenChange,
  suite,
  onConfirm,
  isDeleting = false,
  hasActiveBuilds = false,
  hasActiveRuns = false,
}) => {
  const [confirmText, setConfirmText] = React.useState('')

  React.useEffect(() => {
    if (!open) {
      setConfirmText('')
    }
  }, [open])

  if (!suite) return null

  const isActive = suite.state === 'ACTIVE'
  const isBlocked = hasActiveBuilds || hasActiveRuns
  const isNameConfirmed = !isActive || confirmText.trim() === suite.name.trim()
  const canDelete = !isBlocked && isNameConfirmed && !isDeleting

  const handleConfirm = async () => {
    if (!canDelete) return
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
              Delete Test Suite
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
            Confirm permanent deletion of the test suite and its associated resources.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs">
          {/* Irreversible Destruction Warning */}
          <div className="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-800/60 text-rose-900 dark:text-rose-200 leading-relaxed">
            <p className="font-semibold mb-1">Warning: Irreversible Action</p>
            <p className="text-[11px] opacity-90">
              Deleting <span className="font-bold">{suite.name}</span> cannot be undone. All associated configurations, artifacts, schedules, and historical runs will be permanently deleted from the database.
            </p>
          </div>

          {/* S3 Object Retention Notice */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-300 leading-relaxed">
            <p className="font-semibold text-slate-800 dark:text-slate-200 mb-1">Object Storage Retention</p>
            <p className="text-[11px]">
              Uploaded scenario archives, compiled binaries, and execution reports stored in S3/MinIO are preserved according to bucket lifecycle or retention policies and are not immediately purged.
            </p>
          </div>

          {/* Suite Target Metadata */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 space-y-1">
            <div className="text-[11px] text-slate-500 dark:text-slate-400">Target Suite:</div>
            <div className="font-semibold text-slate-800 dark:text-slate-200">
              {suite.name}
            </div>
            <div className="font-mono text-[11px] text-slate-500 dark:text-slate-400">
              {suite.id}
            </div>
          </div>

          {/* Active Builds Guard Warning */}
          {hasActiveBuilds && (
            <div className="p-3 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800/60 text-amber-900 dark:text-amber-200 flex items-start gap-2.5">
              <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" aria-hidden="true" />
              <div className="text-[11px] leading-relaxed">
                <span className="font-semibold block">Cannot delete suite while builds are in progress.</span>
                Please wait for ephemeral compilation build jobs to complete before deleting this suite.
              </div>
            </div>
          )}

          {/* Active Runs Guard Warning */}
          {hasActiveRuns && (
            <div className="p-3 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800/60 text-amber-900 dark:text-amber-200 flex items-start gap-2.5">
              <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" aria-hidden="true" />
              <div className="text-[11px] leading-relaxed">
                <span className="font-semibold block">Cannot delete suite while test runs are actively executing.</span>
                Active test runs must finish or be aborted before deleting this suite.
              </div>
            </div>
          )}

          {/* Active State Accidental Deletion Guard (Type Name Confirmation) */}
          {isActive && !isBlocked && (
            <div className="space-y-1.5 pt-1">
              <label
                htmlFor="confirm-suite-name"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                To confirm, type <span className="font-bold text-slate-900 dark:text-white select-all">{suite.name}</span> in the box below:
              </label>
              <input
                id="confirm-suite-name"
                type="text"
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                placeholder={suite.name}
                autoComplete="off"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500"
              />
            </div>
          )}
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
            onClick={handleConfirm}
            disabled={!canDelete}
            className="min-h-[40px] bg-rose-600 hover:bg-rose-700 text-white font-semibold focus-visible:ring-rose-500"
          >
            {isDeleting ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                <span>Deleting...</span>
              </>
            ) : (
              <span>Delete Test Suite</span>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
