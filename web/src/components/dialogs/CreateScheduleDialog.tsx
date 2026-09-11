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
import { CronBuilder } from '@/components/schedules/CronBuilder'
import { useSuites } from '@/hooks/use-suites'
import { useProfiles } from '@/hooks/use-profiles'
import { useCreateSchedule } from '@/hooks/use-schedules'
import { api } from '@/lib/api'
import { validateCron } from '@/lib/cron-utils'
import type { CompiledArtifact, SuiteConfiguration } from '@/types/suite'
import { Loader2 } from 'lucide-react'

export interface CreateScheduleDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const CreateScheduleDialog: React.FC<CreateScheduleDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const { data: suites = [] } = useSuites()
  const { data: profiles = [] } = useProfiles()
  const createScheduleMutation = useCreateSchedule()

  const [scheduleName, setScheduleName] = React.useState('')
  const [selectedSuiteId, setSelectedSuiteId] = React.useState('')
  const [selectedArtifactId, setSelectedArtifactId] = React.useState('')
  const [selectedConfigId, setSelectedConfigId] = React.useState<string>('')
  const [selectedProfileId, setSelectedProfileId] = React.useState('')
  const [cronExpr, setCronExpr] = React.useState('0 2 * * *')
  const [artifacts, setArtifacts] = React.useState<CompiledArtifact[]>([])
  const [configs, setConfigs] = React.useState<SuiteConfiguration[]>([])
  const [isLoadingSuiteData, setIsLoadingSuiteData] = React.useState(false)
  const [errorMsg, setErrorMsg] = React.useState<string | null>(null)

  // Auto-select first active suite and profile on open
  React.useEffect(() => {
    if (open) {
      setErrorMsg(null)
      if (suites.length > 0 && !selectedSuiteId) {
        const activeSuite = suites.find((s) => s.state === 'ACTIVE') || suites[0]
        setSelectedSuiteId(activeSuite.id)
      }
      if (profiles.length > 0 && !selectedProfileId) {
        setSelectedProfileId(profiles[0].id)
      }
    }
  }, [open, suites, profiles, selectedSuiteId, selectedProfileId])

  // Load artifacts and configs when selectedSuiteId changes
  React.useEffect(() => {
    if (!selectedSuiteId) {
      setArtifacts([])
      setConfigs([])
      return
    }

    let isMounted = true
    setIsLoadingSuiteData(true)

    Promise.all([
      api.getSuiteArtifacts(selectedSuiteId),
      api.getSuiteConfigs(selectedSuiteId),
    ])
      .then(([loadedArtifacts, loadedConfigs]) => {
        if (!isMounted) return
        const readyArtifacts = loadedArtifacts.filter((a) => a.status === 'READY')
        setArtifacts(readyArtifacts)
        setConfigs(loadedConfigs)

        if (readyArtifacts.length > 0) {
          setSelectedArtifactId(readyArtifacts[0].id)
        } else {
          setSelectedArtifactId('')
        }

        const defConfig = loadedConfigs.find((c) => c.isDefault)
        if (defConfig) {
          setSelectedConfigId(defConfig.id)
        } else if (loadedConfigs.length > 0) {
          setSelectedConfigId(loadedConfigs[0].id)
        } else {
          setSelectedConfigId('')
        }
      })
      .catch((err) => {
        console.error('Failed loading suite artifacts/configs:', err)
      })
      .finally(() => {
        if (isMounted) setIsLoadingSuiteData(false)
      })

    return () => {
      isMounted = false
    }
  }, [selectedSuiteId])

  const cronValidation = validateCron(cronExpr)
  const isFormValid =
    scheduleName.trim() !== '' &&
    selectedSuiteId !== '' &&
    selectedArtifactId !== '' &&
    selectedProfileId !== '' &&
    cronValidation.isValid

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isFormValid) return

    setErrorMsg(null)
    try {
      await createScheduleMutation.mutateAsync({
        name: scheduleName.trim(),
        suite_id: selectedSuiteId,
        artifact_id: selectedArtifactId,
        configuration_id: selectedConfigId ? selectedConfigId : undefined,
        runner_profile_id: selectedProfileId,
        cron_expression: cronExpr.trim(),
      })

      // Reset and close
      setScheduleName('')
      onOpenChange(false)
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to create schedule')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-xl max-h-[90vh] overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>New Cron Schedule</DialogTitle>
          <DialogDescription>
            Schedule recurring load test executions as native Kubernetes CronJobs.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {errorMsg && (
            <div className="p-3 text-xs rounded-xl bg-red-50 dark:bg-red-950/40 border border-red-200 dark:border-red-900 text-red-700 dark:text-red-300">
              {errorMsg}
            </div>
          )}

          {/* Schedule Name */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="schedule-name"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Schedule Name
              </label>
              <HelpTooltip
                text="A descriptive name for your recurring schedule (e.g. Nightly Soak Test)."
                label="Help for schedule name"
              />
            </div>
            <input
              id="schedule-name"
              type="text"
              required
              value={scheduleName}
              onChange={(e) => setScheduleName(e.target.value)}
              placeholder="e.g. Nightly Soak Test (2h)"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            />
          </div>

          {/* Target Suite */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="suite-select"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Target Test Suite
                </label>
                <HelpTooltip
                  text="Select the test suite containing scenarios to execute on schedule."
                  label="Help for target suite"
                />
              </div>
              <select
                id="suite-select"
                aria-label="Target Test Suite"
                value={selectedSuiteId}
                onChange={(e) => setSelectedSuiteId(e.target.value)}
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              >
                {suites.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} ({s.state})
                  </option>
                ))}
              </select>
            </div>

            {/* Runner Profile */}
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="profile-select"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Runner Profile
                </label>
                <HelpTooltip
                  text="CPU, memory, node selector, and sandboxing profile for the runner pods."
                  label="Help for runner profile"
                />
              </div>
              <select
                id="profile-select"
                aria-label="Runner Profile"
                value={selectedProfileId}
                onChange={(e) => setSelectedProfileId(e.target.value)}
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              >
                {profiles.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} ({p.cpu_request} CPU, {p.memory_request} RAM)
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Artifact & Config Selection */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="artifact-select"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Compiled Artifact
                </label>
                <HelpTooltip
                  text="The compiled test binary matching target runner architecture."
                  label="Help for compiled artifact"
                />
              </div>
              {isLoadingSuiteData ? (
                <div className="flex items-center gap-2 py-2 text-xs text-slate-400">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  <span>Loading artifacts...</span>
                </div>
              ) : artifacts.length === 0 ? (
                <div className="text-xs text-amber-600 dark:text-amber-400 py-1">
                  No compiled READY artifacts found for this suite.
                </div>
              ) : (
                <select
                  id="artifact-select"
                  aria-label="Compiled Artifact"
                  value={selectedArtifactId}
                  onChange={(e) => setSelectedArtifactId(e.target.value)}
                  className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                >
                  {artifacts.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.platform} ({a.id.slice(0, 12)}...)
                    </option>
                  ))}
                </select>
              )}
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="config-select"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  YAML Configuration (Optional)
                </label>
                <HelpTooltip
                  text="Runtime YAML configuration (duration, concurrency, ramp-up)."
                  label="Help for configuration"
                />
              </div>
              <select
                id="config-select"
                aria-label="YAML Configuration"
                value={selectedConfigId}
                onChange={(e) => setSelectedConfigId(e.target.value)}
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              >
                <option value="">Default configuration</option>
                {configs.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name} {c.isDefault ? '(Default)' : ''}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Interactive Cron Builder */}
          <CronBuilder value={cronExpr} onChange={setCronExpr} />

          <InfoBadge variant="info" title="UTC Timezone Execution">
            Native Kubernetes CronJobs evaluate schedules strictly in UTC. Account for local
            daylight saving time (DST) shifts when defining launch windows.
          </InfoBadge>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="min-h-[40px]"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!isFormValid || createScheduleMutation.isPending}
              className="min-h-[40px] gap-2"
            >
              {createScheduleMutation.isPending && (
                <Loader2 className="w-4 h-4 animate-spin" />
              )}
              <span>Save Schedule</span>
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
