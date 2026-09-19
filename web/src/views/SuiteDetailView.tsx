import React, { useState, useEffect } from 'react'
import {
  ArrowLeft,
  Play,
  Upload,
  FileCode,
  AlertCircle,
  Clock,
  Layers,
  Trash2,
  Plus,
  Terminal,
  Split,
  FileEdit,
  RotateCw,
  Ban,
  CheckCircle2,
  Archive,
  KeyRound,
  ShieldCheck,
  Copy,
  Check,
  Edit2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TriggerRunDialog } from '@/components/dialogs/TriggerRunDialog'
import { ConfigEditorDialog } from '@/components/dialogs/ConfigEditorDialog'
import { ConfigDiffDialog } from '@/components/dialogs/ConfigDiffDialog'
import { UploadBuildDialog } from '@/components/dialogs/UploadBuildDialog'
import { RunSummaryDialog } from '@/components/dialogs/RunSummaryDialog'
import { BuildLogViewer } from '@/components/build/BuildLogViewer'
import {
  useSuiteConfigs,
  useDeleteSuiteConfig,
  useSuiteArtifacts,
  useSuiteRuns,
  useDeleteSuite,
  useUpdateSuite,
  useCancelSuiteBuild,
  useRetrySuiteBuild,
  useDeleteSuiteArtifact,
} from '@/hooks/use-suites'
import { useToast } from '@/hooks/use-toast'
import { DeleteSuiteDialog } from '@/components/dialogs/DeleteSuiteDialog'
import { DeleteArtifactDialog } from '@/components/dialogs/DeleteArtifactDialog'
import { useBuildEvents } from '@/hooks/use-events'
import { formatErrorRate } from '@/lib/format-utils'
import { CreateSecretDialog } from '@/components/dialogs/CreateSecretDialog'
import { EditSecretDialog } from '@/components/dialogs/EditSecretDialog'
import { DeleteSecretDialog } from '@/components/dialogs/DeleteSecretDialog'
import { useSuiteSecrets, useDeleteSuiteSecret } from '@/hooks/use-secrets'
import type { SuiteSecret } from '@/types/secret'
import type { TestSuite, SuiteConfiguration, HistoricalRun, CompiledArtifact } from '@/types/suite'
import { cn } from '@/lib/utils'

export interface SuiteDetailViewProps {
  suite: TestSuite
  onBack: () => void
}

