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
import { KeyRound, Eye, EyeOff, ShieldCheck, Loader2 } from 'lucide-react'
import { useUpdateSuiteSecret } from '@/hooks/use-secrets'
import { useToast } from '@/hooks/use-toast'
import type { SuiteSecret } from '@/types/secret'

export interface EditSecretDialogProps {
  suiteId: string
  secret: SuiteSecret | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const EditSecretDialog: React.FC<EditSecretDialogProps> = ({
  suiteId,
  secret,
  open,
  onOpenChange,
}) => {
  const [value, setValue] = React.useState('')
  const [showValue, setShowValue] = React.useState(false)

  const { toast } = useToast()
  const updateMutation = useUpdateSuiteSecret(suiteId)

  React.useEffect(() => {
    if (open) {
      setValue('')
      setShowValue(false)
    }
  }, [open, secret])

  if (!secret) return null

  const isFormValid = value.trim().length > 0

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isFormValid || updateMutation.isPending) return

    try {
      await updateMutation.mutateAsync({
        secretId: secret.id,
        payload: {
          value: value.trim(),
        },
      })
      toast({
        title: 'Secret Updated',
        description: `Secret "${secret.key}" has been updated with a new encrypted value.`,
      })
      onOpenChange(false)
    } catch (err: any) {
      toast({
        title: 'Failed to update secret',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-md"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <div className="flex items-center gap-2.5 text-brand-600 dark:text-brand-400 mb-1">
              <div className="p-2 rounded-xl bg-brand-50 dark:bg-brand-950/50 border border-brand-200 dark:border-brand-900">
                <KeyRound className="w-5 h-5" aria-hidden="true" />
              </div>
              <DialogTitle className="text-lg font-bold text-slate-900 dark:text-white">
                Edit Suite Secret
              </DialogTitle>
            </div>
            <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
              Update the secret value. The key name cannot be modified after creation.
            </DialogDescription>
          </DialogHeader>

          {/* Security Disclaimer */}
          <div className="flex items-start gap-2.5 p-3 rounded-xl bg-emerald-50/80 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-900/60 text-emerald-900 dark:text-emerald-200 text-xs">
            <ShieldCheck className="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
            <div className="leading-relaxed text-[11px]">
              <span className="font-semibold">Security Note:</span> New secret values are immediately encrypted at rest via AES-256-GCM. The existing value will be overwritten.
            </div>
          </div>

          <div className="space-y-3">
            {/* Read-Only Key */}
            <div className="space-y-1">
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Secret Key
              </span>
              <div className="px-3.5 py-2 rounded-xl bg-slate-100 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 font-mono text-xs font-semibold text-slate-900 dark:text-slate-100">
                {secret.key}
              </div>
            </div>

            {/* New Value Field with Visibility Toggle */}
            <div className="space-y-1">
              <label
                htmlFor="edit-secret-value"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                New Secret Value
              </label>
              <div className="relative">
                <input
                  id="edit-secret-value"
                  type={showValue ? 'text' : 'password'}
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder="Enter replacement secret value..."
                  required
                  className="w-full font-mono rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 pl-3.5 pr-10 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
                <button
                  type="button"
                  aria-label="Toggle secret visibility"
                  onClick={() => setShowValue(!showValue)}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-colors"
                >
                  {showValue ? (
                    <EyeOff className="w-4 h-4" />
                  ) : (
                    <Eye className="w-4 h-4" />
                  )}
                </button>
              </div>
            </div>
          </div>

          <DialogFooter className="gap-2 sm:gap-0 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={updateMutation.isPending}
              className="min-h-[44px]"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!isFormValid || updateMutation.isPending}
              className="min-h-[44px] gap-1.5"
            >
              {updateMutation.isPending && (
                <Loader2 className="w-4 h-4 animate-spin" />
              )}
              <span>{updateMutation.isPending ? 'Saving...' : 'Update Secret'}</span>
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
