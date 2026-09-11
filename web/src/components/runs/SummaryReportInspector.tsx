import React, { useState } from 'react'
import {
  Table as TableIcon,
  FolderTree,
  Download,
  Copy,
  Check,
  Search,
  AlertTriangle,
  Loader2,
  RefreshCw,
  X,
  FileJson,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useRunReport } from '@/hooks/use-runs'
import { JsonTreeViewer } from '@/components/runs/JsonTreeViewer'
import { SummaryTableView } from '@/components/runs/SummaryTableView'
import { downloadJsonFile } from '@/lib/export-utils'
import type { SummaryReport } from '@/types/report'

export interface SummaryReportInspectorProps {
  runId: string
  report?: SummaryReport
  onClose?: () => void
  className?: string
}

export const SummaryReportInspector: React.FC<SummaryReportInspectorProps> = ({
  runId,
  report: initialReport,
  onClose,
  className = '',
}) => {
  const [viewMode, setViewMode] = useState<'table' | 'tree'>('table')
  const [searchQuery, setSearchQuery] = useState('')
  const [copiedJson, setCopiedJson] = useState(false)

  // Fetch report via hook if not provided directly
  const {
    data: fetchedReport,
    isLoading,
    isError,
    error,
    refetch,
  } = useRunReport(initialReport ? undefined : runId)

  const report = initialReport || fetchedReport

  const handleDownloadJson = () => {
    if (!report) return
    downloadJsonFile(report, `summary-${runId}.json`)
  }

  const handleCopyJson = async () => {
    if (!report) return
    try {
      await navigator.clipboard.writeText(JSON.stringify(report, null, 2))
      setCopiedJson(true)
      setTimeout(() => setCopiedJson(false), 2000)
    } catch (err) {
      console.error('Failed copying JSON to clipboard', err)
    }
  }

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Top Header & Toolbar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-slate-200 dark:border-slate-800">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-brand-600 dark:text-brand-400">
              Telemetry Inspector
            </span>
            <span className="font-mono text-xs text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded-md">
              {runId}
            </span>
            {report?.status && (
              <Badge
                variant={report.status === 'PASS' || report.passed ? 'success' : 'error'}
                className="text-[10px] uppercase font-mono"
              >
                {report.status}
              </Badge>
            )}
          </div>
          <h3 className="text-lg sm:text-xl font-bold text-slate-900 dark:text-white mt-1">
            {report?.suite_name || 'Test Execution Telemetry Report'}
          </h3>
          {report?.scenario && (
            <p className="text-xs text-slate-500 dark:text-slate-400 font-mono mt-0.5">
              Scenario: {report.scenario} {report.version ? `(v${report.version})` : ''}
            </p>
          )}
        </div>

        {/* View Mode & Export Controls */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* View Mode Switcher */}
          <div className="flex items-center bg-slate-100 dark:bg-slate-800 p-1 rounded-lg border border-slate-200/80 dark:border-slate-700/80">
            <button
              type="button"
              onClick={() => setViewMode('table')}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                viewMode === 'table'
                  ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs font-semibold'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
              }`}
              aria-label="Tabular View"
            >
              <TableIcon className="w-3.5 h-3.5" />
              <span>Tabular View</span>
            </button>

            <button
              type="button"
              onClick={() => setViewMode('tree')}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors cursor-pointer ${
                viewMode === 'tree'
                  ? 'bg-white dark:bg-slate-900 text-slate-900 dark:text-white shadow-xs font-semibold'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
              }`}
              aria-label="JSON Tree"
            >
              <FolderTree className="w-3.5 h-3.5" />
              <span>JSON Tree</span>
            </button>
          </div>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleDownloadJson}
            disabled={!report}
            className="h-8 text-xs gap-1.5 border-slate-200 dark:border-slate-800"
            aria-label="Download JSON"
          >
            <Download className="w-3.5 h-3.5 text-brand-600 dark:text-brand-400" />
            <span>Download JSON</span>
          </Button>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopyJson}
            disabled={!report}
            className="h-8 text-xs gap-1.5 border-slate-200 dark:border-slate-800"
            aria-label="Copy JSON"
          >
            {copiedJson ? (
              <Check className="w-3.5 h-3.5 text-emerald-500" />
            ) : (
              <Copy className="w-3.5 h-3.5" />
            )}
            <span>Copy JSON</span>
          </Button>

          {onClose && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={onClose}
              className="h-8 w-8 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
              aria-label="Close inspector"
            >
              <X className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>

      {/* Global Search Filter */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search telemetry steps, metrics, or JSON keys..."
          className="w-full pl-9 pr-8 py-2 text-xs bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl focus:outline-hidden focus:ring-2 focus:ring-brand-500 text-slate-900 dark:text-white placeholder:text-slate-400"
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => setSearchQuery('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
            aria-label="Clear search query"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      {/* State Renderers */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center py-16 text-slate-500 dark:text-slate-400 space-y-3">
          <Loader2 className="w-8 h-8 animate-spin text-brand-600 dark:text-brand-400" />
          <p className="text-sm font-medium">Loading telemetry report...</p>
        </div>
      ) : isError ? (
        <div className="p-6 rounded-2xl bg-rose-50 dark:bg-rose-950/30 border border-rose-200 dark:border-rose-900/50 space-y-3">
          <div className="flex items-center gap-2.5 text-rose-700 dark:text-rose-400 font-semibold">
            <AlertTriangle className="w-5 h-5" />
            <span>Unable to load telemetry report</span>
          </div>
          <p className="text-xs text-rose-600 dark:text-rose-300">
            {error instanceof Error ? error.message : 'An error occurred while fetching summary.json'}
          </p>
          <div className="pt-2 flex items-center gap-3">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              className="gap-1.5 text-xs border-rose-300 dark:border-rose-800"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              <span>Retry</span>
            </Button>
            <Button
              variant="outline"
              size="sm"
              asChild
              className="gap-1.5 text-xs border-slate-200 dark:border-slate-800"
            >
              <a
                href={`/api/v1/runs/${encodeURIComponent(runId)}/report?presign=true`}
                target="_blank"
                rel="noopener noreferrer"
              >
                <FileJson className="w-3.5 h-3.5 text-brand-600 dark:text-brand-400" />
                <span>Download S3 Presigned Report</span>
              </a>
            </Button>
          </div>
        </div>
      ) : !report ? (
        <div className="p-8 text-center text-slate-400 rounded-xl border border-dashed border-slate-200 dark:border-slate-800">
          No telemetry summary report is available for this run.
        </div>
      ) : viewMode === 'table' ? (
        <SummaryTableView
          report={report}
          runId={runId}
          searchQuery={searchQuery}
        />
      ) : (
        <JsonTreeViewer
          data={report}
          searchQuery={searchQuery}
        />
      )}
    </div>
  )
}