export const SuiteDetailView: React.FC<SuiteDetailViewProps> = ({ suite, onBack }) => {
  const [currentSuite, setCurrentSuite] = useState<TestSuite>(suite)
  const [activeTab, setActiveTab] = useState('configs')
  const [isTriggerRunOpen, setIsTriggerRunOpen] = useState(false)
  const [isUploadBuildOpen, setIsUploadBuildOpen] = useState(false)
  const [isConfigEditorOpen, setIsConfigEditorOpen] = useState(false)
  const [editingConfig, setEditingConfig] = useState<SuiteConfiguration | null>(null)
  const [isDiffDialogOpen, setIsDiffDialogOpen] = useState(false)
  const [diffBaseId, setDiffBaseId] = useState<string | undefined>(undefined)
  const [diffComparisonId, setDiffComparisonId] = useState<string | undefined>(undefined)
  const [expandedArtifactLogs, setExpandedArtifactLogs] = useState<Record<string, boolean>>({})
  const [selectedRunForSummary, setSelectedRunForSummary] = useState<HistoricalRun | null>(null)
  const [isRunSummaryOpen, setIsRunSummaryOpen] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [selectedArtifactForDelete, setSelectedArtifactForDelete] = useState<CompiledArtifact | null>(null)
  const [isDeleteArtifactOpen, setIsDeleteArtifactOpen] = useState(false)
  const { data: secrets = [], isError: isSecretsError, error: secretsError } = useSuiteSecrets(currentSuite.id)
  const isSecretsDisabled = Boolean(
    isSecretsError && (
      (secretsError as any)?.status === 501 ||
      String((secretsError as any)?.message || '').toLowerCase().includes('secrets_encryption_key') ||
      String((secretsError as any)?.message || '').toLowerCase().includes('disabled')
    )
  )
  const deleteSecretMutation = useDeleteSuiteSecret(currentSuite.id)
  const [isCreateSecretOpen, setIsCreateSecretOpen] = useState(false)
  const [selectedSecretForEdit, setSelectedSecretForEdit] = useState<SuiteSecret | null>(null)
  const [isEditSecretOpen, setIsEditSecretOpen] = useState(false)
  const [selectedSecretForDelete, setSelectedSecretForDelete] = useState<SuiteSecret | null>(null)
  const [isDeleteSecretOpen, setIsDeleteSecretOpen] = useState(false)
  const [copiedSecretKey, setCopiedSecretKey] = useState<string | null>(null)

  const handleCopyPlaceholder = (key: string) => {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard.writeText(`\${secrets.${key}}`)
    }
    setCopiedSecretKey(key)
    toast({
      title: 'Placeholder Copied',
      description: `\${secrets.${key}} copied to clipboard.`,
    })
    setTimeout(() => setCopiedSecretKey(null), 2000)
  }

  const handleConfirmDeleteSecret = async () => {
    if (!selectedSecretForDelete) return
    try {
      await deleteSecretMutation.mutateAsync(selectedSecretForDelete.id)
      toast({
        title: 'Secret Deleted',
        description: `Secret "${selectedSecretForDelete.key}" was permanently removed.`,
      })
      setIsDeleteSecretOpen(false)
      setSelectedSecretForDelete(null)
    } catch (err: any) {
      toast({
        title: 'Failed to delete secret',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }


  const { toast } = useToast()
  const updateSuiteMutation = useUpdateSuite(currentSuite.id)

  useEffect(() => {
    setCurrentSuite(suite)
  }, [suite])

  // Subscribe to live SSE build status changes for reactive artifact updates
  useBuildEvents(currentSuite.id, (event) => {
    if (event.status === 'READY' && currentSuite.state === 'DRAFT') {
      toast({
        title: 'Build Artifact Ready',
        description: `Artifact ${event.artifact_id} (${event.platform}) compiled successfully. You can now activate the suite.`,
      })
    }
  })

  const handleActivateSuite = async () => {
    try {
      const updated = await updateSuiteMutation.mutateAsync({
        name: currentSuite.name,
        description: currentSuite.description,
        state: 'ACTIVE',
      })
      setCurrentSuite(updated)
      toast({
        title: 'Test Suite Activated',
        description: `Suite "${updated.name}" is now ACTIVE and ready for execution.`,
      })
    } catch (err: any) {
      toast({
        title: 'Failed to activate test suite',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  const handleStateTransition = async (targetState: 'ACTIVE' | 'DRAFT' | 'ARCHIVED') => {
    try {
      const updated = await updateSuiteMutation.mutateAsync({
        name: currentSuite.name,
        description: currentSuite.description,
        state: targetState,
      })
      setCurrentSuite(updated)
      toast({
        title: `Suite Status Updated: ${targetState}`,
        description: `Suite "${updated.name}" transitioned to ${targetState}.`,
      })
    } catch (err: any) {
      toast({
        title: 'Failed to update suite status',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  const { data: configs = [], isLoading: isLoadingConfigs } = useSuiteConfigs(currentSuite.id)
  const deleteConfigMutation = useDeleteSuiteConfig(currentSuite.id)

  const { data: artifacts = [], isLoading: isLoadingArtifacts } = useSuiteArtifacts(currentSuite.id)
  const { data: runs = [], isLoading: isLoadingRuns } = useSuiteRuns(currentSuite.id)
  const deleteSuiteMutation = useDeleteSuite()
  const cancelBuildMutation = useCancelSuiteBuild(currentSuite.id)
  const retryBuildMutation = useRetrySuiteBuild(currentSuite.id)
  const deleteArtifactMutation = useDeleteSuiteArtifact(currentSuite.id)

  const handleCancelBuild = async (artifactId: string) => {
    try {
      await cancelBuildMutation.mutateAsync({ artifactId, reason: 'Cancelled from Artifacts view' })
      toast({
        title: 'Build Cancelled',
        description: `Build artifact ${artifactId} was cancelled.`,
      })
    } catch (err: any) {
      toast({
        title: 'Failed to cancel build',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  const handleRetryBuild = async (artifactId: string) => {
    try {
      await retryBuildMutation.mutateAsync(artifactId)
      toast({
        title: 'Build Retried',
        description: `Compilation job restarted for artifact ${artifactId}.`,
      })
    } catch (err: any) {
      toast({
        title: 'Failed to retry build',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  const handleConfirmDeleteArtifact = async () => {
    if (!selectedArtifactForDelete) return
    try {
      await deleteArtifactMutation.mutateAsync(selectedArtifactForDelete.id)
      toast({
        title: 'Artifact Deleted',
        description: `Artifact ${selectedArtifactForDelete.id} was permanently deleted.`,
      })
      setIsDeleteArtifactOpen(false)
      setSelectedArtifactForDelete(null)
    } catch (err: any) {
      toast({
        title: 'Failed to delete artifact',
        description: err.message || 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  const hasActiveBuilds =
    currentSuite.buildStatus === 'BUILDING' || artifacts.some((a) => a.status === 'BUILDING')
  const hasActiveRuns = runs.some((r) => r.status === 'RUNNING' || r.status === 'QUEUED')

  const handleDeleteSuite = async () => {
    try {
      await deleteSuiteMutation.mutateAsync(currentSuite.id)
      toast({
        title: 'Test Suite Deleted',
        description: `Suite "${currentSuite.name}" was permanently deleted.`,
      })
      setIsDeleteDialogOpen(false)
      onBack()
    } catch (err: any) {
      toast({
        title: 'Failed to delete test suite',
        description: err.message || 'An unexpected error occurred while deleting the suite.',
        variant: 'destructive',
      })
    }
  }

  const handleDeleteConfig = async (configId: string) => {
    if (confirm('Are you sure you want to delete this configuration?')) {
      try {
        await deleteConfigMutation.mutateAsync(configId)
      } catch (err) {
        console.error('Failed deleting config:', err)
      }
    }
  }

  return (
    <div className="space-y-6">
      {/* Header with Navigation and Quick Actions */}
      <div className="space-y-4">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="outline"
              onClick={onBack}
              className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400"
            >
              <ArrowLeft className="w-4 h-4" />
              <span>Back to Suites</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>Back to test suites catalog</TooltipContent>
        </Tooltip>

        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400">
                <Layers className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
                  {currentSuite.name}
                </h1>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs font-mono px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                    {currentSuite.id}
                  </span>
                  <Badge
                    variant={
                      currentSuite.state === 'ACTIVE'
                        ? 'success'
                        : currentSuite.state === 'ARCHIVED'
                        ? 'warning'
                        : 'default'
                    }
                  >
                    {currentSuite.state}
                  </Badge>
                </div>
              </div>
            </div>
            {currentSuite.description && (
              <p className="text-sm text-slate-600 dark:text-slate-400 mt-2 max-w-2xl">
                {currentSuite.description}
              </p>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            {currentSuite.state === 'DRAFT' && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    onClick={handleActivateSuite}
                    disabled={updateSuiteMutation.isPending}
                    className="min-h-[44px] gap-2 bg-emerald-600 hover:bg-emerald-700 text-white"
                    aria-label={`Activate suite ${currentSuite.name}`}
                  >
                    <CheckCircle2 className="w-4 h-4" />
                    <span>{updateSuiteMutation.isPending ? 'Activating...' : 'Activate Suite'}</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Activate test suite to enable execution</TooltipContent>
              </Tooltip>
            )}

            {currentSuite.state === 'ACTIVE' && (
              <>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      onClick={() => handleStateTransition('DRAFT')}
                      disabled={updateSuiteMutation.isPending}
                      className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400"
                      aria-label={`Deactivate suite ${currentSuite.name}`}
                    >
                      <FileEdit className="w-4 h-4" />
                      <span>Deactivate</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Set test suite to DRAFT</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      onClick={() => handleStateTransition('ARCHIVED')}
                      disabled={updateSuiteMutation.isPending}
                      className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400"
                      aria-label={`Archive suite ${currentSuite.name}`}
                    >
                      <Archive className="w-4 h-4" />
                      <span>Archive</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Archive test suite</TooltipContent>
                </Tooltip>
              </>
            )}

            {currentSuite.state === 'ARCHIVED' && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    onClick={handleActivateSuite}
                    disabled={updateSuiteMutation.isPending}
                    className="min-h-[44px] gap-2 bg-emerald-600 hover:bg-emerald-700 text-white"
                    aria-label={`Re-activate suite ${currentSuite.name}`}
                  >
                    <CheckCircle2 className="w-4 h-4" />
                    <span>{updateSuiteMutation.isPending ? 'Activating...' : 'Re-activate Suite'}</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Re-activate archived test suite</TooltipContent>
              </Tooltip>
            )}

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  onClick={() => {
                    setEditingConfig(null)
                    setIsConfigEditorOpen(true)
                  }}
                  className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
                >
                  <FileCode className="w-4 h-4" />
                  <span>Attach Config</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Attach scenario configuration</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  onClick={() => setIsUploadBuildOpen(true)}
                  className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
                >
                  <Upload className="w-4 h-4" />
                  <span>Upload Build</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Upload load test source package</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  onClick={() => setIsTriggerRunOpen(true)}
                  className="min-h-[44px] gap-2"
                >
                  <Play className="w-4 h-4 fill-current" />
                  <span>Trigger Run</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Trigger load test execution</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  onClick={() => setIsDeleteDialogOpen(true)}
                  className="min-h-[44px] gap-2 border-rose-200 dark:border-rose-900/60 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 hover:border-rose-300"
                  aria-label={`Delete suite ${currentSuite.name}`}
                >
                  <Trash2 className="w-4 h-4" />
                  <span>Delete Suite</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Delete suite {currentSuite.name}</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </div>

      {/* Informative Lifecycle State Banners */}
      {currentSuite.state === 'DRAFT' && (
        <div
          data-testid="draft-suite-banner"
          className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 rounded-xl border border-amber-200 dark:border-amber-900/60 bg-amber-50/70 dark:bg-amber-950/30 text-amber-900 dark:text-amber-200 shadow-2xs"
        >
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" />
            <div className="space-y-1 text-xs sm:text-sm">
              <p className="font-semibold text-amber-900 dark:text-amber-100">
                This test suite is currently in DRAFT state.
              </p>
              <p className="text-amber-700 dark:text-amber-300">
                Attach a scenario configuration and upload a build package, then activate the suite to enable test execution and scheduled runs.
              </p>
            </div>
          </div>
          <Button
            size="sm"
            onClick={handleActivateSuite}
            disabled={updateSuiteMutation.isPending}
            className="min-h-[38px] sm:min-h-[40px] gap-1.5 shrink-0 bg-amber-600 hover:bg-amber-700 text-white font-medium"
            aria-label="Activate Suite from banner"
          >
            <CheckCircle2 className="w-4 h-4" />
            <span>{updateSuiteMutation.isPending ? 'Activating...' : 'Activate Suite'}</span>
          </Button>
        </div>
      )}

      {currentSuite.state === 'ARCHIVED' && (
        <div
          data-testid="archived-suite-banner"
          className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-100/70 dark:bg-slate-800/40 text-slate-800 dark:text-slate-200 shadow-2xs"
        >
          <div className="flex items-start gap-3">
            <Archive className="w-5 h-5 text-slate-500 mt-0.5 shrink-0" />
            <div className="space-y-1 text-xs sm:text-sm">
              <p className="font-semibold text-slate-900 dark:text-white">
                This test suite is ARCHIVED.
              </p>
              <p className="text-slate-600 dark:text-slate-400">
                Execution is disabled for archived suites. Re-activate the suite to enable running load tests.
              </p>
            </div>
          </div>
          <Button
            size="sm"
            onClick={handleActivateSuite}
            disabled={updateSuiteMutation.isPending}
            className="min-h-[38px] sm:min-h-[40px] gap-1.5 shrink-0 bg-brand-600 hover:bg-brand-700 text-white font-medium"
            aria-label="Re-activate Suite from banner"
          >
            <CheckCircle2 className="w-4 h-4" />
            <span>{updateSuiteMutation.isPending ? 'Activating...' : 'Re-activate Suite'}</span>
          </Button>
        </div>
      )}

      {/* Tabbed Navigation */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="bg-slate-100 dark:bg-slate-800/80 p-1 rounded-xl">
          <TabsTrigger value="configs" className="gap-2 min-h-[40px] px-4">
            <FileCode className="w-4 h-4" />
            <span>Configurations ({configs.length})</span>
          </TabsTrigger>
          <TabsTrigger value="artifacts" className="gap-2 min-h-[40px] px-4">
            <Layers className="w-4 h-4" />
            <span>Artifacts ({artifacts.length})</span>
          </TabsTrigger>
          <TabsTrigger value="runs" className="gap-2 min-h-[40px] px-4">
            <Play className="w-4 h-4" />
            <span>Historical Runs ({runs.length})</span>
          </TabsTrigger>
          <TabsTrigger value="secrets" className="gap-2 min-h-[40px] px-4">
            <KeyRound className="w-4 h-4" />
            <span>Secrets ({secrets.length})</span>
          </TabsTrigger>
        </TabsList>

        {/* Configurations Tab */}
        <TabsContent value="configs">
          <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xs">
            <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Scenario Configurations
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Traffic profiles, virtual user targets, and stage progression rules.
                </p>
              </div>
              <div className="flex items-center gap-2">
                {configs.length >= 2 && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setDiffBaseId(configs[0].id)
                          setDiffComparisonId(configs[1].id)
                          setIsDiffDialogOpen(true)
                        }}
                        className="gap-1.5 min-h-[36px]"
                      >
                        <Split className="w-3.5 h-3.5" />
                        <span>Compare Versions</span>
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Compare scenario configurations</TooltipContent>
                  </Tooltip>
                )}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditingConfig(null)
                        setIsConfigEditorOpen(true)
                      }}
                      className="gap-1.5 min-h-[36px]"
                    >
                      <Plus className="w-3.5 h-3.5" />
                      <span>New Config</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Attach new scenario configuration</TooltipContent>
                </Tooltip>
              </div>
            </div>

            {isLoadingConfigs ? (
              <div className="py-8 text-center text-sm text-slate-500">Loading configurations...</div>
            ) : configs.length === 0 ? (
              <div className="py-12 text-center">
                <FileCode className="w-10 h-10 mx-auto text-slate-300 dark:text-slate-700 mb-2" />
                <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                  No Configurations Attached
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm mx-auto">
                  Attach scenario parameters or ramp-up profiles to customize virtual load generation.
                </p>
                <Button
                  variant="outline"
                  onClick={() => {
                    setEditingConfig(null)
                    setIsConfigEditorOpen(true)
                  }}
                  className="mt-4 min-h-[40px]"
                >
                  Attach Configuration
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                {configs.map((cfg) => (
                  <div
                    key={cfg.id}
                    className="flex items-center justify-between p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm text-slate-900 dark:text-white">
                          {cfg.name}
                        </span>
                        {cfg.isDefault && <Badge variant="info">Default</Badge>}
                      </div>
                      {cfg.description && (
                        <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                          {cfg.description}
                        </p>
                      )}
                      <div className="text-xs text-slate-400 font-mono mt-1">
                        {cfg.id} • Attached {new Date(cfg.createdAt).toLocaleDateString()}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setEditingConfig(cfg)
                              setIsConfigEditorOpen(true)
                            }}
                            className="gap-1 text-slate-600 dark:text-slate-400 min-h-[36px]"
                            aria-label={`Edit and view YAML for ${cfg.name}`}
                          >
                            <FileEdit className="w-4 h-4" />
                            <span>Edit / Tune</span>
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Edit configuration</TooltipContent>
                      </Tooltip>

                      {configs.length >= 2 && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setDiffBaseId(cfg.id)
                                const other = configs.find((c) => c.id !== cfg.id)
                                setDiffComparisonId(other?.id)
                                setIsDiffDialogOpen(true)
                              }}
                              className="gap-1 text-slate-600 dark:text-slate-400 min-h-[36px]"
                              aria-label={`Compare configuration ${cfg.name}`}
                            >
                              <Split className="w-4 h-4" />
                              <span>Diff</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Compare configuration</TooltipContent>
                        </Tooltip>
                      )}

                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteConfig(cfg.id)}
                            className="text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/40 min-h-[36px] min-w-[36px] p-0 flex items-center justify-center"
                            aria-label={`Delete configuration ${cfg.name}`}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Delete configuration</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </TabsContent>

        {/* Compiled Artifacts Tab */}
        <TabsContent value="artifacts">
          <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xs">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Compiled Scenario Binaries
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Cross-compiled Go binaries packaged in S3 object store for container execution.
                </p>
              </div>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setIsUploadBuildOpen(true)}
                    className="gap-1.5 min-h-[36px]"
                  >
                    <Upload className="w-3.5 h-3.5" />
                    <span>Upload Source</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Upload scenario source package</TooltipContent>
              </Tooltip>
            </div>

            {isLoadingArtifacts ? (
              <div className="py-8 text-center text-sm text-slate-500">Loading artifacts...</div>
            ) : artifacts.length === 0 ? (
              <div className="py-12 text-center">
                <Layers className="w-10 h-10 mx-auto text-slate-300 dark:text-slate-700 mb-2" />
                <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                  No Compiled Artifacts Found
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm mx-auto">
                  Upload a Go load-test scenario package (.tar.gz, .tar.bz2, or .zip) to compile executable binaries.
                </p>
                <Button
                  variant="outline"
                  onClick={() => setIsUploadBuildOpen(true)}
                  className="mt-4 min-h-[40px]"
                >
                  Upload Source Package
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                {artifacts.map((art) => {
                  const hasLogs = Boolean(art.errorMessage || art.buildLogsS3Key || art.status === 'FAILED')
                  const isLogExpanded = Boolean(expandedArtifactLogs[art.id])

                  return (
                    <div
                      key={art.id}
                      className="p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30 space-y-3"
                    >
                      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="px-2 py-0.5 rounded text-xs font-mono font-semibold bg-brand-50 dark:bg-brand-950 text-brand-700 dark:text-brand-300">
                              {art.platform}
                            </span>
                            <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
                              {art.id}
                            </span>
                            <Badge
                              variant={
                                art.status === 'READY'
                                  ? 'success'
                                  : art.status === 'BUILDING' || art.status === 'PENDING'
                                  ? 'info'
                                  : art.status === 'CANCELLED'
                                  ? 'warning'
                                  : 'error'
                              }
                            >
                              {art.status}
                            </Badge>
                          </div>
                          {art.sha256Checksum && (
                            <div className="text-xs text-slate-500 font-mono mt-1 truncate max-w-md">
                              SHA256: {art.sha256Checksum}
                            </div>
                          )}
                          {art.errorMessage && (
                            <div className="text-xs text-red-500 mt-1 flex items-center gap-1">
                              <AlertCircle className="w-3.5 h-3.5" />
                              <span>{art.errorMessage}</span>
                            </div>
                          )}
                        </div>

                        <div className="flex items-center gap-2">
                          {(art.status === 'BUILDING' || art.status === 'PENDING') && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleCancelBuild(art.id)}
                                  disabled={cancelBuildMutation.isPending}
                                  className="gap-1 min-h-[36px] text-xs text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 border-rose-200 dark:border-rose-900/60"
                                  aria-label={`Cancel build for ${art.id}`}
                                >
                                  <Ban className="w-3.5 h-3.5" />
                                  <span>Cancel</span>
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Cancel build</TooltipContent>
                            </Tooltip>
                          )}
                          {(art.status === 'FAILED' || art.status === 'CANCELLED') && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleRetryBuild(art.id)}
                                  disabled={retryBuildMutation.isPending}
                                  className="gap-1 min-h-[36px] text-xs text-slate-700 dark:text-slate-300"
                                  aria-label={`Retry build for ${art.id}`}
                                >
                                  <RotateCw className={cn('w-3.5 h-3.5', retryBuildMutation.isPending && 'animate-spin')} />
                                  <span>Retry</span>
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Retry build</TooltipContent>
                            </Tooltip>
                          )}
                          {hasLogs && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() =>
                                    setExpandedArtifactLogs((prev) => ({
                                      ...prev,
                                      [art.id]: !prev[art.id],
                                    }))
                                  }
                                  className="gap-1.5 min-h-[36px] text-xs"
                                >
                                  <Terminal className="w-3.5 h-3.5" />
                                  <span>{isLogExpanded ? 'Hide Logs' : 'View Logs'}</span>
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>{isLogExpanded ? 'Hide build logs' : 'View build logs'}</TooltipContent>
                            </Tooltip>
                          )}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                  setSelectedArtifactForDelete(art)
                                  setIsDeleteArtifactOpen(true)
                                }}
                                disabled={art.status === 'BUILDING' || art.status === 'PENDING'}
                                className="text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/40 min-h-[36px] min-w-[36px] p-0 flex items-center justify-center"
                                aria-label={`Delete artifact ${art.id}`}
                              >
                                <Trash2 className="w-4 h-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Delete artifact</TooltipContent>
                          </Tooltip>
                          <div className="text-xs text-slate-400 font-mono hidden sm:block">
                            {new Date(art.createdAt).toLocaleDateString()}
                          </div>
                        </div>
                      </div>

                      {hasLogs && isLogExpanded && (
                        <BuildLogViewer
                          logs={art.errorMessage || 'No additional log details available.'}
                          title={`Build Logs (${art.platform})`}
                          defaultExpanded={true}
                        />
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </TabsContent>

        {/* Historical Runs Tab */}
        <TabsContent value="runs">
          <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xs">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Historical Executions
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Distributed load runs and indexed performance telemetry for this scenario.
                </p>
              </div>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setIsTriggerRunOpen(true)}
                    className="gap-1.5 min-h-[36px]"
                  >
                    <Play className="w-3.5 h-3.5 fill-current" />
                    <span>Execute Run</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Execute load test run</TooltipContent>
              </Tooltip>
            </div>

            {isLoadingRuns ? (
              <div className="py-8 text-center text-sm text-slate-500">Loading runs...</div>
            ) : runs.length === 0 ? (
              <div className="py-12 text-center">
                <Clock className="w-10 h-10 mx-auto text-slate-300 dark:text-slate-700 mb-2" />
                <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                  No Execution History
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-sm mx-auto">
                  Trigger an ad-hoc test run or configure a Kubernetes CronJob schedule.
                </p>
                <Button
                  variant="outline"
                  onClick={() => setIsTriggerRunOpen(true)}
                  className="mt-4 min-h-[40px]"
                >
                  Execute Test Run
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                {runs.map((r) => (
                  <div
                    key={r.id}
                    onClick={() => {
                      setSelectedRunForSummary(r)
                      setIsRunSummaryOpen(true)
                    }}
                    className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30 hover:bg-slate-100/70 dark:hover:bg-slate-800/60 transition-colors cursor-pointer"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm font-mono text-slate-900 dark:text-white">
                          {r.id}
                        </span>
                        <Badge
                          variant={
                            r.status === 'COMPLETED'
                              ? 'success'
                              : r.status === 'RUNNING'
                              ? 'info'
                              : r.status === 'QUEUED'
                              ? 'default'
                              : 'error'
                          }
                        >
                          {r.status}
                        </Badge>
                        {r.slaPassed !== undefined && (
                          <span
                            className={`px-2 py-0.5 text-[11px] font-semibold rounded ${
                              r.slaPassed
                                ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300'
                                : 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300'
                            }`}
                          >
                            {r.slaPassed ? 'SLA Passed' : 'SLA Failed'}
                          </span>
                        )}
                      </div>
                      {r.metrics && (
                        <div className="flex gap-3 text-xs text-slate-500 dark:text-slate-400 mt-1.5">
                          {r.metrics.avgTps !== undefined && <span>{r.metrics.avgTps} req/s</span>}
                          {r.metrics.p95DurationMs !== undefined && (
                            <span>P95: {r.metrics.p95DurationMs}ms</span>
                          )}
                          {r.metrics.errorRatePct !== undefined && (
                            <span>Error: {formatErrorRate(r.metrics.errorRatePct)}</span>
                          )}
                        </div>
                      )}
                    </div>

                    <div className="flex items-center gap-3">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              setSelectedRunForSummary(r)
                              setIsRunSummaryOpen(true)
                            }}
                            className="text-xs text-brand-600 hover:text-brand-700 dark:text-brand-400 min-h-[36px]"
                            aria-label={`View summary for run ${r.id}`}
                          >
                            View Summary
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>View run summary</TooltipContent>
                      </Tooltip>
                      <div className="text-xs text-slate-400 font-mono">
                        {new Date(r.createdAt).toLocaleDateString()}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </TabsContent>
        {/* Suite Secrets Tab */}
        <TabsContent value="secrets">
          <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xs">
            <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Suite Secrets
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Encrypted credentials, API tokens, and parameters referenced in scenario configurations via <span className="font-mono text-slate-700 dark:text-slate-300 font-medium">${'{secrets.KEY}'}</span>.
                </p>
              </div>
              <Button
                onClick={() => setIsCreateSecretOpen(true)}
                disabled={isSecretsDisabled}
                className="gap-2 min-h-[40px] shadow-sm"
              >
                <Plus className="w-4 h-4" />
                <span>Add Suite Secret</span>
              </Button>
            </div>

            {isSecretsDisabled ? (
              <div className="text-center py-12 px-4 border border-dashed border-amber-200 dark:border-amber-900/60 rounded-xl bg-amber-50/20 dark:bg-amber-950/10">
                <div className="w-12 h-12 rounded-2xl bg-amber-100 dark:bg-amber-950/60 text-amber-600 dark:text-amber-400 flex items-center justify-center mx-auto mb-3 border border-amber-200 dark:border-amber-900/60">
                  <AlertCircle className="w-6 h-6" />
                </div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">
                  Secrets Management Disabled
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 max-w-md mx-auto mb-4 leading-relaxed">
                  Suite secrets management requires <span className="font-mono text-slate-700 dark:text-slate-300 font-medium">SECRETS_ENCRYPTION_KEY</span> configured on the control plane deployment. Configure a 32-byte AES-256-GCM encryption key to enable creating and referencing encrypted suite secrets.
                </p>
                <Button
                  disabled
                  className="gap-1.5 text-xs min-h-[36px]"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Suite Secret</span>
                </Button>
              </div>
            ) : secrets.length === 0 ? (
              <div className="text-center py-12 px-4 border border-dashed border-slate-200 dark:border-slate-800 rounded-xl bg-slate-50/50 dark:bg-slate-800/20">
                <div className="w-12 h-12 rounded-2xl bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400 flex items-center justify-center mx-auto mb-3 border border-brand-200 dark:border-brand-900/60">
                  <ShieldCheck className="w-6 h-6" />
                </div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-1">
                  No suite secrets defined
                </h3>
                <p className="text-xs text-slate-500 dark:text-slate-400 max-w-md mx-auto mb-4 leading-relaxed">
                  Store sensitive credentials securely encrypted with AES-256-GCM. Scenarios can reference them using <span className="font-mono text-slate-700 dark:text-slate-300 font-medium">${'{secrets.KEY}'}</span> placeholders without committing secrets in configuration files.
                </p>
                <Button
                  onClick={() => setIsCreateSecretOpen(true)}
                  className="gap-1.5 text-xs min-h-[36px]"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Suite Secret</span>
                </Button>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-slate-800 text-slate-500 dark:text-slate-400">
                      <th className="pb-3 font-semibold">Key / Placeholder</th>
                      <th className="pb-3 font-semibold">Storage & Security</th>
                      <th className="pb-3 font-semibold">Masked Value</th>
                      <th className="pb-3 font-semibold">Last Updated</th>
                      <th className="pb-3 font-semibold text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800/60">
                    {secrets.map((sec) => (
                      <tr
                        key={sec.id}
                        className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
                      >
                        <td className="py-3.5">
                          <div className="flex items-center gap-2">
                            <span className="font-mono font-bold text-sm text-slate-900 dark:text-white">
                              {sec.key}
                            </span>
                            <button
                              type="button"
                              onClick={() => handleCopyPlaceholder(sec.key)}
                              aria-label={`Copy placeholder \${secrets.${sec.key}}`}
                              className="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                              title="Copy placeholder"
                            >
                              {copiedSecretKey === sec.key ? (
                                <Check className="w-3.5 h-3.5 text-emerald-600" />
                              ) : (
                                <Copy className="w-3.5 h-3.5" />
                              )}
                            </button>
                          </div>
                          <span className="font-mono text-[11px] text-brand-600 dark:text-brand-400 mt-0.5 block">
                            ${`{secrets.${sec.key}}`}
                          </span>
                        </td>
                        <td className="py-3.5">
                          <div className="flex items-center gap-1.5">
                            <Badge variant="outline" className="text-[10px] font-mono border-emerald-300 dark:border-emerald-800 text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-950/40">
                              AES-256-GCM
                            </Badge>
                            <span className="text-[11px] text-slate-500 dark:text-slate-400">PostgreSQL</span>
                          </div>
                        </td>
                        <td className="py-3.5 font-mono text-slate-400 tracking-widest text-xs">
                          ••••••••
                        </td>
                        <td className="py-3.5 text-slate-500 dark:text-slate-400">
                          {sec.updatedAt ? new Date(sec.updatedAt).toLocaleDateString() : 'N/A'}
                        </td>
                        <td className="py-3.5 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setSelectedSecretForEdit(sec)
                                setIsEditSecretOpen(true)
                              }}
                              className="h-8 px-2 text-xs gap-1 text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white"
                              aria-label={`Edit secret ${sec.key}`}
                            >
                              <Edit2 className="w-3.5 h-3.5" />
                              <span>Edit</span>
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setSelectedSecretForDelete(sec)
                                setIsDeleteSecretOpen(true)
                              }}
                              className="h-8 px-2 text-xs gap-1 text-rose-600 hover:text-rose-700 hover:bg-rose-50 dark:hover:bg-rose-950/40"
                              aria-label={`Delete secret ${sec.key}`}
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                              <span>Delete</span>
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </TabsContent>
      </Tabs>

      {/* Action Dialogs */}
      <RunSummaryDialog
        open={isRunSummaryOpen}
        onOpenChange={setIsRunSummaryOpen}
        run={selectedRunForSummary}
      />
      <TriggerRunDialog
        open={isTriggerRunOpen}
        onOpenChange={setIsTriggerRunOpen}
        initialSuiteId={currentSuite.id}
      />
      <UploadBuildDialog
        suiteId={currentSuite.id}
        open={isUploadBuildOpen}
        onOpenChange={setIsUploadBuildOpen}
      />
      <ConfigEditorDialog
        suiteId={currentSuite.id}
        open={isConfigEditorOpen}
        onOpenChange={setIsConfigEditorOpen}
        initialConfig={editingConfig}
        existingConfigs={configs}
      />
      <ConfigDiffDialog
        open={isDiffDialogOpen}
        onOpenChange={setIsDiffDialogOpen}
        configs={configs}
        initialBaseId={diffBaseId}
        initialComparisonId={diffComparisonId}
      />
      <DeleteSuiteDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        suite={currentSuite}
        onConfirm={handleDeleteSuite}
        isDeleting={deleteSuiteMutation.isPending}
        hasActiveBuilds={hasActiveBuilds}
        hasActiveRuns={hasActiveRuns}
      />
      <DeleteArtifactDialog
        open={isDeleteArtifactOpen}
        onOpenChange={setIsDeleteArtifactOpen}
        artifact={selectedArtifactForDelete}
        onConfirm={handleConfirmDeleteArtifact}
        isDeleting={deleteArtifactMutation.isPending}
      />
      <CreateSecretDialog
        suiteId={currentSuite.id}
        open={isCreateSecretOpen}
        onOpenChange={setIsCreateSecretOpen}
      />
      <EditSecretDialog
        suiteId={currentSuite.id}
        secret={selectedSecretForEdit}
        open={isEditSecretOpen}
        onOpenChange={setIsEditSecretOpen}
      />
      <DeleteSecretDialog
        open={isDeleteSecretOpen}
        onOpenChange={setIsDeleteSecretOpen}
        secret={selectedSecretForDelete}
        onConfirm={handleConfirmDeleteSecret}
        isDeleting={deleteSecretMutation.isPending}
      />
    </div>
  )
}

