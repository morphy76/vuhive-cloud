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
import { Progress } from '@/components/ui/progress'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { InfoBadge } from '@/components/help/InfoBadge'
import { BuildStatusStepper } from '@/components/build/BuildStatusStepper'
import { AstErrorCallout } from '@/components/build/AstErrorCallout'
import { BuildLogViewer } from '@/components/build/BuildLogViewer'
import { useUploadSuiteBuild, useCancelSuiteBuild, useRetrySuiteBuild } from '@/hooks/use-suites'
import { useBuildEvents, type BuildStatusChangedEvent } from '@/hooks/use-events'
import { UploadCloud, FileArchive, X, AlertCircle, CheckCircle2, RotateCw, Ban } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface UploadBuildDialogProps {
  suiteId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

const MAX_FILE_SIZE = 50 * 1024 * 1024 // 50MB
const ALLOWED_EXTENSIONS = ['.tar.gz', '.tgz', '.tar.bz2', '.tbz2', '.zip', '.tar']

export const UploadBuildDialog: React.FC<UploadBuildDialogProps> = ({
  suiteId,
  open,
  onOpenChange,
}) => {
  const [file, setFile] = React.useState<File | null>(null)
  const [platform, setPlatform] = React.useState<string>('linux/amd64')
  const [allowInsecure, setAllowInsecure] = React.useState(false)
  const [isDragging, setIsDragging] = React.useState(false)
  const [validationError, setValidationError] = React.useState<string | null>(null)
  const [astError, setAstError] = React.useState<string | null>(null)

  // Upload & build lifecycle
  const [isUploading, setIsUploading] = React.useState(false)
  const [uploadProgress, setUploadProgress] = React.useState(0)
  const [activeArtifactId, setActiveArtifactId] = React.useState<string | null>(null)
  const [buildStatus, setBuildStatus] = React.useState<string | null>(null)
  const [buildError, setBuildError] = React.useState<string | null>(null)
  const [compilationLogs, setCompilationLogs] = React.useState<string | null>(null)

  const uploadMutation = useUploadSuiteBuild(suiteId)
  const cancelMutation = useCancelSuiteBuild(suiteId)
  const retryMutation = useRetrySuiteBuild(suiteId)

  // Listen to SSE live build status changes
  useBuildEvents(suiteId, (event: BuildStatusChangedEvent) => {
    if (activeArtifactId && event.artifact_id !== activeArtifactId) {
      return
    }

    setBuildStatus(event.status)
    if (event.artifact_id && !activeArtifactId) {
      setActiveArtifactId(event.artifact_id)
    }

    if (event.status === 'FAILED') {
      const errMsg = event.error_message || 'Compilation failed in Kubernetes builder'
      setBuildError(errMsg)
      setCompilationLogs(errMsg)
    } else if (event.status === 'CANCELLED') {
      const errMsg = event.error_message || 'Build cancelled'
      setBuildError(errMsg)
      setCompilationLogs(errMsg)
    } else if (event.status === 'READY') {
      setBuildError(null)
    }
  })

  // Reset dialog state when closed
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setFile(null)
      setValidationError(null)
      setAstError(null)
      setIsUploading(false)
      setUploadProgress(0)
      setActiveArtifactId(null)
      setBuildStatus(null)
      setBuildError(null)
      setCompilationLogs(null)
    }
    onOpenChange(newOpen)
  }

  const validateFile = (candidate: File): boolean => {
    setValidationError(null)
    setAstError(null)

    const lowerName = candidate.name.toLowerCase()
    const isValidExtension = ALLOWED_EXTENSIONS.some((ext) => lowerName.endsWith(ext))

    if (!isValidExtension) {
      setValidationError(
        'Unsupported archive format. Please upload a .tar.gz, .tar.bz2, or .zip archive.'
      )
      return false
    }

    if (candidate.size > MAX_FILE_SIZE) {
      setValidationError('File size exceeds 50MB limit.')
      return false
    }

    return true
  }

  const handleFileSelection = (candidate: File | null) => {
    if (!candidate) {
      setFile(null)
      return
    }

    if (validateFile(candidate)) {
      setFile(candidate)
    } else {
      setFile(null)
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)

    const droppedFile = e.dataTransfer.files?.[0]
    if (droppedFile) {
      handleFileSelection(droppedFile)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!file) return

    setAstError(null)
    setBuildError(null)
    setCompilationLogs(null)
    setIsUploading(true)
    setUploadProgress(10)

    const formData = new FormData()
    formData.append('file', file)
    formData.append('source', file)
    formData.append('platform', platform)
    if (allowInsecure) {
      formData.append('allow_insecure_imports', 'true')
    }

    try {
      const res = await uploadMutation.mutateAsync({
        formData,
        onProgress: (percent) => {
          setUploadProgress(percent)
        },
      })

      setIsUploading(false)
      setUploadProgress(100)
      setBuildStatus('QUEUED')

      if (res && Array.isArray(res.artifacts) && res.artifacts.length > 0) {
        const first = res.artifacts[0]
        setActiveArtifactId(first.id)
        if (first.status) {
          setBuildStatus(first.status)
        }
      }
    } catch (err: any) {
      setIsUploading(false)
      const message = err?.message || 'Failed uploading and triggering build'

      // Check if this error is an AST static analysis rejection
      const isAstRejection =
        message.toLowerCase().includes('forbidden') ||
        message.toLowerCase().includes('prohibited') ||
        message.toLowerCase().includes('vuhive') ||
        message.toLowerCase().includes('package main') ||
        message.toLowerCase().includes('scenario') ||
        message.toLowerCase().includes('import') ||
        message.toLowerCase().includes('go.mod')

      if (isAstRejection) {
        setAstError(message)
      } else {
        setBuildError(message)
        setCompilationLogs(message)
      }
    }
  }

  const handleCancelBuild = async () => {
    if (!activeArtifactId) return
    try {
      await cancelMutation.mutateAsync({ artifactId: activeArtifactId, reason: 'Cancelled by user' })
      setBuildStatus('CANCELLED')
      setBuildError('Build was cancelled.')
    } catch (err: any) {
      console.error('Failed to cancel build:', err)
    }
  }

  const handleRetryBuild = async () => {
    if (!activeArtifactId) return
    try {
      await retryMutation.mutateAsync(activeArtifactId)
      setBuildStatus('BUILDING')
      setBuildError(null)
      setCompilationLogs(null)
    } catch (err: any) {
      console.error('Failed to retry build:', err)
    }
  }

  const handleResetForRetry = () => {
    setBuildStatus(null)
    setBuildError(null)
    setAstError(null)
    setCompilationLogs(null)
    setIsUploading(false)
    setUploadProgress(0)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="sm:max-w-xl max-h-[90vh] overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <UploadCloud className="w-5 h-5 text-brand-600 dark:text-brand-400" />
            <span>Upload Source &amp; Trigger Build</span>
          </DialogTitle>
          <DialogDescription>
            Upload a Go load scenario archive (.tar.gz, .tar.bz2, or .zip) for ephemeral compilation in a Kubernetes builder container.
          </DialogDescription>
        </DialogHeader>

        {/* Live Build Stepper View when build has been dispatched */}
        {buildStatus && (
          <div className="space-y-4 py-2">
            <BuildStatusStepper
              currentStatus={buildStatus}
              platform={platform}
              artifactId={activeArtifactId || undefined}
            />

            {buildStatus === 'READY' && (
              <div className="rounded-xl p-3 bg-emerald-50 dark:bg-emerald-950/60 border border-emerald-200 dark:border-emerald-900/60 flex items-center gap-2.5 text-xs text-emerald-800 dark:text-emerald-300">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0" />
                <span>Compilation succeeded! Binary artifact is ready and stored in S3.</span>
              </div>
            )}

            {buildStatus === 'FAILED' && (
              <div className="space-y-3">
                <div className="rounded-xl p-3 bg-red-50 dark:bg-red-950/60 border border-red-200 dark:border-red-900/60 flex items-start gap-2.5 text-xs text-red-800 dark:text-red-300">
                  <AlertCircle className="w-4 h-4 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <span className="font-semibold">Compilation Job Failed:</span>
                    <p className="mt-0.5">{buildError || 'The container compilation exited with a non-zero status.'}</p>
                  </div>
                </div>

                {compilationLogs && (
                  <BuildLogViewer
                    logs={compilationLogs}
                    title="Ephemeral Builder Job Output"
                    defaultExpanded={true}
                  />
                )}
              </div>
            )}

            {buildStatus === 'CANCELLED' && (
              <div className="space-y-3">
                <div className="rounded-xl p-3 bg-amber-50 dark:bg-amber-950/60 border border-amber-200 dark:border-amber-900/60 flex items-start gap-2.5 text-xs text-amber-800 dark:text-amber-300">
                  <Ban className="w-4 h-4 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <span className="font-semibold">Build Cancelled:</span>
                    <p className="mt-0.5">{buildError || 'The build was cancelled.'}</p>
                  </div>
                </div>

                {compilationLogs && (
                  <BuildLogViewer
                    logs={compilationLogs}
                    title="Build Cancellation Output"
                    defaultExpanded={true}
                  />
                )}
              </div>
            )}

            <DialogFooter className="gap-2 sm:gap-0 mt-4">
              {buildStatus === 'FAILED' || buildStatus === 'CANCELLED' ? (
                <>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleRetryBuild}
                    disabled={retryMutation.isPending || !activeArtifactId}
                    className="min-h-[44px] gap-1.5"
                  >
                    <RotateCw className={cn('w-4 h-4', retryMutation.isPending && 'animate-spin')} />
                    <span>Retry Build</span>
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleResetForRetry}
                    className="min-h-[44px] gap-1.5"
                  >
                    <UploadCloud className="w-4 h-4" />
                    <span>Upload New Package</span>
                  </Button>
                  <Button
                    type="button"
                    onClick={() => handleOpenChange(false)}
                    className="min-h-[44px]"
                  >
                    Close
                  </Button>
                </>
              ) : buildStatus === 'READY' ? (
                <Button
                  type="button"
                  onClick={() => handleOpenChange(false)}
                  className="min-h-[44px]"
                >
                  Done
                </Button>
              ) : (
                <>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCancelBuild}
                    disabled={cancelMutation.isPending || !activeArtifactId}
                    className="min-h-[44px] gap-1.5 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 hover:border-rose-300 border-rose-200 dark:border-rose-900/60"
                  >
                    <Ban className="w-4 h-4" />
                    <span>Cancel Build</span>
                  </Button>
                  <Button
                    type="button"
                    onClick={() => handleOpenChange(false)}
                    className="min-h-[44px]"
                  >
                    Dismiss
                  </Button>
                </>
              )}
            </DialogFooter>
          </div>
        )}

        {/* Upload Form View */}
        {!buildStatus && (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-4 py-1">
              {/* Drag-and-Drop File Dropzone */}
              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <label
                    htmlFor="source-file"
                    className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                  >
                    Source Package Archive (.tar.gz, .tar.bz2, .zip)
                  </label>
                  <HelpTooltip
                    text="Must contain go.mod and scenario.go with func NewScenario()."
                    label="Help for source archive"
                  />
                </div>

                {!file ? (
                  <div
                    data-testid="dropzone"
                    onDragOver={handleDragOver}
                    onDragEnter={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={() => document.getElementById('source-file')?.click()}
                    className={cn(
                      'border-2 border-dashed rounded-2xl p-6 text-center transition-all duration-200 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500',
                      isDragging
                        ? 'border-brand-500 bg-brand-50/70 dark:bg-brand-950/40 dark:border-brand-400 scale-[1.01]'
                        : 'border-slate-300 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/40 hover:bg-slate-50 dark:hover:bg-slate-800/40'
                    )}
                    role="button"
                    tabIndex={0}
                    aria-label="Drag and drop file upload area"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        document.getElementById('source-file')?.click()
                      }
                    }}
                  >
                    <input
                      id="source-file"
                      data-testid="source-file-input"
                      type="file"
                      accept=".tar.gz,.tgz,.tar.bz2,.tbz2,.zip,.tar"
                      onChange={(e) => handleFileSelection(e.target.files?.[0] || null)}
                      className="sr-only"
                    />

                    <div className="flex flex-col items-center justify-center space-y-2">
                      <div className="p-3 rounded-full bg-brand-100/80 dark:bg-brand-950 text-brand-600 dark:text-brand-400">
                        <UploadCloud className="w-6 h-6" />
                      </div>
                      <div className="space-y-1">
                        <p className="text-sm font-semibold text-slate-900 dark:text-white">
                          Drag &amp; drop your Go scenario archive
                        </p>
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                          Supports <span className="font-mono font-medium">.tar.gz</span>,{' '}
                          <span className="font-mono font-medium">.tar.bz2</span>, and{' '}
                          <span className="font-mono font-medium">.zip</span> archives up to 50MB
                        </p>
                      </div>
                      <span className="text-xs font-medium text-brand-600 dark:text-brand-400 underline underline-offset-2">
                        or browse from computer
                      </span>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-3.5 rounded-xl border border-brand-200 dark:border-brand-900/60 bg-brand-50/50 dark:bg-brand-950/30">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="p-2 rounded-lg bg-brand-100 dark:bg-brand-900 text-brand-600 dark:text-brand-300">
                        <FileArchive className="w-5 h-5" />
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-semibold text-slate-900 dark:text-white truncate">
                          {file.name}
                        </p>
                        <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">
                          {formatFileSize(file.size)}
                        </p>
                      </div>
                    </div>

                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => setFile(null)}
                      aria-label="Remove file"
                      className="h-8 w-8 p-0 rounded-full text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                    >
                      <X className="w-4 h-4" />
                    </Button>
                  </div>
                )}

                {/* Validation error message */}
                {validationError && (
                  <div className="flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400 font-medium">
                    <AlertCircle className="w-4 h-4 flex-shrink-0" />
                    <span>{validationError}</span>
                  </div>
                )}
              </div>

              {/* Architecture Selector */}
              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                    Target Platform
                  </span>
                  <HelpTooltip
                    text="Select target architecture for cross-compilation in Kubernetes."
                    label="Help for platform"
                  />
                </div>
                <div className="flex flex-wrap gap-2">
                  {[
                    { label: 'linux/amd64', val: 'linux/amd64' },
                    { label: 'linux/arm64', val: 'linux/arm64' },
                    { label: 'All Architectures', val: 'all' },
                  ].map((p) => {
                    const isSelected = platform === p.val
                    return (
                      <button
                        key={p.val}
                        type="button"
                        onClick={() => setPlatform(p.val)}
                        aria-pressed={isSelected}
                        className={cn(
                          'px-3 py-2 rounded-xl text-xs font-mono font-medium transition-all duration-200 border cursor-pointer min-h-[40px]',
                          isSelected
                            ? 'bg-brand-50 border-brand-400 text-brand-700 dark:bg-brand-950 dark:border-brand-700 dark:text-brand-300 shadow-xs'
                            : 'bg-slate-50 border-slate-200 text-slate-600 dark:bg-slate-800/60 dark:border-slate-700 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                        )}
                      >
                        {p.label}
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Allow Insecure Imports Switch */}
              <div className="flex items-center justify-between rounded-xl border border-slate-200 dark:border-slate-800 p-3 bg-slate-50 dark:bg-slate-800/40">
                <div className="space-y-0.5">
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    Allow Insecure Imports
                  </span>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    Override static AST safety filters for experimental or custom system packages.
                  </p>
                </div>
                <Switch checked={allowInsecure} onCheckedChange={setAllowInsecure} />
              </div>

              {/* Upload Progress Bar */}
              {isUploading && (
                <div className="space-y-2 p-3 rounded-xl bg-brand-50/50 dark:bg-brand-950/30 border border-brand-200 dark:border-brand-900/50">
                  <div className="flex items-center justify-between text-xs">
                    <span className="font-semibold text-brand-700 dark:text-brand-300">
                      Uploading scenario package...
                    </span>
                    <span className="font-mono text-brand-600 dark:text-brand-400">
                      {uploadProgress}%
                    </span>
                  </div>
                  <Progress value={uploadProgress} />
                </div>
              )}

              {/* AST Static Analysis Failure Callout */}
              {astError && (
                <AstErrorCallout
                  error={astError}
                  showInsecureOption={!allowInsecure}
                  onAllowInsecureToggle={() => setAllowInsecure(true)}
                />
              )}

              {/* AST Static Analysis Information Badge */}
              <InfoBadge variant="warning" title="AST Static Analysis Enforced">
                Uploaded code is verified via Go AST static analysis. Inverted control is enforced (
                <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 font-mono text-[11px]">
                  package scenario
                </code>
                ) and unsafe system imports (
                <code className="px-1 py-0.5 rounded bg-amber-100 dark:bg-amber-900/60 font-mono text-[11px]">
                  os/exec, syscall, unsafe
                </code>
                ) are blocked unless explicitly permitted.
              </InfoBadge>
            </div>

            <DialogFooter className="gap-2 sm:gap-0">
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                className="min-h-[44px]"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={!file || isUploading || Boolean(validationError)}
                className="min-h-[44px]"
              >
                {isUploading ? `Uploading (${uploadProgress}%)...` : 'Upload & Build'}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
