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
import { YamlDiffViewer } from '@/components/editor/YamlDiffViewer'
import type { SuiteConfiguration } from '@/types/suite'
import { ArrowLeftRight } from 'lucide-react'

export interface ConfigDiffDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  configs: SuiteConfiguration[]
  initialBaseId?: string
  initialComparisonId?: string
}

export const ConfigDiffDialog: React.FC<ConfigDiffDialogProps> = ({
  open,
  onOpenChange,
  configs,
  initialBaseId,
  initialComparisonId,
}) => {
  const [baseId, setBaseId] = React.useState<string>('')
  const [comparisonId, setComparisonId] = React.useState<string>('')

  React.useEffect(() => {
    if (open) {
      if (initialBaseId && configs.some((c) => c.id === initialBaseId)) {
        setBaseId(initialBaseId)
      } else if (configs.length > 0) {
        setBaseId(configs[0].id)
      }

      if (initialComparisonId && configs.some((c) => c.id === initialComparisonId)) {
        setComparisonId(initialComparisonId)
      } else if (configs.length > 1) {
        setComparisonId(configs[1].id)
      } else if (configs.length > 0) {
        setComparisonId(configs[0].id)
      }
    }
  }, [open, configs, initialBaseId, initialComparisonId])

  const baseConfig = React.useMemo(() => configs.find((c) => c.id === baseId), [configs, baseId])
  const comparisonConfig = React.useMemo(
    () => configs.find((c) => c.id === comparisonId),
    [configs, comparisonId]
  )

  const handleSwap = () => {
    setBaseId(comparisonId)
    setComparisonId(baseId)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-4xl max-h-[90vh] overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>Compare Scenario Configurations</DialogTitle>
          <DialogDescription>
            Inspect side-by-side YAML diffs and line-by-line modifications between attached scenario configurations.
          </DialogDescription>
        </DialogHeader>

        {/* Configuration Selectors & Swap Action */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3 p-3 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/40">
          <div className="flex-1 w-full space-y-1">
            <label
              htmlFor="base-config-select"
              className="text-xs font-semibold text-slate-700 dark:text-slate-300"
            >
              Base Configuration (Left)
            </label>
            <select
              id="base-config-select"
              value={baseId}
              onChange={(e) => setBaseId(e.target.value)}
              className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            >
              {configs.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} {c.isDefault ? '(Default)' : ''} — {new Date(c.createdAt).toLocaleDateString()}
                </option>
              ))}
            </select>
          </div>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleSwap}
            aria-label="Swap configurations"
            className="h-8 px-2.5 mt-4 sm:mt-5 text-xs gap-1 border-slate-200 dark:border-slate-700"
          >
            <ArrowLeftRight className="w-3.5 h-3.5" />
            <span className="sr-only sm:not-sr-only">Swap</span>
          </Button>

          <div className="flex-1 w-full space-y-1">
            <label
              htmlFor="comp-config-select"
              className="text-xs font-semibold text-slate-700 dark:text-slate-300"
            >
              Comparison Configuration (Right)
            </label>
            <select
              id="comp-config-select"
              value={comparisonId}
              onChange={(e) => setComparisonId(e.target.value)}
              className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            >
              {configs.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} {c.isDefault ? '(Default)' : ''} — {new Date(c.createdAt).toLocaleDateString()}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Diff Display Surface */}
        <YamlDiffViewer
          baseYaml={baseConfig?.contentYaml || ''}
          comparisonYaml={comparisonConfig?.contentYaml || ''}
          baseTitle={`Base: ${baseConfig?.name || 'None'}`}
          comparisonTitle={`Target: ${comparisonConfig?.name || 'None'}`}
        />

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="min-h-[44px]"
          >
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
