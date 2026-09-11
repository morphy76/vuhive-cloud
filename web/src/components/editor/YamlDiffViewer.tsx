import React, { useMemo, useState } from 'react'
import { diffLines, Change } from 'diff'
import { Columns, Split, CheckCircle2, Plus, Minus } from 'lucide-react'
import { Button } from '@/components/ui/button'

export interface YamlDiffViewerProps {
  baseYaml: string
  comparisonYaml: string
  baseTitle?: string
  comparisonTitle?: string
  className?: string
}

interface DiffLine {
  lineNum?: number
  text: string
  type: 'added' | 'removed' | 'unchanged'
}

export const YamlDiffViewer: React.FC<YamlDiffViewerProps> = ({
  baseYaml,
  comparisonYaml,
  baseTitle = 'Base Version',
  comparisonTitle = 'Comparison Version',
  className = '',
}) => {
  const [viewMode, setViewMode] = useState<'split' | 'unified'>('split')

  const { leftLines, rightLines, unifiedLines, addedCount, removedCount, isIdentical } =
    useMemo(() => {
      const changes: Change[] = diffLines(baseYaml || '', comparisonYaml || '')
      const left: DiffLine[] = []
      const right: DiffLine[] = []
      const unified: DiffLine[] = []

      let leftLineNum = 1
      let rightLineNum = 1
      let added = 0
      let removed = 0

      for (let i = 0; i < changes.length; i++) {
        const change = changes[i]
        const rawLines = change.value.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n')
        // if trailing empty line due to ending newline, discard it if it was the last token
        if (rawLines.length > 0 && rawLines[rawLines.length - 1] === '') {
          rawLines.pop()
        }

        if (change.added) {
          added += rawLines.length
          for (const line of rawLines) {
            right.push({ lineNum: rightLineNum++, text: line, type: 'added' })
            unified.push({ lineNum: rightLineNum - 1, text: line, type: 'added' })
          }
        } else if (change.removed) {
          removed += rawLines.length
          for (const line of rawLines) {
            left.push({ lineNum: leftLineNum++, text: line, type: 'removed' })
            unified.push({ lineNum: leftLineNum - 1, text: line, type: 'removed' })
          }
        } else {
          for (const line of rawLines) {
            left.push({ lineNum: leftLineNum++, text: line, type: 'unchanged' })
            right.push({ lineNum: rightLineNum++, text: line, type: 'unchanged' })
            unified.push({ lineNum: rightLineNum - 1, text: line, type: 'unchanged' })
          }
        }
      }

      // Pad panels so they align
      const maxRows = Math.max(left.length, right.length)
      while (left.length < maxRows) {
        left.push({ text: '', type: 'unchanged' })
      }
      while (right.length < maxRows) {
        right.push({ text: '', type: 'unchanged' })
      }

      const identical = added === 0 && removed === 0 && baseYaml.trim() === comparisonYaml.trim()

      return {
        leftLines: left,
        rightLines: right,
        unifiedLines: unified,
        addedCount: added,
        removedCount: removed,
        isIdentical: identical,
      }
    }, [baseYaml, comparisonYaml])

  return (
    <div
      role="region"
      aria-label="YAML Difference Viewer"
      className={`rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 overflow-hidden shadow-xs flex flex-col ${className}`}
    >
      {/* Diff Header Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800">
        <div className="flex items-center gap-3">
          {isIdentical ? (
            <div className="flex items-center gap-1.5 text-xs font-medium text-emerald-600 dark:text-emerald-400">
              <CheckCircle2 className="w-4 h-4" />
              <span>Configurations are identical</span>
            </div>
          ) : (
            <div className="flex items-center gap-2 text-xs font-mono font-medium">
              <span className="flex items-center gap-0.5 px-2 py-0.5 rounded bg-emerald-100 dark:bg-emerald-950/80 text-emerald-700 dark:text-emerald-300">
                <Plus className="w-3 h-3" />
                {addedCount} added
              </span>
              <span className="flex items-center gap-0.5 px-2 py-0.5 rounded bg-rose-100 dark:bg-rose-950/80 text-rose-700 dark:text-rose-300">
                <Minus className="w-3 h-3" />
                {removedCount} removed
              </span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant={viewMode === 'split' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('split')}
            className="h-8 gap-1 text-xs px-2.5"
            aria-label="Side-by-side split diff view"
          >
            <Columns className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Side-by-Side</span>
          </Button>
          <Button
            type="button"
            variant={viewMode === 'unified' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('unified')}
            className="h-8 gap-1 text-xs px-2.5"
            aria-label="Unified inline diff view"
          >
            <Split className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Unified</span>
          </Button>
        </div>
      </div>

      {/* Side-by-side Panel Headings */}
      {viewMode === 'split' && (
        <div className="grid grid-cols-2 text-xs font-semibold border-b border-slate-200 dark:border-slate-800 bg-slate-100/50 dark:bg-slate-800/30">
          <div className="px-4 py-2 text-slate-700 dark:text-slate-300 border-r border-slate-200 dark:border-slate-800 truncate">
            {baseTitle}
          </div>
          <div className="px-4 py-2 text-slate-700 dark:text-slate-300 truncate">
            {comparisonTitle}
          </div>
        </div>
      )}

      {/* Diff Content Body */}
      <div className="overflow-x-auto max-h-[480px] font-mono text-xs select-text">
        {viewMode === 'split' ? (
          <div className="grid grid-cols-2 divide-x divide-slate-200 dark:divide-slate-800 min-w-[600px]">
            {/* Left / Base Side */}
            <div className="divide-y divide-slate-100 dark:divide-slate-800/40">
              {leftLines.map((line, idx) => (
                <div
                  key={`left-${idx}`}
                  className={`flex items-start min-h-[22px] px-2 py-0.5 ${
                    line.type === 'removed'
                      ? 'bg-rose-50 dark:bg-rose-950/40 text-rose-800 dark:text-rose-200'
                      : 'text-slate-800 dark:text-slate-200'
                  }`}
                >
                  <span className="w-8 shrink-0 select-none text-right pr-3 text-slate-400 dark:text-slate-600 text-[11px]">
                    {line.lineNum ?? ''}
                  </span>
                  <span className="w-4 shrink-0 select-none font-bold text-rose-500">
                    {line.type === 'removed' ? '-' : ' '}
                  </span>
                  <pre className="overflow-x-auto whitespace-pre font-mono flex-1 m-0 p-0 bg-transparent">
                    {line.text}
                  </pre>
                </div>
              ))}
            </div>

            {/* Right / Comparison Side */}
            <div className="divide-y divide-slate-100 dark:divide-slate-800/40">
              {rightLines.map((line, idx) => (
                <div
                  key={`right-${idx}`}
                  className={`flex items-start min-h-[22px] px-2 py-0.5 ${
                    line.type === 'added'
                      ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-800 dark:text-emerald-200'
                      : 'text-slate-800 dark:text-slate-200'
                  }`}
                >
                  <span className="w-8 shrink-0 select-none text-right pr-3 text-slate-400 dark:text-slate-600 text-[11px]">
                    {line.lineNum ?? ''}
                  </span>
                  <span className="w-4 shrink-0 select-none font-bold text-emerald-500">
                    {line.type === 'added' ? '+' : ' '}
                  </span>
                  <pre className="overflow-x-auto whitespace-pre font-mono flex-1 m-0 p-0 bg-transparent">
                    {line.text}
                  </pre>
                </div>
              ))}
            </div>
          </div>
        ) : (
          /* Unified View */
          <div className="divide-y divide-slate-100 dark:divide-slate-800/40 min-w-[500px]">
            {unifiedLines.map((line, idx) => (
              <div
                key={`uni-${idx}`}
                className={`flex items-start min-h-[22px] px-3 py-0.5 ${
                  line.type === 'added'
                    ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-800 dark:text-emerald-200'
                    : line.type === 'removed'
                    ? 'bg-rose-50 dark:bg-rose-950/40 text-rose-800 dark:text-rose-200'
                    : 'text-slate-800 dark:text-slate-200'
                }`}
              >
                <span className="w-8 shrink-0 select-none text-right pr-3 text-slate-400 dark:text-slate-600 text-[11px]">
                  {line.lineNum ?? ''}
                </span>
                <span
                  className={`w-4 shrink-0 select-none font-bold ${
                    line.type === 'added'
                      ? 'text-emerald-500'
                      : line.type === 'removed'
                      ? 'text-rose-500'
                      : 'text-slate-400'
                  }`}
                >
                  {line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '}
                </span>
                <pre className="overflow-x-auto whitespace-pre font-mono flex-1 m-0 p-0 bg-transparent">
                  {line.text}
                </pre>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
