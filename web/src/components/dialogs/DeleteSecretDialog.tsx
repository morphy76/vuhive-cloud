import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Trash2, AlertTriangle, Loader2 } from 'lucide-react'
import type { SuiteSecret } from '@/types/secret'

export interface DeleteSecretDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  secret: SuiteSecret | null
  onConfirm: () => Promise<void> | void
  isDeleting?: boolean
}

export const DeleteSecretDialog: React.FC<DeleteSecretDialogProps> = ({
  open,
  onOpenChange,
  secret,
  onConfirm,
  isDeleting = false,
}) => {
  if (!secret) return null

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
              Delete Suite Secret
            </DialogTitle>
          </div>
          <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
            Confirm permanent deletion of this suite-scoped secret.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 py-2 text-xs">
          {/* Breaking Scenario Warning */}
          <div className="p-3 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-800/60 text-amber-900 dark:text-amber-200 space-y-1">
            <div className="flex items-center gap-2 font-semibold">
              <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0" />
              <span>Execution Impact Warning</span>
            </div>
            <p className="text-[11px] leading-relaxed opacity-95">
              Any test scenarios referencing this secret via <span className="font-mono font-bold">${`{secrets.${secret.key}}`}</span> will fail execution during startup initialization if the secret is removed.
            </p>
          </div>

          {/* Secret Details */}
          <div className="p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800 space-y-1.5">
            <div className="flex justify-between items-center">
              <span className="text-slate-500 dark:text-slate-400">Secret Key:</span>
              <span className="font-mono font-bold text-slate-800 dark:text-slate-200">{secret.key}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-slate-500 dark:text-slate-400">Placeholder:</span>
              <span className="font-mono text-brand-600 dark:text-brand-400">${`{secrets.${secret.key}}`}</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-slate-500 dark:text-slate-400">Storage:</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">AES-256-GCM Encrypted</span>
            </div>
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
            variant="destructive"
            onClick={handleConfirm}
            disabled={isDeleting}
            className="min-h-[44px] gap-1.5"
          >
            {isDeleting && <Loader2 className="w-4 h-4 animate-spin" />}
            <span>{isDeleting ? 'Deleting...' : 'Delete Secret'}</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
