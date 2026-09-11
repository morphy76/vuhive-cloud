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
import { useCreateSuite, useSuites } from '@/hooks/use-suites'
import type { TestSuite } from '@/types/suite'

export interface CreateSuiteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  existingSuites?: TestSuite[]
  onCreated?: (suite: TestSuite) => void
}

export const CreateSuiteDialog: React.FC<CreateSuiteDialogProps> = ({
  open,
  onOpenChange,
  existingSuites,
  onCreated,
}) => {
  const [suiteName, setSuiteName] = React.useState('')
  const [suiteDescription, setSuiteDescription] = React.useState('')
  const [selectedArchs, setSelectedArchs] = React.useState<string[]>([
    'linux/amd64',
    'linux/arm64',
  ])
  const [hasInteracted, setHasInteracted] = React.useState(false)

  const toggleArch = (arch: string) => {
    setSelectedArchs((prev) =>
      prev.includes(arch) ? prev.filter((a) => a !== arch) : [...prev, arch]
    )
  }

  const { data: fetchedSuites } = useSuites()
  const suitesPool = existingSuites ?? fetchedSuites ?? []

  const createMutation = useCreateSuite()

  const trimmedName = suiteName.trim()
  const isDuplicate = React.useMemo(() => {
    if (!trimmedName) return false
    return suitesPool.some(
      (s) => s.name.trim().toLowerCase() === trimmedName.toLowerCase()
    )
  }, [trimmedName, suitesPool])

  const nameError = React.useMemo(() => {
    if (!hasInteracted) return null
    if (!trimmedName) return 'Suite name is required.'
    if (isDuplicate) return 'A suite with this name already exists.'
    return null
  }, [hasInteracted, trimmedName, isDuplicate])

  const isValid = trimmedName.length > 0 && !isDuplicate

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setHasInteracted(true)
    if (!isValid || createMutation.isPending) return

    try {
      const created = await createMutation.mutateAsync({
        name: trimmedName,
        description: suiteDescription.trim(),
      })
      setSuiteName('')
      setSuiteDescription('')
      setHasInteracted(false)
      onOpenChange(false)
      onCreated?.(created)
    } catch (err) {
      console.error('Failed to create suite:', err)
    }
  }

  const handleClose = (newOpen: boolean) => {
    if (!newOpen) {
      setSuiteName('')
      setSuiteDescription('')
      setHasInteracted(false)
    }
    onOpenChange(newOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent
        className="sm:max-w-lg"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>New Test Suite</DialogTitle>
            <DialogDescription>
              Configure and register a test scenario aggregate. Source packages and configurations can be attached once registered.
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
                  text="A unique descriptive name for your load test scenario (e.g. Checkout & Payment Stress Test)."
                  label="Help for suite name"
                />
              </div>
              <input
                id="suite-name"
                type="text"
                value={suiteName}
                onChange={(e) => {
                  setSuiteName(e.target.value)
                  setHasInteracted(true)
                }}
                placeholder="e.g. Checkout & Payment Stress Test"
                aria-invalid={nameError ? 'true' : 'false'}
                aria-describedby={nameError ? 'suite-name-error' : undefined}
                className={`w-full rounded-xl border px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 bg-white dark:bg-slate-900 transition-colors ${
                  nameError
                    ? 'border-red-500 focus-visible:ring-red-500'
                    : 'border-slate-200 dark:border-slate-800 focus-visible:ring-brand-500'
                }`}
              />
              {nameError && (
                <p id="suite-name-error" className="text-xs text-red-500 font-medium">
                  {nameError}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="suite-description"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Description (Optional)
                </label>
                <HelpTooltip
                  text="Brief context or objectives regarding the traffic profile and simulated workloads."
                  label="Help for suite description"
                />
              </div>
              <textarea
                id="suite-description"
                rows={3}
                value={suiteDescription}
                onChange={(e) => setSuiteDescription(e.target.value)}
                placeholder="Describe scenario traffic patterns, target endpoints, or SLAs..."
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 resize-none"
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
              variant="info"
              title="Suite Aggregate Lifecycle"
            >
              Suites are registered in{' '}
              <code className="px-1 py-0.5 rounded bg-blue-100 dark:bg-blue-900/60 font-mono text-[11px]">
                DRAFT
              </code>{' '}
              state. Once compiled binaries or scenario configs are attached, the suite transitions to{' '}
              <code className="px-1 py-0.5 rounded bg-emerald-100 dark:bg-emerald-900/60 font-mono text-[11px]">
                ACTIVE
              </code>{' '}
              for Kubernetes runner execution.
            </InfoBadge>

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

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleClose(false)}
              className="min-h-[44px]"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!isValid || createMutation.isPending}
              className="min-h-[44px]"
            >
              {createMutation.isPending ? 'Creating...' : 'Create Suite'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
