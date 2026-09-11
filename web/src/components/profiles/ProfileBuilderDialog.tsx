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
import { Badge } from '@/components/ui/badge'
import { AlertCircle, CheckCircle2, Sparkles, Loader2, Code2, Eye } from 'lucide-react'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { ResourceSliderInput } from './ResourceSliderInput'
import { SchedulingRulesBuilder } from './SchedulingRulesBuilder'
import { ProfilePayloadPreview } from './ProfilePayloadPreview'
import { PROFILE_PRESETS } from '@/data/profile-presets'
import { validateResourceRequirements } from '@/lib/k8s-resources'
import { useCreateProfile, useUpdateProfile } from '@/hooks/use-profiles'
import { useToast } from '@/hooks/use-toast'
import type {
  RunnerProfile,
  CreateProfileInput,
  NodeAffinityTerm,
  TolerationSpec,
} from '@/types/profile'

export interface ProfileBuilderDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialProfile?: RunnerProfile | null
  onSaved?: (profile: RunnerProfile) => void
}

export const ProfileBuilderDialog: React.FC<ProfileBuilderDialogProps> = ({
  open,
  onOpenChange,
  initialProfile,
  onSaved,
}) => {
  const isEditing = Boolean(initialProfile?.id)
  const createMutation = useCreateProfile()
  const updateMutation = useUpdateProfile(initialProfile?.id || '')
  const { toast } = useToast()

  // Form State
  const [name, setName] = React.useState('')
  const [description, setDescription] = React.useState('')
  const [runnerImage, setRunnerImage] = React.useState('alpine:3.20')
  const [cpuRequest, setCpuRequest] = React.useState('500m')
  const [cpuLimit, setCpuLimit] = React.useState('1000m')
  const [memoryRequest, setMemoryRequest] = React.useState('512Mi')
  const [memoryLimit, setMemoryLimit] = React.useState('1Gi')
  const [nodeSelector, setNodeSelector] = React.useState<Record<string, string>>({})
  const [affinityTerms, setAffinityTerms] = React.useState<NodeAffinityTerm[]>([])
  const [tolerations, setTolerations] = React.useState<TolerationSpec[]>([])
  const [activeDeadlineSeconds, setActiveDeadlineSeconds] = React.useState<number | undefined>(3600)
  const [runtimeClassName, setRuntimeClassName] = React.useState<string>('')
  const [showPayloadPreview, setShowPayloadPreview] = React.useState(false)
  const [hasInteracted, setHasInteracted] = React.useState(false)

  // Populate when opening or changing initialProfile
  React.useEffect(() => {
    if (initialProfile) {
      setName(initialProfile.name || '')
      setDescription(initialProfile.description || '')
      setRunnerImage(initialProfile.runner_image || 'alpine:3.20')
      setCpuRequest(initialProfile.cpu_request || '500m')
      setCpuLimit(initialProfile.cpu_limit || '1000m')
      setMemoryRequest(initialProfile.memory_request || '512Mi')
      setMemoryLimit(initialProfile.memory_limit || '1Gi')
      setNodeSelector(initialProfile.node_selector || {})
      setAffinityTerms(initialProfile.affinity?.node_selector_terms || [])
      setTolerations(initialProfile.tolerations || [])
      setActiveDeadlineSeconds(initialProfile.active_deadline_seconds ?? 3600)
      setRuntimeClassName(initialProfile.runtime_class_name || '')
      setShowPayloadPreview(false)
      setHasInteracted(false)
    } else {
      // Default to Standard Single-Node Preset
      const standard = PROFILE_PRESETS[0].values
      setName('')
      setDescription(standard.description || '')
      setRunnerImage(standard.runner_image || 'alpine:3.20')
      setCpuRequest(standard.cpu_request || '500m')
      setCpuLimit(standard.cpu_limit || '1000m')
      setMemoryRequest(standard.memory_request || '512Mi')
      setMemoryLimit(standard.memory_limit || '1Gi')
      setNodeSelector(standard.node_selector || {})
      setAffinityTerms(standard.affinity?.node_selector_terms || [])
      setTolerations(standard.tolerations || [])
      setActiveDeadlineSeconds(standard.active_deadline_seconds ?? 3600)
      setRuntimeClassName(standard.runtime_class_name || '')
      setShowPayloadPreview(false)
      setHasInteracted(false)
    }
  }, [initialProfile, open])

  // Quick-load preset handler
  const handleApplyPreset = (presetId: string) => {
    const preset = PROFILE_PRESETS.find((p) => p.id === presetId)
    if (!preset) return

    setHasInteracted(true)
    if (!isEditing) {
      setName(preset.values.name)
    }
    setDescription(preset.values.description || '')
    setRunnerImage(preset.values.runner_image || 'alpine:3.20')
    setCpuRequest(preset.values.cpu_request || '500m')
    setCpuLimit(preset.values.cpu_limit || '1000m')
    setMemoryRequest(preset.values.memory_request || '512Mi')
    setMemoryLimit(preset.values.memory_limit || '1Gi')
    setNodeSelector(preset.values.node_selector || {})
    setAffinityTerms(preset.values.affinity?.node_selector_terms || [])
    setTolerations(preset.values.tolerations || [])
    setActiveDeadlineSeconds(preset.values.active_deadline_seconds ?? 3600)
    setRuntimeClassName(preset.values.runtime_class_name || '')

    toast({
      title: `Applied Preset: ${preset.label}`,
      description: preset.description,
    })
  }

  // Validation
  const resourceValidation = React.useMemo(() => {
    return validateResourceRequirements({
      cpuRequest,
      cpuLimit,
      memoryRequest,
      memoryLimit,
    })
  }, [cpuRequest, cpuLimit, memoryRequest, memoryLimit])

  const trimmedName = name.trim()
  const nameError = React.useMemo(() => {
    if (!hasInteracted) return null
    if (!trimmedName) return 'Profile name is required.'
    if (!/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(trimmedName)) {
      return 'Name must consist of lowercase alphanumeric characters or "-", and must start and end with an alphanumeric character.'
    }
    return null
  }, [trimmedName, hasInteracted])

  const isFormValid = Boolean(
    trimmedName &&
      !nameError &&
      resourceValidation.isValid
  )

  // Construct generated payload for submission & preview
  const payloadToSubmit = React.useMemo((): CreateProfileInput => {
    const affinity =
      affinityTerms.length > 0
        ? {
            node_selector_terms: affinityTerms.map((t) => ({
              key: t.key.trim(),
              operator: t.operator,
              values: t.values && t.values.length > 0 ? t.values : undefined,
            })),
          }
        : undefined

    const cleanedTolerations =
      tolerations.length > 0
        ? tolerations.map((t) => ({
            key: t.key?.trim() || undefined,
            operator: t.operator || 'Equal',
            value: t.operator === 'Exists' ? undefined : t.value?.trim() || undefined,
            effect: t.effect || 'NoSchedule',
            toleration_seconds:
              t.effect === 'NoExecute' && typeof t.toleration_seconds === 'number'
                ? t.toleration_seconds
                : undefined,
          }))
        : undefined

    return {
      name: trimmedName,
      description: description.trim() || undefined,
      runner_image: runnerImage.trim() || 'alpine:3.20',
      cpu_request: cpuRequest.trim(),
      cpu_limit: cpuLimit.trim(),
      memory_request: memoryRequest.trim(),
      memory_limit: memoryLimit.trim(),
      node_selector: Object.keys(nodeSelector).length > 0 ? nodeSelector : undefined,
      affinity,
      tolerations: cleanedTolerations,
      active_deadline_seconds:
        typeof activeDeadlineSeconds === 'number' && activeDeadlineSeconds > 0
          ? activeDeadlineSeconds
          : undefined,
      runtime_class_name: runtimeClassName.trim() || undefined,
    }
  }, [
    trimmedName,
    description,
    runnerImage,
    cpuRequest,
    cpuLimit,
    memoryRequest,
    memoryLimit,
    nodeSelector,
    affinityTerms,
    tolerations,
    activeDeadlineSeconds,
    runtimeClassName,
  ])

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setHasInteracted(true)

    if (!isFormValid) return

    try {
      let result: RunnerProfile
      if (isEditing && initialProfile) {
        result = await updateMutation.mutateAsync(payloadToSubmit)
        toast({
          title: 'Profile Updated',
          description: `Runner profile "${result.name}" was successfully updated.`,
        })
      } else {
        result = await createMutation.mutateAsync(payloadToSubmit)
        toast({
          title: 'Profile Created',
          description: `Runner profile "${result.name}" has been registered in the control plane.`,
        })
      }
      onSaved?.(result)
      onOpenChange(false)
    } catch (err: any) {
      toast({
        variant: 'destructive',
        title: isEditing ? 'Update Failed' : 'Creation Failed',
        description: err.message || 'Failed to save runner profile.',
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-3xl max-h-[90vh] overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
            <div>
              <DialogTitle className="text-xl font-bold">
                {isEditing ? 'Edit Runner Profile' : 'Visual Runner Profile Builder'}
              </DialogTitle>
              <DialogDescription className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Configure compute resources, node affinity expressions, and pod tolerations for Kubernetes test executions.
              </DialogDescription>
            </div>
            {resourceValidation.isGuaranteedQoS && (
              <Badge variant="success" className="self-start sm:self-auto gap-1 text-[11px]">
                <CheckCircle2 className="w-3.5 h-3.5" />
                <span>Guaranteed QoS</span>
              </Badge>
            )}
          </div>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 py-2">
          {/* Preset Quick-Load Templates */}
          <div className="rounded-xl border border-brand-200 dark:border-brand-900/60 bg-brand-50/50 dark:bg-brand-950/20 p-3.5 space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-bold text-brand-900 dark:text-brand-300">
              <Sparkles className="w-4 h-4 text-brand-600 dark:text-brand-400" />
              <span>Quick-Load Preset Templates</span>
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Select an optimized architecture blueprint to prefill scheduling rules and resource allocations:
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 pt-1">
              {PROFILE_PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  type="button"
                  onClick={() => handleApplyPreset(preset.id)}
                  className="text-left rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-2.5 hover:border-brand-500 dark:hover:border-brand-500 hover:shadow-xs transition-all group"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-slate-900 dark:text-slate-100 group-hover:text-brand-600 dark:group-hover:text-brand-400">
                      {preset.label}
                    </span>
                    <Badge variant="outline" className="text-[10px] py-0 px-1.5">
                      {preset.badge}
                    </Badge>
                  </div>
                  <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1 line-clamp-2">
                    {preset.description}
                  </p>
                </button>
              ))}
            </div>
          </div>

          {/* General Information */}
          <div className="space-y-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 p-4">
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              Profile Metadata
            </h4>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <label
                    htmlFor="profile-name"
                    className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                  >
                    Profile Name <span className="text-rose-500">*</span>
                  </label>
                  <HelpTooltip
                    text="Unique alphanumeric profile identifier used when scheduling ad-hoc runs or CronJobs."
                    label="Help for profile name"
                  />
                </div>
                <input
                  id="profile-name"
                  type="text"
                  placeholder="e.g. high-throughput-dedicated"
                  value={name}
                  disabled={isEditing}
                  onChange={(e) => {
                    setName(e.target.value)
                    setHasInteracted(true)
                  }}
                  className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-60"
                />
                {nameError && (
                  <p className="text-[11px] text-rose-500 font-medium">{nameError}</p>
                )}
              </div>

              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <label
                    htmlFor="profile-image"
                    className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                  >
                    Runner Base Image
                  </label>
                  <HelpTooltip
                    text="Docker/OCI container image used as the runner wrapper execution environment."
                    label="Help for runner image"
                  />
                </div>
                <input
                  id="profile-image"
                  type="text"
                  placeholder="alpine:3.20"
                  value={runnerImage}
                  onChange={(e) => setRunnerImage(e.target.value)}
                  className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label
                htmlFor="profile-description"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Description
              </label>
              <input
                id="profile-description"
                type="text"
                placeholder="Operational purpose, node topology target, or test scope..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>
          </div>

          {/* Resource Configuration */}
          <div className="space-y-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 p-4">
            <div className="flex items-center justify-between">
              <div>
                <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
                  Compute Resources (CPU & Memory)
                </h4>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Configure request (minimum guaranteed) and limit (maximum throttle ceiling).
                </p>
              </div>
            </div>

            {/* Validation Warnings */}
            {(!resourceValidation.isValid || resourceValidation.cpuWarning || resourceValidation.memWarning) && (
              <div className="space-y-1.5">
                {resourceValidation.cpuWarning && (
                  <div className="flex items-center gap-2 rounded-lg bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/60 p-2.5 text-xs text-rose-700 dark:text-rose-300 font-medium">
                    <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
                    <span>{resourceValidation.cpuWarning}</span>
                  </div>
                )}
                {resourceValidation.memWarning && (
                  <div className="flex items-center gap-2 rounded-lg bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/60 p-2.5 text-xs text-rose-700 dark:text-rose-300 font-medium">
                    <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
                    <span>{resourceValidation.memWarning}</span>
                  </div>
                )}
              </div>
            )}

            {/* CPU Sliders */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <ResourceSliderInput
                idPrefix="cpu-request"
                label="CPU Request"
                helpText="Guaranteed minimum CPU allocated to the runner pod."
                type="cpu"
                value={cpuRequest}
                onChange={setCpuRequest}
              />
              <ResourceSliderInput
                idPrefix="cpu-limit"
                label="CPU Limit"
                helpText="Maximum ceiling CPU threshold before CFS scheduler throttles pod execution."
                type="cpu"
                value={cpuLimit}
                onChange={setCpuLimit}
              />
            </div>

            {/* Memory Sliders */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <ResourceSliderInput
                idPrefix="mem-request"
                label="Memory Request"
                helpText="Guaranteed minimum physical memory allocated to the runner container."
                type="memory"
                value={memoryRequest}
                onChange={setMemoryRequest}
              />
              <ResourceSliderInput
                idPrefix="mem-limit"
                label="Memory Limit"
                helpText="Maximum hard memory boundary. Crossing this threshold causes OOMKilled eviction."
                type="memory"
                value={memoryLimit}
                onChange={setMemoryLimit}
              />
            </div>
          </div>

          {/* Scheduling Rules Builder (Node Selectors, Affinities, Tolerations) */}
          <SchedulingRulesBuilder
            nodeSelector={nodeSelector}
            onNodeSelectorChange={setNodeSelector}
            affinityTerms={affinityTerms}
            onAffinityTermsChange={setAffinityTerms}
            tolerations={tolerations}
            onTolerationsChange={setTolerations}
          />

          {/* Advanced Sandbox & Execution Settings */}
          <div className="space-y-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 p-4">
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              Isolation & Execution Deadlines
            </h4>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <label
                    htmlFor="runtime-class"
                    className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                  >
                    Kubernetes RuntimeClass
                  </label>
                  <HelpTooltip
                    text="Optional container runtime sandbox handler (e.g. gvisor, runsc, kata) for kernel-level multi-tenant isolation."
                    label="Help for runtime class"
                  />
                </div>
                <input
                  id="runtime-class"
                  type="text"
                  placeholder="e.g. gvisor"
                  value={runtimeClassName}
                  onChange={(e) => setRuntimeClassName(e.target.value)}
                  className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
              </div>

              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <label
                    htmlFor="active-deadline"
                    className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                  >
                    Active Deadline Seconds
                  </label>
                  <HelpTooltip
                    text="Maximum wall-clock execution duration in seconds before the Kubernetes Job controller terminates the pod."
                    label="Help for active deadline seconds"
                  />
                </div>
                <input
                  id="active-deadline"
                  type="number"
                  min={60}
                  step={60}
                  placeholder="3600"
                  value={activeDeadlineSeconds ?? ''}
                  onChange={(e) =>
                    setActiveDeadlineSeconds(
                      e.target.value ? Number(e.target.value) : undefined
                    )
                  }
                  className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-xs font-mono text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
              </div>
            </div>
          </div>

          {/* Toggleable Live JSON Payload Preview */}
          <div className="space-y-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setShowPayloadPreview((prev) => !prev)}
              className="gap-2 text-xs border-slate-200 dark:border-slate-800"
            >
              {showPayloadPreview ? (
                <>
                  <Eye className="w-3.5 h-3.5" />
                  <span>Hide Kubernetes JSON Payload</span>
                </>
              ) : (
                <>
                  <Code2 className="w-3.5 h-3.5" />
                  <span>Inspect Generated Kubernetes JSON</span>
                </>
              )}
            </Button>

            {showPayloadPreview && (
              <ProfilePayloadPreview payload={payloadToSubmit} />
            )}
          </div>

          <DialogFooter className="gap-2 sm:gap-0 pt-2 border-t border-slate-200 dark:border-slate-800">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!isFormValid || isSubmitting}
              className="gap-2"
            >
              {isSubmitting && <Loader2 className="w-4 h-4 animate-spin" />}
              <span>{isEditing ? 'Save Changes' : 'Create Profile'}</span>
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
