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
import { useCreateSuiteConfig } from '@/hooks/use-suites'

export interface AttachConfigDialogProps {
  suiteId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const AttachConfigDialog: React.FC<AttachConfigDialogProps> = ({
  suiteId,
  open,
  onOpenChange,
}) => {
  const [configName, setConfigName] = React.useState('')
  const [contentYaml, setContentYaml] = React.useState(
    'duration: 5m\nconcurrency: 50\nramp_up: 30s\n'
  )
  const [isDefault, setIsDefault] = React.useState(false)

  const createConfigMutation = useCreateSuiteConfig(suiteId)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!configName.trim() || !contentYaml.trim()) return

    try {
      await createConfigMutation.mutateAsync({
        name: configName.trim(),
        content_yaml: contentYaml,
        is_default: isDefault,
      })
      setConfigName('')
      onOpenChange(false)
    } catch (err) {
      console.error('Failed attaching configuration:', err)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg" onOpenAutoFocus={(e) => e.preventDefault()}>
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Attach Scenario Configuration</DialogTitle>
            <DialogDescription>
              Upload or declare scenario parameters, duration, ramp-up curves, and concurrency limits in YAML format.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="config-name"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Configuration Name
                </label>
                <HelpTooltip
                  text="Descriptive filename for this configuration (e.g. high-load.yaml)."
                  label="Help for config name"
                />
              </div>
              <input
                id="config-name"
                type="text"
                value={configName}
                onChange={(e) => setConfigName(e.target.value)}
                placeholder="e.g. production-stress.yaml"
                required
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="config-yaml"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Scenario Configuration YAML
                </label>
                <HelpTooltip
                  text="Runtime parameters consumed by the runner during test execution."
                  label="Help for config yaml"
                />
              </div>
              <textarea
                id="config-yaml"
                rows={6}
                value={contentYaml}
                onChange={(e) => setContentYaml(e.target.value)}
                required
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 font-mono text-xs text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>

            <div className="flex items-center justify-between rounded-xl border border-slate-200 dark:border-slate-800 p-3 bg-slate-50 dark:bg-slate-800/40">
              <div className="space-y-0.5">
                <span className="text-sm font-medium text-slate-900 dark:text-white">
                  Default Configuration
                </span>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  Automatically select this profile when dispatching ad-hoc runs.
                </p>
              </div>
              <Switch checked={isDefault} onCheckedChange={setIsDefault} />
            </div>
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
              disabled={!configName.trim() || !contentYaml.trim() || createConfigMutation.isPending}
              className="min-h-[44px]"
            >
              {createConfigMutation.isPending ? 'Attaching...' : 'Attach Configuration'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
