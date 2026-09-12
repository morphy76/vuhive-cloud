import React, { useState } from 'react'
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
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
  useCancelSuiteBuild,
  useRetrySuiteBuild,
  useDeleteSuiteArtifact,
} from '@/hooks/use-suites'
import { useToast } from '@/hooks/use-toast'
import { DeleteSuiteDialog } from '@/components/dialogs/DeleteSuiteDialog'
import { DeleteArtifactDialog } from '@/components/dialogs/DeleteArtifactDialog'
import { useBuildEvents } from '@/hooks/use-events'
import type { TestSuite, SuiteConfiguration, HistoricalRun, CompiledArtifact } from '@/types/suite'
import { cn } from '@/lib/utils'

export interface SuiteDetailViewProps {
  suite: TestSuite
  onBack: () => void
}

export const SuiteDetailView: React.FC<SuiteDetailViewProps> = ({ suite, onBack }) => {
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

  // Subscribe to live SSE build status changes for reactive artifact updates
  useBuildEvents(suite.id)

  const { data: configs = [], isLoading: isLoadingConfigs } = useSuiteConfigs(suite.id)
  const deleteConfigMutation = useDeleteSuiteConfig(suite.id)

  const { data: artifacts = [], isLoading: isLoadingArtifacts } = useSuiteArtifacts(suite.id)
  const { data: runs = [], isLoading: isLoadingRuns } = useSuiteRuns(suite.id)
  const deleteSuiteMutation = useDeleteSuite()
  const cancelBuildMutation = useCancelSuiteBuild(suite.id)
  const retryBuildMutation = useRetrySuiteBuild(suite.id)
  const deleteArtifactMutation = useDeleteSuiteArtifact(suite.id)
  const { toast } = useToast()

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
    suite.buildStatus === 'BUILDING' || artifacts.some((a) => a.status === 'BUILDING')
  const hasActiveRuns = runs.some((r) => r.status === 'RUNNING' || r.status === 'QUEUED')

  const handleDeleteSuite = async () => {
    try {
      await deleteSuiteMutation.mutateAsync(suite.id)
      toast({
        title: 'Test Suite Deleted',
        description: `Suite "${suite.name}" was permanently deleted.`,
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
        <Button
          variant="outline"
          onClick={onBack}
          className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Back to Suites</span>
        </Button>

        <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400">
                <Layers className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
                  {suite.name}
                </h1>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs font-mono px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                    {suite.id}
                  </span>
                  <Badge variant={suite.state === 'ACTIVE' ? 'success' : 'default'}>
                    {suite.state}
                  </Badge>
                </div>
              </div>
            </div>
            {suite.description && (
              <p className="text-sm text-slate-600 dark:text-slate-400 mt-2 max-w-2xl">
                {suite.description}
              </p>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
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
            <Button
              variant="outline"
              onClick={() => setIsUploadBuildOpen(true)}
              className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
            >
              <Upload className="w-4 h-4" />
              <span>Upload Build</span>
            </Button>
            <Button
              onClick={() => setIsTriggerRunOpen(true)}
              className="min-h-[44px] gap-2"
            >
              <Play className="w-4 h-4 fill-current" />
              <span>Trigger Run</span>
            </Button>
            <Button
              variant="outline"
              onClick={() => setIsDeleteDialogOpen(true)}
              className="min-h-[44px] gap-2 border-rose-200 dark:border-rose-900/60 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 hover:border-rose-300"
              aria-label={`Delete suite ${suite.name}`}
            >
              <Trash2 className="w-4 h-4" />
              <span>Delete Suite</span>
            </Button>
          </div>
        </div>
      </div>

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
                )}
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
                      <div className="text-xs text-slate-400 font-mono mt-1">
                        {cfg.id} • Attached {new Date(cfg.createdAt).toLocaleDateString()}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
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
                      {configs.length >= 2 && (
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
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteConfig(cfg.id)}
                        className="text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/40 min-h-[36px]"
                        aria-label={`Delete configuration ${cfg.name}`}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
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
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsUploadBuildOpen(true)}
                className="gap-1.5 min-h-[36px]"
              >
                <Upload className="w-3.5 h-3.5" />
                <span>Upload Source</span>
              </Button>
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
                          )}
                          {(art.status === 'FAILED' || art.status === 'CANCELLED') && (
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
                          )}
                          {hasLogs && (
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
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => {
                              setSelectedArtifactForDelete(art)
                              setIsDeleteArtifactOpen(true)
                            }}
                            disabled={art.status === 'BUILDING' || art.status === 'PENDING'}
                            className="text-red-500 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/40 min-h-[36px]"
                            aria-label={`Delete artifact ${art.id}`}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
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
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsTriggerRunOpen(true)}
                className="gap-1.5 min-h-[36px]"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>Execute Run</span>
              </Button>
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
                            <span>Error: {r.metrics.errorRatePct}%</span>
                          )}
                        </div>
                      )}
                    </div>

                    <div className="flex items-center gap-3">
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
        initialSuiteId={suite.id}
      />
      <UploadBuildDialog
        suiteId={suite.id}
        open={isUploadBuildOpen}
        onOpenChange={setIsUploadBuildOpen}
      />
      <ConfigEditorDialog
        suiteId={suite.id}
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
        suite={suite}
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
    </div>
  )
}

