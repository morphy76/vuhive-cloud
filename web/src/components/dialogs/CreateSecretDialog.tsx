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
import { useCreateSuiteSecret } from '@/hooks/use-secrets'
import { useToast } from '@/hooks/use-toast'

export interface CreateSecretDialogProps {
  suiteId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

const SECRET_KEY_REGEX = /^[A-Z][A-Z0-9_]*$/

export const CreateSecretDialog: React.FC<CreateSecretDialogProps> = ({
  suiteId,
  open,
  onOpenChange,
}) => {
  const [key, setKey] = React.useState('')
  const [value, setValue] = React.useState('')
  const [description, setDescription] = React.useState('')
  const [showValue, setShowValue] = React.useState(false)

  const { toast } = useToast()
  const createMutation = useCreateSuiteSecret(suiteId)

  React.useEffect(() => {
    if (open) {
      setKey('')
      setValue('')
      setDescription('')
      setShowValue(false)
    }
  }, [open])

  const isKeyTouched = key.length > 0
  const isKeyValid = SECRET_KEY_REGEX.test(key)
  const isFormValid = isKeyValid && value.trim().length > 0

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isFormValid || createMutation.isPending) return

    try {
      await createMutation.mutateAsync({
        key: key.trim(),
        value: value.trim(),
      })
      toast({
        title: 'Suite Secret Created',
        description: `Secret "${key.trim()}" has been encrypted and stored securely.`,
      })
      onOpenChange(false)
    } catch (err: any) {
      toast({
        title: 'Failed to create secret',
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
                Add Suite Secret
              </DialogTitle>
            </div>
            <DialogDescription className="text-xs text-slate-500 dark:text-slate-400">
              Create an encrypted secret scoped to this test suite. You can reference this secret in scenario configurations via <span className="font-mono text-slate-700 dark:text-slate-300 font-medium">${'{secrets.KEY}'}</span>.
            </DialogDescription>
          </DialogHeader>

          {/* Security Disclaimer */}
          <div className="flex items-start gap-2.5 p-3 rounded-xl bg-emerald-50/80 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-900/60 text-emerald-900 dark:text-emerald-200 text-xs">
            <ShieldCheck className="w-4 h-4 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
            <div className="leading-relaxed text-[11px]">
              <span className="font-semibold">Security Note:</span> Values are encrypted at rest using AES-256-GCM by the control plane. Plaintext values are never returned over the API and will never be exposed in plaintext logs.
            </div>
          </div>

          <div className="space-y-3">
            {/* Key Field */}
            <div className="space-y-1">
              <label
                htmlFor="secret-key"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Secret Key
              </label>
              <input
                id="secret-key"
                type="text"
                value={key}
                onChange={(e) => setKey(e.target.value)}
                placeholder="e.g. API_KEY, AUTH_TOKEN"
                required
                className="w-full font-mono uppercase rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
              {isKeyTouched && !isKeyValid && (
                <p className="text-[11px] text-rose-500 dark:text-rose-400 mt-1">
                  Key must start with an uppercase letter and contain only uppercase letters, numbers, and underscores (e.g. API_KEY).
                </p>
              )}
            </div>

            {/* Value Field with Visibility Toggle */}
            <div className="space-y-1">
              <label
                htmlFor="secret-value"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Secret Value
              </label>
              <div className="relative">
                <input
                  id="secret-value"
                  type={showValue ? 'text' : 'password'}
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder="Enter secret value..."
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

            {/* Optional Description Field */}
            <div className="space-y-1">
              <label
                htmlFor="secret-description"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Description <span className="text-slate-400 font-normal">(Optional)</span>
              </label>
              <input
                id="secret-description"
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="e.g. Bearer token for authentication service"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>
          </div>

          <DialogFooter className="gap-2 sm:gap-0 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={createMutation.isPending}
              className="min-h-[44px]"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!isFormValid || createMutation.isPending}
              className="min-h-[44px] gap-1.5"
            >
              {createMutation.isPending && (
                <Loader2 className="w-4 h-4 animate-spin" />
              )}
              <span>{createMutation.isPending ? 'Saving...' : 'Add Secret'}</span>
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
