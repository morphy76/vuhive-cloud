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
import type { CompiledArtifact } from '@/types/suite'

export interface DeleteArtifactDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  artifact: CompiledArtifact | null
  onConfirm: () => Promise<void> | void
  isDeleting?: boolean
}

export const DeleteArtifactDialog: React.FC<DeleteArtifactDialogProps> = ({
  open,
  onOpenChange,
  artifact,
  onConfirm,
  isDeleting = false,
}) => {
  if (!artifact) return null

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
          <div className="flex items-center gap-2.5 text-rose-600 dark:text-rose-400 mb-1">
            <div className="p-2 rounded-xl bg-rose-50 dark:bg-rose-950/50 border border-rose-200 dark:border-rose-900">
              <Trash2 className="w-5 h-5" aria-hidden="true" />
            </div>
            <DialogTitle className="text-lg font-bold text-slate-900 dark:text-white">
              Delete Artifact
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
            Confirm permanent deletion of the compiled binary artifact and associated records.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs">
          {/* Irreversible Destruction Warning */}
          <div className="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-800/60 text-rose-900 dark:text-rose-200 leading-relaxed">
            <p className="font-semibold mb-1">Warning: Irreversible Action</p>
            <p className="text-[11px] opacity-90">
              Deleting artifact <span className="font-mono font-bold">{artifact.id}</span> cannot be undone. The artifact record and any compiled binary stored in S3 object storage will be permanently purged.
            </p>
          </div>

          {/* Artifact Details */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 space-y-1.5">
            <div className="flex justify-between items-center">
              <span className="text-slate-500 dark:text-slate-400">Artifact ID:</span>
              <span className="font-mono font-medium text-slate-800 dark:text-slate-200">{artifact.id}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-slate-500 dark:text-slate-400">Platform:</span>
              <span className="font-mono font-semibold text-brand-600 dark:text-brand-400">{artifact.platform}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-slate-500 dark:text-slate-400">Status:</span>
              <span className="font-medium text-slate-800 dark:text-slate-200">{artifact.status}</span>
            </div>
            {artifact.s3BinaryKey && (
              <div className="flex justify-between items-center">
                <span className="text-slate-500 dark:text-slate-400">S3 Key:</span>
                <span className="font-mono text-[10px] text-slate-600 dark:text-slate-300 truncate max-w-[200px]">{artifact.s3BinaryKey}</span>
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isDeleting}
            className="min-h-[44px]"
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleConfirm}
            disabled={isDeleting}
            className="min-h-[44px] gap-2 bg-rose-600 hover:bg-rose-700 text-white focus:ring-rose-500"
          >
            {isDeleting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Deleting Artifact...</span>
              </>
            ) : (
              <>
                <Trash2 className="w-4 h-4" />
                <span>Delete Artifact</span>
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
