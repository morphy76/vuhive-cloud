import * as React from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
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
import { useSuites } from '@/hooks/use-suites'
import { useProfiles } from '@/hooks/use-profiles'
import { useTriggerRun } from '@/hooks/use-runs'
import { api } from '@/lib/api'
import { queryClient as fallbackQueryClient } from '@/lib/query-client'
import type { HistoricalRun } from '@/types/suite'

function useSafeQueryClient() {
  try {
    return useQueryClient()
  } catch {
    return fallbackQueryClient
  }
}

export interface TriggerRunDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialSuiteId?: string
  onRunTriggered?: (run: HistoricalRun) => void
}

export const TriggerRunDialog: React.FC<TriggerRunDialogProps> = ({
  open,
  onOpenChange,
  initialSuiteId,
  onRunTriggered,
}) => {
  const { data: suites = [] } = useSuites()
  const { data: profiles = [] } = useProfiles()
  const triggerRunMutation = useTriggerRun()

  const [selectedSuiteId, setSelectedSuiteId] = React.useState<string>(initialSuiteId || '')
  const [selectedPlatform, setSelectedPlatform] = React.useState<'all' | 'linux/amd64' | 'linux/arm64'>('all')
  const [selectedArtifactId, setSelectedArtifactId] = React.useState<string>('')
  const [selectedConfigId, setSelectedConfigId] = React.useState<string>('')
  const [selectedProfileId, setSelectedProfileId] = React.useState<string>('')
  const [activeDeadlineSeconds, setActiveDeadlineSeconds] = React.useState<string>('3600')
  const [errorAlert, setErrorAlert] = React.useState<string | null>(null)

  // Technical runner parameter state (backwards compatibility for guidance tooltips)
  const [pods, setPods] = React.useState('8')
  const [cpu, setCpu] = React.useState('1000m')
  const [memory, setMemory] = React.useState('1Gi')
  const [tolerations, setTolerations] = React.useState('dedicated=loadgen:NoSchedule')
  const [barrierEnabled, setBarrierEnabled] = React.useState(true)

  // Set default suite when open
  React.useEffect(() => {
    if (initialSuiteId) {
      setSelectedSuiteId(initialSuiteId)
    } else if (!selectedSuiteId && suites.length > 0) {
      const activeSuite = suites.find((s) => s.state === 'ACTIVE') || suites[0]
      setSelectedSuiteId(activeSuite.id)
    }
  }, [initialSuiteId, suites, selectedSuiteId])

  const queryClient = useSafeQueryClient()

  // Fetch compiled artifacts for selected suite
  const { data: artifacts = [] } = useQuery(
    {
      queryKey: ['suites', selectedSuiteId, 'artifacts'],
      queryFn: () => api.getSuiteArtifacts(selectedSuiteId),
      enabled: Boolean(selectedSuiteId),
    },
    queryClient
  )

  // Fetch configs for selected suite
  const { data: configs = [] } = useQuery(
    {
      queryKey: ['suites', selectedSuiteId, 'configs'],
      queryFn: () => api.getSuiteConfigs(selectedSuiteId),
      enabled: Boolean(selectedSuiteId),
    },
    queryClient
  )

  // Filter artifacts by selected platform
  const filteredArtifacts = React.useMemo(() => {
    return artifacts.filter((a) => {
      if (selectedPlatform === 'all') return true
      return a.platform === selectedPlatform
    })
  }, [artifacts, selectedPlatform])

  // Auto-select ready artifact when artifacts change
  React.useEffect(() => {
    if (filteredArtifacts.length > 0) {
      const ready = filteredArtifacts.find((a) => a.status === 'READY') || filteredArtifacts[0]
      setSelectedArtifactId(ready.id)
    } else {
      setSelectedArtifactId('')
    }
  }, [filteredArtifacts])

  // Auto-select default config when configs load
  React.useEffect(() => {
    if (configs.length > 0) {
      const def = configs.find((c) => c.isDefault)
      if (def && !selectedConfigId) {
        setSelectedConfigId(def.id)
      }
    }
  }, [configs, selectedConfigId])

  // Auto-select first profile if none selected
  React.useEffect(() => {
    if (!selectedProfileId && profiles.length > 0) {
      setSelectedProfileId(profiles[0].id)
      const p = profiles[0]
      setCpu(p.cpu_limit || p.cpu_request || '1000m')
      setMemory(p.memory_limit || p.memory_request || '1Gi')
      if (p.active_deadline_seconds) {
        setActiveDeadlineSeconds(String(p.active_deadline_seconds))
      }
      if (p.tolerations && p.tolerations.length > 0) {
        const tol = p.tolerations[0]
        setTolerations(`${tol.key || ''}=${tol.value || ''}:${tol.effect || 'NoSchedule'}`)
      }
    }
  }, [profiles, selectedProfileId])

  const handleProfileChange = (profileId: string) => {
    setSelectedProfileId(profileId)
    if (!profileId) return
    const p = profiles.find((item) => item.id === profileId)
    if (p) {
      setCpu(p.cpu_limit || p.cpu_request || '1000m')
      setMemory(p.memory_limit || p.memory_request || '1Gi')
      if (p.active_deadline_seconds) {
        setActiveDeadlineSeconds(String(p.active_deadline_seconds))
      }
      if (p.tolerations && p.tolerations.length > 0) {
        const tol = p.tolerations[0]
        setTolerations(`${tol.key || ''}=${tol.value || ''}:${tol.effect || 'NoSchedule'}`)
      } else {
        setTolerations('')
      }
    }
  }

  const handleDispatch = async () => {
    setErrorAlert(null)
    if (!selectedSuiteId) {
      setErrorAlert('Please select a target test suite.')
      return
    }
    if (!selectedArtifactId) {
      setErrorAlert('Please select a compiled artifact.')
      return
    }
    if (!selectedProfileId) {
      setErrorAlert('Please select a runner profile.')
      return
    }

    try {
      const timeoutVal = parseInt(activeDeadlineSeconds, 10)
      const newRun = await triggerRunMutation.mutateAsync({
        suite_id: selectedSuiteId,
        artifact_id: selectedArtifactId,
        runner_profile_id: selectedProfileId,
        configuration_id: selectedConfigId || undefined,
        active_deadline_seconds: !isNaN(timeoutVal) && timeoutVal > 0 ? timeoutVal : undefined,
      })

      onRunTriggered?.(newRun)
      onOpenChange(false)
    } catch (err: any) {
      setErrorAlert(err.message || 'Failed to dispatch load test run.')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-xl max-h-[90vh] overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>Execute Test Run</DialogTitle>
          <DialogDescription>
            Dispatch an on-demand distributed load test run across Kubernetes worker pods.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2 text-xs">
          {errorAlert && (
            <div className="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900 text-rose-700 dark:text-rose-300 font-medium">
              {errorAlert}
            </div>
          )}

          {/* Test Suite Dropdown */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="suite-select"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Test Suite
              </label>
              <HelpTooltip
                text="The target test suite scenario defining load generation behavior."
                label="Help for test suite selection"
              />
            </div>
            <select
              id="suite-select"
              value={selectedSuiteId}
              onChange={(e) => setSelectedSuiteId(e.target.value)}
              disabled={Boolean(initialSuiteId)}
              aria-label="Test Suite"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-60"
            >
              {suites.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} ({s.state})
                </option>
              ))}
            </select>
          </div>

          {/* Compiled Artifact Selection with Platform Filter */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="artifact-select"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Compiled Artifact
                </label>
                <HelpTooltip
                  text="Compiled binary artifact for the test suite, filtered by target CPU architecture."
                  label="Help for compiled artifact selection"
                />
              </div>

              {/* Platform Selector Buttons */}
              <div className="flex items-center gap-1 bg-slate-100 dark:bg-slate-800 p-0.5 rounded-lg text-[10px]">
                {(['all', 'linux/amd64', 'linux/arm64'] as const).map((plat) => (
                  <button
                    key={plat}
                    type="button"
                    onClick={() => setSelectedPlatform(plat)}
                    className={`px-2 py-0.5 rounded-md font-medium transition-colors ${
                      selectedPlatform === plat
                        ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-2xs'
                        : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
                    }`}
                  >
                    {plat}
                  </button>
                ))}
              </div>
            </div>

            <select
              id="artifact-select"
              value={selectedArtifactId}
              onChange={(e) => setSelectedArtifactId(e.target.value)}
              aria-label="Compiled Artifact"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            >
              {filteredArtifacts.length === 0 && (
                <option value="">No artifacts available for platform</option>
              )}
              {filteredArtifacts.map((a) => (
                <option key={a.id} value={a.id} disabled={a.status !== 'READY'}>
                  {a.platform} - {a.id} ({a.status})
                  {a.sha256Checksum ? ` [${a.sha256Checksum.substring(0, 8)}]` : ''}
                </option>
              ))}
            </select>
          </div>

          {/* Scenario Configuration Dropdown */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="configuration-select"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Scenario Configuration
              </label>
              <HelpTooltip
                text="Optional vuhive.yaml execution profile defining duration, concurrency, and thresholds."
                label="Help for scenario configuration"
              />
            </div>
            <select
              id="configuration-select"
              value={selectedConfigId}
              onChange={(e) => setSelectedConfigId(e.target.value)}
              aria-label="Scenario Configuration"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            >
              <option value="">None (Embedded Defaults)</option>
              {configs.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} {c.isDefault ? '(Default)' : ''}
                </option>
              ))}
            </select>
          </div>

          {/* Runner Profile Dropdown */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="runner-profile-select"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Runner Profile
              </label>
              <HelpTooltip
                text="Select a pre-registered Runner Profile to auto-fill compute limits and scheduling rules, or customize manually."
                label="Help for runner profile selection"
              />
            </div>
            <select
              id="runner-profile-select"
              value={selectedProfileId}
              onChange={(e) => handleProfileChange(e.target.value)}
              aria-label="Runner Profile"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            >
              <option value="">Custom Configuration</option>
              {profiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} (CPU: {p.cpu_request}/{p.cpu_limit}, RAM: {p.memory_request}/{p.memory_limit})
                </option>
              ))}
            </select>
          </div>

          {/* Execution Timeout Override Input */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="active-deadline-seconds"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Execution Timeout (seconds)
              </label>
              <HelpTooltip
                text="Optional activeDeadlineSeconds override for the Kubernetes Job. Pods are safely killed after this duration."
                label="Help for execution timeout"
              />
            </div>
            <input
              id="active-deadline-seconds"
              type="number"
              value={activeDeadlineSeconds}
              onChange={(e) => setActiveDeadlineSeconds(e.target.value)}
              placeholder="3600"
              aria-label="Execution Timeout (seconds)"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            />
          </div>

          {/* Runner Pods & Resource Specifications */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-pods"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Runner Pods Count
                </label>
                <HelpTooltip
                  text="Number of parallel worker pods to spawn in Kubernetes."
                  label="Help for runner pods count"
                />
              </div>
              <input
                id="runner-pods"
                type="number"
                value={pods}
                onChange={(e) => setPods(e.target.value)}
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-cpu"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  CPU Millicores
                </label>
                <HelpTooltip
                  text="Kubernetes CPU allocation in millicores (e.g., 500m = 0.5 CPU, 1000m = 1 vCPU). Limits prevent noisy-neighbor throttling."
                  label="Help for cpu allocation"
                />
              </div>
              <input
                id="runner-cpu"
                type="text"
                value={cpu}
                onChange={(e) => setCpu(e.target.value)}
                placeholder="1000m"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-memory"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Memory Units
                </label>
                <HelpTooltip
                  text="Kubernetes memory allocation using binary SI units (e.g., 512Mi, 1Gi, 2Gi). Exceeding limits triggers OOMKill."
                  label="Help for memory allocation"
                />
              </div>
              <input
                id="runner-memory"
                type="text"
                value={memory}
                onChange={(e) => setMemory(e.target.value)}
                placeholder="1Gi"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-tolerations"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Node Tolerations
                </label>
                <HelpTooltip
                  text="Kubernetes tolerations syntax (key=value:NoSchedule) allowing runner pods to schedule onto dedicated tainted load-generation worker nodes."
                  label="Help for node tolerations"
                />
              </div>
              <input
                id="runner-tolerations"
                type="text"
                value={tolerations}
                onChange={(e) => setTolerations(e.target.value)}
                placeholder="dedicated=loadgen:NoSchedule"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>
          </div>

          <div className="flex items-center justify-between p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800">
            <div className="space-y-0.5">
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-semibold text-slate-800 dark:text-slate-200">
                  Distributed Start Barrier
                </span>
                <HelpTooltip
                  text="Zero clock-skew distributed rendezvous barrier coordinating simultaneous test execution across all pods."
                  label="Help for start barrier"
                />
              </div>
              <p className="text-[11px] text-slate-500 dark:text-slate-400">
                Hold load generators until all pods reach rendezvous readiness.
              </p>
            </div>
            <Switch
              checked={barrierEnabled}
              onCheckedChange={setBarrierEnabled}
              aria-label="Toggle distributed start barrier"
            />
          </div>

          <InfoBadge
            variant="info"
            title="Kubernetes Runner Resource Allocation"
          >
            Configure memory requests equal to limits for Guaranteed Quality of Service (QoS).
            Allocate sufficient CPU millicores (e.g. 1000m) to minimize garbage collection latency
            impacts during high-throughput benchmark runs.
          </InfoBadge>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={triggerRunMutation.isPending}
            className="min-h-[40px]"
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleDispatch}
            disabled={triggerRunMutation.isPending}
            className="min-h-[40px]"
          >
            {triggerRunMutation.isPending ? 'Dispatching...' : 'Dispatch Run'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
