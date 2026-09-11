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
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { YamlEditor } from '@/components/editor/YamlEditor'
import { YamlDiffViewer } from '@/components/editor/YamlDiffViewer'
import { useCreateSuiteConfig, useUpdateSuiteConfig } from '@/hooks/use-suites'
import type { SuiteConfiguration } from '@/types/suite'
import type { YamlValidationResult } from '@/lib/yaml-validator'
import { FileCode, Split, CopyPlus } from 'lucide-react'

export interface ConfigEditorDialogProps {
  suiteId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  initialConfig?: SuiteConfiguration | null
  existingConfigs?: SuiteConfiguration[]
}

const DEFAULT_STARTER_YAML = `version: "1.0"
execution:
  vus: 50
  duration: 60s
  ramp_up: 10s
thresholds:
  p95_latency_ms: 250
  error_rate_pct: 1.0
`

export const ConfigEditorDialog: React.FC<ConfigEditorDialogProps> = ({
  suiteId,
  open,
  onOpenChange,
  initialConfig,
  existingConfigs = [],
}) => {
  const isEditMode = Boolean(initialConfig)

  const [configName, setConfigName] = React.useState('')
  const [contentYaml, setContentYaml] = React.useState(DEFAULT_STARTER_YAML)
  const [isDefault, setIsDefault] = React.useState(false)
  const [activeTab, setActiveTab] = React.useState<'editor' | 'diff'>('editor')
  const [isValidYaml, setIsValidYaml] = React.useState(true)
  const [diffBaseId, setDiffBaseId] = React.useState<string>('original')

  const createMutation = useCreateSuiteConfig(suiteId)
  const updateMutation = useUpdateSuiteConfig(suiteId)

  // Sync state when dialog opens or initialConfig changes
  React.useEffect(() => {
    if (open) {
      if (initialConfig) {
        setConfigName(initialConfig.name)
        setContentYaml(initialConfig.contentYaml)
        setIsDefault(initialConfig.isDefault)
      } else {
        setConfigName('')
        setContentYaml(DEFAULT_STARTER_YAML)
        setIsDefault(false)
      }
      setActiveTab('editor')
      setDiffBaseId('original')
    }
  }, [open, initialConfig])

  // Resolve base YAML for diff comparison
  const diffBaseYaml = React.useMemo(() => {
    if (diffBaseId === 'original') {
      return initialConfig ? initialConfig.contentYaml : DEFAULT_STARTER_YAML
    }
    const found = existingConfigs.find((c) => c.id === diffBaseId)
    return found ? found.contentYaml : ''
  }, [diffBaseId, initialConfig, existingConfigs])

  const handleValidationChange = React.useCallback((result: YamlValidationResult) => {
    setIsValidYaml(result.isValid)
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!configName.trim() || !contentYaml.trim() || !isValidYaml) return

    try {
      if (isEditMode && initialConfig) {
        await updateMutation.mutateAsync({
          configId: initialConfig.id,
          data: {
            name: configName.trim(),
            content_yaml: contentYaml,
            is_default: isDefault,
          },
        })
      } else {
        await createMutation.mutateAsync({
          name: configName.trim(),
          content_yaml: contentYaml,
          is_default: isDefault,
        })
      }
      onOpenChange(false)
    } catch (err) {
      console.error('Failed saving scenario configuration:', err)
    }
  }

  const handleSaveAsNewVersion = async () => {
    if (!configName.trim() || !contentYaml.trim() || !isValidYaml) return

    try {
      // Suggest incremented or distinct name if name hasn't changed
      let targetName = configName.trim()
      if (isEditMode && initialConfig && targetName === initialConfig.name) {
        targetName = targetName.replace(/(\.ya?ml)?$/, '-copy$1')
      }

      await createMutation.mutateAsync({
        name: targetName,
        content_yaml: contentYaml,
        is_default: isDefault,
      })
      onOpenChange(false)
    } catch (err) {
      console.error('Failed creating new configuration version:', err)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-3xl max-h-[90vh] overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {isEditMode ? `Edit Configuration: ${initialConfig?.name}` : 'Attach Scenario Configuration'}
            </DialogTitle>
            <DialogDescription>
              {isEditMode
                ? 'Tune scenario virtual users, durations, stage plateaus, and latency thresholds with real-time YAML validation and side-by-side diffing.'
                : 'Upload or declare scenario parameters, duration, ramp-up curves, and concurrency limits in YAML format.'}
            </DialogDescription>
          </DialogHeader>

          {/* Configuration Metadata Controls */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-1">
            <div className="sm:col-span-2 space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="config-name"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Configuration Name
                </label>
                <HelpTooltip
                  text="Descriptive filename for this configuration (e.g. production-stress.yaml)."
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

            <div className="flex items-center justify-between rounded-xl border border-slate-200 dark:border-slate-800 p-3 bg-slate-50 dark:bg-slate-800/40">
              <div className="space-y-0.5 pr-2">
                <span className="text-xs font-semibold text-slate-900 dark:text-white">
                  Default Config
                </span>
                <p className="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-1">
                  Auto-selected
                </p>
              </div>
              <Switch
                checked={isDefault}
                onCheckedChange={setIsDefault}
                aria-label="Set as default configuration"
              />
            </div>
          </div>

          {/* Editor and Diff Tabs */}
          <Tabs value={activeTab} onValueChange={(val) => setActiveTab(val as 'editor' | 'diff')} className="space-y-3">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 dark:border-slate-800 pb-1">
              <TabsList className="bg-slate-100 dark:bg-slate-800/80 p-1 rounded-xl">
                <TabsTrigger value="editor" className="gap-1.5 text-xs min-h-[34px] px-3">
                  <FileCode className="w-3.5 h-3.5" />
                  <span>Editor</span>
                </TabsTrigger>
                <TabsTrigger value="diff" className="gap-1.5 text-xs min-h-[34px] px-3">
                  <Split className="w-3.5 h-3.5" />
                  <span>Diff View</span>
                </TabsTrigger>
              </TabsList>

              {activeTab === 'diff' && existingConfigs.length > 0 && (
                <div className="flex items-center gap-2 text-xs">
                  <span className="text-slate-500">Compare against:</span>
                  <select
                    value={diffBaseId}
                    onChange={(e) => setDiffBaseId(e.target.value)}
                    className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-2 py-1 text-xs text-slate-700 dark:text-slate-300"
                  >
                    <option value="original">
                      {isEditMode ? 'Original Version' : 'Starter Template'}
                    </option>
                    {existingConfigs.map((cfg) => (
                      <option key={cfg.id} value={cfg.id}>
                        {cfg.name} ({cfg.id.slice(0, 8)})
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>

            <TabsContent value="editor" className="m-0">
              <YamlEditor
                value={contentYaml}
                onChange={setContentYaml}
                showTemplates={true}
                height="320px"
                onValidationChange={handleValidationChange}
                ariaLabel="Scenario Configuration YAML Editor"
              />
            </TabsContent>

            <TabsContent value="diff" className="m-0">
              <YamlDiffViewer
                baseYaml={diffBaseYaml}
                comparisonYaml={contentYaml}
                baseTitle={
                  diffBaseId === 'original'
                    ? isEditMode
                      ? 'Original Configuration'
                      : 'Default Starter'
                    : `Base: ${existingConfigs.find((c) => c.id === diffBaseId)?.name || diffBaseId}`
                }
                comparisonTitle="Current Edited YAML"
              />
            </TabsContent>
          </Tabs>

          <DialogFooter className="gap-2 sm:gap-2 flex-wrap sm:flex-nowrap">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="min-h-[44px]"
            >
              Cancel
            </Button>

            {isEditMode && (
              <Button
                type="button"
                variant="outline"
                onClick={handleSaveAsNewVersion}
                disabled={!configName.trim() || !contentYaml.trim() || !isValidYaml || isPending}
                className="min-h-[44px] gap-1.5"
              >
                <CopyPlus className="w-4 h-4" />
                <span>Save as New Version</span>
              </Button>
            )}

            <Button
              type="submit"
              disabled={!configName.trim() || !contentYaml.trim() || !isValidYaml || isPending}
              className="min-h-[44px]"
            >
              {isPending
                ? 'Saving...'
                : isEditMode
                ? 'Save Changes'
                : 'Attach Configuration'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
