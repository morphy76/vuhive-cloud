import * as React from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import {
  Search,
  ChevronDown,
  ChevronUp,
  Download,
  Copy,
  Check,
  WrapText,
  SunMoon,
  ArrowDownToLine,
  Terminal,
  Loader2,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { parseAnsi, stripAnsi } from '@/lib/ansi'
import { cn } from '@/lib/utils'

export interface VirtualizedLogViewerProps {
  logs?: string
  runId?: string
  title?: string
  subtitle?: string
  isLoading?: boolean
  rawLogUrl?: string
  onDownloadRawLog?: () => void
  onClose?: () => void
  className?: string
}

/**
 * Highlights substring matches inside an ANSI segment text
 */
function renderHighlightedText(
  text: string,
  searchQuery: string,
  isActiveMatch: boolean
): React.ReactNode {
  if (!searchQuery) return text

  const lowerText = text.toLowerCase()
  const lowerQuery = searchQuery.toLowerCase()
  const startIndex = lowerText.indexOf(lowerQuery)

  if (startIndex === -1) return text

  const before = text.slice(0, startIndex)
  const match = text.slice(startIndex, startIndex + searchQuery.length)
  const after = text.slice(startIndex + searchQuery.length)

  return (
    <>
      {before}
      <mark
        className={cn(
          'rounded-xs px-0.5 font-bold',
          isActiveMatch
            ? 'bg-amber-400 text-slate-950 ring-2 ring-amber-300'
            : 'bg-yellow-500/40 text-white'
        )}
      >
        {match}
      </mark>
      {renderHighlightedText(after, searchQuery, isActiveMatch)}
    </>
  )
}

/**
 * Renders an individual log line with line numbers and parsed ANSI segments
 */
const AnsiLogLine = React.memo<{
  line: string
  lineNumber: number
  highContrast: boolean
  lineWrap: boolean
  searchQuery: string
  isCurrentMatchLine: boolean
}>(({ line, lineNumber, highContrast, lineWrap, searchQuery, isCurrentMatchLine }) => {
  const segments = React.useMemo(
    () => parseAnsi(line, { highContrast }),
    [line, highContrast]
  )

  return (
    <div
      className={cn(
        'flex items-start font-mono text-xs leading-6 select-text hover:bg-slate-800/40 transition-colors',
        isCurrentMatchLine && 'bg-brand-950/60 ring-1 ring-inset ring-brand-500/50'
      )}
    >
      <span
        className={cn(
          'select-none pr-3 pl-2 text-right flex-shrink-0 font-mono text-[11px]',
          highContrast ? 'text-slate-400 font-bold' : 'text-slate-500'
        )}
        style={{ width: '4.5rem' }}
      >
        {lineNumber}
      </span>
      <div
        className={cn(
          'flex-1 pr-4 min-w-0',
          lineWrap ? 'whitespace-pre-wrap break-all' : 'whitespace-pre overflow-visible'
        )}
      >
        {segments.length === 0 ? (
          <span>&nbsp;</span>
        ) : (
          segments.map((seg, idx) => (
            <span
              key={idx}
              style={seg.style}
              className={cn(seg.className, !seg.style?.color && (highContrast ? 'text-white' : 'text-slate-200'))}
            >
              {renderHighlightedText(seg.text, searchQuery, isCurrentMatchLine)}
            </span>
          ))
        )}
      </div>
    </div>
  )
})

AnsiLogLine.displayName = 'AnsiLogLine'

export const VirtualizedLogViewer: React.FC<VirtualizedLogViewerProps> = ({
  logs = '',
  runId,
  title = 'Execution Logs',
  subtitle,
  isLoading = false,
  rawLogUrl,
  onDownloadRawLog,
  onClose,
  className,
}) => {
  const parentRef = React.useRef<HTMLDivElement>(null)

  // Viewer Preferences & Controls
  const [searchTerm, setSearchTerm] = React.useState('')
  const [currentMatchIndex, setCurrentMatchIndex] = React.useState(0)
  const [autoScroll, setAutoScroll] = React.useState(true)
  const [lineWrap, setLineWrap] = React.useState(false)
  const [highContrast, setHighContrast] = React.useState(false)
  const [copied, setCopied] = React.useState(false)

  // Memoize lines array for optimal memory and fast index access
  const lines = React.useMemo(() => {
    if (!logs) return []
    return logs.split(/\r?\n/)
  }, [logs])

  // Memoize stripped lines for performant search matching
  const plainLines = React.useMemo(() => {
    return lines.map((l) => stripAnsi(l).toLowerCase())
  }, [lines])

  // Calculate matching line indices
  const matchingLineIndices = React.useMemo(() => {
    const query = searchTerm.trim().toLowerCase()
    if (!query) return []
    const matches: number[] = []
    for (let i = 0; i < plainLines.length; i++) {
      if (plainLines[i].includes(query)) {
        matches.push(i)
      }
    }
    return matches
  }, [plainLines, searchTerm])

  // TanStack Virtualizer for high-performance rendering across 50,000+ lines
  const rowVirtualizer = useVirtualizer({
    count: lines.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 24, // 24px line-height
    overscan: 25,
  })

  // Keep search match index in bounds when matches change
  React.useEffect(() => {
    setCurrentMatchIndex(0)
  }, [searchTerm])

  // Scroll to active search match
  const scrollToMatch = React.useCallback(
    (targetMatchIndex: number) => {
      if (matchingLineIndices.length === 0) return
      const lineIdx = matchingLineIndices[targetMatchIndex]
      if (lineIdx !== undefined) {
        rowVirtualizer.scrollToIndex(lineIdx, { align: 'center', behavior: 'auto' })
      }
    },
    [matchingLineIndices, rowVirtualizer]
  )

  const handleNextMatch = () => {
    if (matchingLineIndices.length === 0) return
    const nextIdx = (currentMatchIndex + 1) % matchingLineIndices.length
    setCurrentMatchIndex(nextIdx)
    scrollToMatch(nextIdx)
  }

  const handlePrevMatch = () => {
    if (matchingLineIndices.length === 0) return
    const prevIdx =
      (currentMatchIndex - 1 + matchingLineIndices.length) % matchingLineIndices.length
    setCurrentMatchIndex(prevIdx)
    scrollToMatch(prevIdx)
  }

  // Auto-scroll to bottom if autoScroll is enabled
  React.useEffect(() => {
    if (autoScroll && lines.length > 0 && !searchTerm) {
      rowVirtualizer.scrollToIndex(lines.length - 1, { align: 'end', behavior: 'auto' })
    }
  }, [lines.length, autoScroll, searchTerm, rowVirtualizer])

  // Handle Copy
  const handleCopyLogs = async () => {
    if (!logs) return
    try {
      await navigator.clipboard.writeText(logs)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // ignore
    }
  }

  // Handle Download
  const handleDownload = () => {
    if (onDownloadRawLog) {
      onDownloadRawLog()
      return
    }

    if (rawLogUrl) {
      window.open(rawLogUrl, '_blank')
      return
    }

    if (runId) {
      window.open(`/api/v1/runs/${encodeURIComponent(runId)}/logs`, '_blank')
      return
    }

    // Fallback: download raw in-memory string as blob
    const blob = new Blob([logs], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `run-${runId || 'execution'}.log`
    a.click()
    URL.revokeObjectURL(url)
  }

  const currentMatchedLine =
    matchingLineIndices.length > 0 ? matchingLineIndices[currentMatchIndex] : -1

  return (
    <div
      data-high-contrast={highContrast}
      className={cn(
        'flex flex-col rounded-2xl border shadow-lg overflow-hidden transition-colors',
        highContrast
          ? 'bg-black border-slate-700 text-white'
          : 'bg-slate-950 border-slate-800 text-slate-100',
        className
      )}
    >
      {/* 1. Header Toolbar */}
      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-b',
          highContrast
            ? 'bg-neutral-900 border-neutral-800'
            : 'bg-slate-900/90 border-slate-800/80 backdrop-blur-xs'
        )}
      >
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'p-2 rounded-xl flex items-center justify-center',
              highContrast
                ? 'bg-white text-black'
                : 'bg-brand-500/10 text-brand-400 border border-brand-500/20'
            )}
          >
            <Terminal className="w-4 h-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-bold tracking-tight text-white">
                {title}
              </h3>
              {runId && (
                <span className="font-mono text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                  {runId}
                </span>
              )}
            </div>
            <div className="flex items-center gap-2 mt-0.5 text-xs text-slate-400">
              {subtitle && <span>{subtitle} • </span>}
              <span className="font-mono">
                {lines.length.toLocaleString()} {lines.length === 1 ? 'line' : 'lines'}
              </span>
            </div>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2">
          {/* Search Input Bar */}
          <div className="relative flex items-center">
            <Search className="w-3.5 h-3.5 absolute left-2.5 text-slate-400 pointer-events-none" />
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search logs..."
              className={cn(
                'h-8 pl-8 pr-20 text-xs rounded-lg border font-mono transition-colors focus:outline-none focus:ring-1',
                highContrast
                  ? 'bg-black border-slate-600 text-white focus:ring-white placeholder-slate-400'
                  : 'bg-slate-950 border-slate-700 text-slate-200 focus:ring-brand-500 placeholder-slate-500'
              )}
            />

            {/* Match Counter & Next/Prev Controls */}
            {searchTerm.trim() && (
              <div className="absolute right-1.5 flex items-center gap-0.5">
                <span className="text-[10px] font-mono px-1 text-slate-400 select-none">
                  {matchingLineIndices.length > 0
                    ? `${currentMatchIndex + 1} of ${matchingLineIndices.length}`
                    : '0 matches'}
                </span>
                <button
                  type="button"
                  onClick={handlePrevMatch}
                  disabled={matchingLineIndices.length === 0}
                  className="p-1 text-slate-400 hover:text-white disabled:opacity-30 cursor-pointer"
                  aria-label="Previous match"
                >
                  <ChevronUp className="w-3 h-3" />
                </button>
                <button
                  type="button"
                  onClick={handleNextMatch}
                  disabled={matchingLineIndices.length === 0}
                  className="p-1 text-slate-400 hover:text-white disabled:opacity-30 cursor-pointer"
                  aria-label="Next match"
                >
                  <ChevronDown className="w-3 h-3" />
                </button>
              </div>
            )}
          </div>

          {/* Toggle Auto-scroll Button */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setAutoScroll(!autoScroll)}
            aria-pressed={autoScroll}
            aria-label="Toggle auto-scroll"
            className={cn(
              'h-8 px-2.5 text-xs font-mono gap-1.5 border transition-colors',
              autoScroll
                ? highContrast
                  ? 'bg-white text-black border-white'
                  : 'bg-brand-500/20 text-brand-300 border-brand-500/40 hover:bg-brand-500/30'
                : 'bg-transparent text-slate-400 border-slate-800 hover:text-slate-200 hover:bg-slate-800/60'
            )}
          >
            <ArrowDownToLine className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Auto-scroll</span>
          </Button>

          {/* Toggle Line Wrap Button */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setLineWrap(!lineWrap)}
            aria-pressed={lineWrap}
            aria-label="Toggle line wrap"
            className={cn(
              'h-8 px-2.5 text-xs font-mono gap-1.5 border transition-colors',
              lineWrap
                ? highContrast
                  ? 'bg-white text-black border-white'
                  : 'bg-brand-500/20 text-brand-300 border-brand-500/40 hover:bg-brand-500/30'
                : 'bg-transparent text-slate-400 border-slate-800 hover:text-slate-200 hover:bg-slate-800/60'
            )}
          >
            <WrapText className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Wrap</span>
          </Button>

          {/* Toggle Dark / High-Contrast Mode Button */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setHighContrast(!highContrast)}
            aria-pressed={highContrast}
            aria-label="Toggle high-contrast mode"
            className={cn(
              'h-8 px-2 text-xs border transition-colors',
              highContrast
                ? 'bg-amber-400 text-black border-amber-400 font-bold'
                : 'bg-transparent text-slate-400 border-slate-800 hover:text-slate-200 hover:bg-slate-800/60'
            )}
          >
            <SunMoon className="w-3.5 h-3.5" />
          </Button>

          {/* Copy Logs Button */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleCopyLogs}
            aria-label="Copy logs"
            className="h-8 px-2 text-xs text-slate-400 hover:text-white border border-slate-800 hover:bg-slate-800/60 gap-1"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-[11px] text-emerald-400 hidden sm:inline">Copied</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span className="hidden sm:inline">Copy</span>
              </>
            )}
          </Button>

          {/* Download Raw Log Button */}
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleDownload}
            aria-label="Download raw log"
            className="h-8 px-2.5 text-xs text-slate-300 hover:text-white border border-slate-800 hover:bg-slate-800/60 gap-1.5"
          >
            <Download className="w-3.5 h-3.5 text-brand-400" />
            <span className="hidden sm:inline">Download</span>
          </Button>

          {onClose && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={onClose}
              className="h-8 w-8 text-slate-400 hover:text-white"
              aria-label="Close log viewer"
            >
              <X className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>

      {/* 2. Virtualized Log Content Viewport */}
      <div
        ref={parentRef}
        tabIndex={0}
        aria-label="Virtual execution log terminal stream"
        className={cn(
          'flex-1 min-h-[400px] max-h-[70vh] overflow-auto outline-none select-text',
          highContrast ? 'bg-black' : 'bg-slate-950'
        )}
      >
        {isLoading ? (
          <div className="h-64 flex flex-col items-center justify-center gap-2 text-slate-400">
            <Loader2 className="w-6 h-6 animate-spin text-brand-400" />
            <span className="text-xs font-mono">Loading logs...</span>
          </div>
        ) : lines.length === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center text-slate-500 font-mono text-xs">
            No log output recorded yet.
          </div>
        ) : (
          <div
            style={{
              height: `${rowVirtualizer.getTotalSize()}px`,
              width: '100%',
              position: 'relative',
            }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const line = lines[virtualRow.index] || ''
              const isMatch = virtualRow.index === currentMatchedLine

              return (
                <div
                  key={virtualRow.key}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  <AnsiLogLine
                    line={line}
                    lineNumber={virtualRow.index + 1}
                    highContrast={highContrast}
                    lineWrap={lineWrap}
                    searchQuery={searchTerm.trim()}
                    isCurrentMatchLine={isMatch}
                  />
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
