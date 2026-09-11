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
  Eye,
  Plus,
  Terminal,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TriggerRunDialog } from '@/components/dialogs/TriggerRunDialog'
import { AttachConfigDialog } from '@/components/dialogs/AttachConfigDialog'
import { UploadBuildDialog } from '@/components/dialogs/UploadBuildDialog'
import { BuildLogViewer } from '@/components/build/BuildLogViewer'
import {
  useSuiteConfigs,
  useDeleteSuiteConfig,
  useSuiteArtifacts,
  useSuiteRuns,
} from '@/hooks/use-suites'
import { useBuildEvents } from '@/hooks/use-events'
import type { TestSuite, SuiteConfiguration } from '@/types/suite'

export interface SuiteDetailViewProps {
  suite: TestSuite
  onBack: () => void
}

export const SuiteDetailView: React.FC<SuiteDetailViewProps> = ({ suite, onBack }) => {
  const [activeTab, setActiveTab] = useState('configs')
  const [isTriggerRunOpen, setIsTriggerRunOpen] = useState(false)
  const [isUploadBuildOpen, setIsUploadBuildOpen] = useState(false)
  const [isAttachConfigOpen, setIsAttachConfigOpen] = useState(false)
  const [selectedYamlConfig, setSelectedYamlConfig] = useState<SuiteConfiguration | null>(null)
  const [expandedArtifactLogs, setExpandedArtifactLogs] = useState<Record<string, boolean>>({})

  // Subscribe to live SSE build status changes for reactive artifact updates
  useBuildEvents(suite.id)

  const { data: configs = [], isLoading: isLoadingConfigs } = useSuiteConfigs(suite.id)
  const deleteConfigMutation = useDeleteSuiteConfig(suite.id)

  const { data: artifacts = [], isLoading: isLoadingArtifacts } = useSuiteArtifacts(suite.id)
  const { data: runs = [], isLoading: isLoadingRuns } = useSuiteRuns(suite.id)

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
              onClick={() => setIsAttachConfigOpen(true)}
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
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Scenario Configurations
                </h2>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  Traffic profiles, virtual user targets, and stage progression rules.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsAttachConfigOpen(true)}
                className="gap-1.5 min-h-[36px]"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>New Config</span>
              </Button>
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
                  onClick={() => setIsAttachConfigOpen(true)}
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
                        onClick={() => setSelectedYamlConfig(cfg)}
                        className="gap-1 text-slate-600 dark:text-slate-400 min-h-[36px]"
                        aria-label={`View YAML for ${cfg.name}`}
                      >
                        <Eye className="w-4 h-4" />
                        <span>View YAML</span>
                      </Button>
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
                  Upload a Go load-test scenario package (.tar.gz) to compile executable binaries.
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
                            <Badge
                              variant={
                                art.status === 'READY'
                                  ? 'success'
                                  : art.status === 'BUILDING'
                                  ? 'info'
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

                        <div className="flex items-center gap-3">
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
                          <div className="text-xs text-slate-400 font-mono">
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
                    className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-800/30"
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

                    <div className="text-xs text-slate-400 font-mono">
                      {new Date(r.createdAt).toLocaleDateString()}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </TabsContent>
      </Tabs>

      {/* YAML Viewer Modal */}
      {selectedYamlConfig && (
        <div
          role="dialog"
          aria-modal="true"
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/60 backdrop-blur-xs"
        >
          <div className="w-full max-w-lg rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-6 shadow-xl">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-white">
                  {selectedYamlConfig.name}
                </h3>
                <p className="text-xs text-slate-500 font-mono">
                  {selectedYamlConfig.id}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSelectedYamlConfig(null)}
                className="min-h-[36px]"
              >
                Close
              </Button>
            </div>
            <pre className="p-4 rounded-xl bg-slate-950 text-slate-100 font-mono text-xs overflow-x-auto max-h-80">
              {selectedYamlConfig.contentYaml}
            </pre>
          </div>
        </div>
      )}

      {/* Child Action Dialogs */}
      <TriggerRunDialog open={isTriggerRunOpen} onOpenChange={setIsTriggerRunOpen} />
      <UploadBuildDialog
        suiteId={suite.id}
        open={isUploadBuildOpen}
        onOpenChange={setIsUploadBuildOpen}
      />
      <AttachConfigDialog
        suiteId={suite.id}
        open={isAttachConfigOpen}
        onOpenChange={setIsAttachConfigOpen}
      />
    </div>
  )
}
