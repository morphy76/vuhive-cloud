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
import { Switch } from '@/components/ui/switch'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { InfoBadge } from '@/components/help/InfoBadge'
import { useUploadSuiteBuild } from '@/hooks/use-suites'

export interface UploadBuildDialogProps {
  suiteId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const UploadBuildDialog: React.FC<UploadBuildDialogProps> = ({
  suiteId,
  open,
  onOpenChange,
}) => {
  const [file, setFile] = React.useState<File | null>(null)
  const [platform, setPlatform] = React.useState<string>('linux/amd64')
  const [allowInsecure, setAllowInsecure] = React.useState(false)

  const uploadMutation = useUploadSuiteBuild(suiteId)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!file) return

    const formData = new FormData()
    formData.append('file', file)
    formData.append('platform', platform)
    if (allowInsecure) {
      formData.append('allow_insecure_imports', 'true')
    }

    try {
      await uploadMutation.mutateAsync(formData)
      setFile(null)
      onOpenChange(false)
    } catch (err) {
      console.error('Failed triggering build:', err)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg" onOpenAutoFocus={(e) => e.preventDefault()}>
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Upload Source & Trigger Build</DialogTitle>
            <DialogDescription>
              Upload a Go scenario archive (.tar.gz) for ephemeral compilation in a Kubernetes builder container.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="source-file"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Source Archive (.tar.gz)
                </label>
                <HelpTooltip
                  text="Must contain scenario.go implementing package scenario with func NewScenario()."
                  label="Help for source archive"
                />
              </div>
              <input
                id="source-file"
                type="file"
                accept=".tar.gz,.tgz,.tar"
                onChange={(e) => setFile(e.target.files?.[0] || null)}
                required
                className="w-full text-xs text-slate-500 file:mr-4 file:py-2 file:px-4 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-brand-50 file:text-brand-700 dark:file:bg-brand-950/60 dark:file:text-brand-300 hover:file:bg-brand-100 cursor-pointer"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Target Platform
                </span>
                <HelpTooltip
                  text="Select architecture or all platforms for cross-compilation."
                  label="Help for platform"
                />
              </div>
              <div className="flex gap-2">
                {[
                  { label: 'linux/amd64', val: 'linux/amd64' },
                  { label: 'linux/arm64', val: 'linux/arm64' },
                  { label: 'All Architectures', val: 'all' },
                ].map((p) => (
                  <button
                    key={p.val}
                    type="button"
                    onClick={() => setPlatform(p.val)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-mono font-medium transition-colors border ${
                      platform === p.val
                        ? 'bg-brand-50 border-brand-300 text-brand-700 dark:bg-brand-950/60 dark:border-brand-800 dark:text-brand-300'
                        : 'bg-slate-50 border-slate-200 text-slate-600 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400'
                    }`}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex items-center justify-between rounded-xl border border-slate-200 dark:border-slate-800 p-3 bg-slate-50 dark:bg-slate-800/40">
              <div className="space-y-0.5">
                <span className="text-sm font-medium text-slate-900 dark:text-white">
                  Allow Insecure Imports
                </span>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Override static AST safety filters for experimental or legacy packages.
                </p>
              </div>
              <Switch checked={allowInsecure} onCheckedChange={setAllowInsecure} />
            </div>

            <InfoBadge
              variant="warning"
              title="AST Static Analysis Enforced"
            >
              The compilation pipeline statically inspects AST imports. Unsafe packages (
              <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 font-mono text-[11px]">
                os/exec, syscall, unsafe, plugin
              </code>
              ) are automatically rejected.
            </InfoBadge>
          </div>

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="min-h-[44px]"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!file || uploadMutation.isPending}
              className="min-h-[44px]"
            >
              {uploadMutation.isPending ? 'Compiling...' : 'Upload & Build'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
