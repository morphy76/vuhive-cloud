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
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { InfoBadge } from '@/components/help/InfoBadge'

export interface CreateSuiteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const CreateSuiteDialog: React.FC<CreateSuiteDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const [suiteName, setSuiteName] = React.useState('')
  const [selectedArchs, setSelectedArchs] = React.useState<string[]>([
    'linux/amd64',
    'linux/arm64',
  ])

  const toggleArch = (arch: string) => {
    setSelectedArchs((prev) =>
      prev.includes(arch) ? prev.filter((a) => a !== arch) : [...prev, arch]
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-lg"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>New Test Suite</DialogTitle>
          <DialogDescription>
            Configure and package load-testing scenarios for ephemeral cross-compilation.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="suite-name"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Suite Name
              </label>
              <HelpTooltip
                text="A unique identifier for your load test scenario (e.g. checkout-stress-test)."
                label="Help for suite name"
              />
            </div>
            <input
              id="suite-name"
              type="text"
              value={suiteName}
              onChange={(e) => setSuiteName(e.target.value)}
              placeholder="e.g. checkout-payment-stress"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            />
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                Target Architecture
              </span>
              <HelpTooltip
                text="Target CPU architectures for ephemeral cross-compilation in golang:1.26-alpine build containers."
                label="Help for target architecture"
              />
            </div>
            <div className="flex gap-2">
              {['linux/amd64', 'linux/arm64'].map((arch) => (
                <button
                  key={arch}
                  type="button"
                  onClick={() => toggleArch(arch)}
                  className={`px-3 py-1.5 rounded-lg text-xs font-mono font-medium transition-colors border ${
                    selectedArchs.includes(arch)
                      ? 'bg-brand-50 border-brand-300 text-brand-700 dark:bg-brand-950/60 dark:border-brand-800 dark:text-brand-300'
                      : 'bg-slate-50 border-slate-200 text-slate-600 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400'
                  }`}
                >
                  {arch}
                </button>
              ))}
            </div>
          </div>

          <InfoBadge
            variant="warning"
            title="AST Static Analysis & Framework Enforcement"
          >
            Scenario source packages must declare{' '}
            <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 font-mono text-[11px]">
              package scenario
            </code>{' '}
            and depend directly on{' '}
            <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 font-mono text-[11px]">
              github.com/morphy76/vuhive
            </code>
            . Blocked packages (
            <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 font-mono text-[11px]">
              os/exec, syscall, unsafe
            </code>
            , plugin, runtime/cgo) are rejected at the pre-build stage.
          </InfoBadge>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="min-h-[40px]"
          >
            Cancel
          </Button>
          <Button
            onClick={() => onOpenChange(false)}
            className="min-h-[40px]"
          >
            Create Suite
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
