import React, { useState, useMemo, useCallback } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  Search,
  Maximize2,
  Minimize2,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

export interface JsonTreeViewerProps {
  data: any
  initialExpandedDepth?: number
  searchQuery?: string
  onCopyPath?: (path: string) => void
  className?: string
}

function matchesSearch(key: string, val: any, query: string): boolean {
  if (!query) return false
  const lowerQ = query.toLowerCase()
  if (key.toLowerCase().includes(lowerQ)) return true
  if (typeof val === 'string' && val.toLowerCase().includes(lowerQ)) return true
  if (typeof val === 'number' && String(val).includes(lowerQ)) return true
  if (typeof val === 'boolean' && String(val).includes(lowerQ)) return true
  return false
}

function hasMatchingDescendant(data: any, query: string): boolean {
  if (!query || data === null || data === undefined) return false
  if (typeof data !== 'object') return false

  if (Array.isArray(data)) {
    for (let i = 0; i < data.length; i++) {
      if (matchesSearch(String(i), data[i], query) || hasMatchingDescendant(data[i], query)) {
        return true
      }
    }
  } else {
    for (const [k, v] of Object.entries(data)) {
      if (matchesSearch(k, v, query) || hasMatchingDescendant(v, query)) {
        return true
      }
    }
  }
  return false
}

function countTotalMatches(data: any, query: string): number {
  if (!query || data === null || data === undefined) return 0
  let count = 0

  function traverse(obj: any, parentKey: string) {
    if (matchesSearch(parentKey, obj, query)) {
      count++
    }
    if (obj && typeof obj === 'object') {
      if (Array.isArray(obj)) {
        obj.forEach((item, idx) => traverse(item, String(idx)))
      } else {
        Object.entries(obj).forEach(([k, v]) => traverse(v, k))
      }
    }
  }

  traverse(data, '$')
  return count
}

function collectAllPaths(data: any, currentPath = '$', maxDepth = 10): string[] {
  if (maxDepth <= 0 || !data || typeof data !== 'object') return []
  const paths: string[] = [currentPath]

  if (Array.isArray(data)) {
    data.forEach((item, idx) => {
      const p = `${currentPath}[${idx}]`
      paths.push(...collectAllPaths(item, p, maxDepth - 1))
    })
  } else {
    Object.entries(data).forEach(([k, v]) => {
      const p = currentPath === '$' ? k : `${currentPath}.${k}`
      paths.push(...collectAllPaths(v, p, maxDepth - 1))
    })
  }

  return paths
}

function collectInitialExpandedPaths(data: any, maxDepth: number, currentPath = '$', currentDepth = 0): string[] {
  if (currentDepth >= maxDepth || !data || typeof data !== 'object') return []
  const paths: string[] = [currentPath]

  if (Array.isArray(data)) {
    data.forEach((item, idx) => {
      const p = `${currentPath}[${idx}]`
      paths.push(...collectInitialExpandedPaths(item, maxDepth, p, currentDepth + 1))
    })
  } else {
    Object.entries(data).forEach(([k, v]) => {
      const p = currentPath === '$' ? k : `${currentPath}.${k}`
      paths.push(...collectInitialExpandedPaths(v, maxDepth, p, currentDepth + 1))
    })
  }

  return paths
}

interface NodeProps {
  nodeKey?: string
  value: any
  path: string
  depth: number
  expandedPaths: Set<string>
  togglePath: (path: string) => void
  searchQuery: string
  copiedPath: string | null
  onCopyPath: (path: string) => void
  onCopyValue: (val: any) => void
}

function HighlightText({ text, query }: { text: string; query: string }) {
  if (!query) return <>{text}</>
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()
  const idx = lowerText.indexOf(lowerQuery)
  if (idx === -1) return <>{text}</>

  const before = text.slice(0, idx)
  const match = text.slice(idx, idx + query.length)
  const after = text.slice(idx + query.length)

  return (
    <>
      {before}
      <mark className="bg-amber-200 dark:bg-amber-900/60 text-amber-950 dark:text-amber-100 rounded-xs px-0.5 font-bold">
        {match}
      </mark>
      <HighlightText text={after} query={query} />
    </>
  )
}

const TreeNode: React.FC<NodeProps> = ({
  nodeKey,
  value,
  path,
  depth,
  expandedPaths,
  togglePath,
  searchQuery,
  copiedPath,
  onCopyPath,
  onCopyValue,
}) => {
  const isObject = value !== null && typeof value === 'object'
  const isArray = Array.isArray(value)
  const isExpanded = expandedPaths.has(path)
  const isMatching = nodeKey ? matchesSearch(nodeKey, value, searchQuery) : false
  const hasMatchingChild = searchQuery ? hasMatchingDescendant(value, searchQuery) : false

  const cleanPath = path.startsWith('$.') ? path.slice(2) : path.startsWith('$[') ? path.slice(1) : path === '$' ? '' : path

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (isObject) {
      togglePath(path)
    }
  }

  return (
    <div className="font-mono text-xs select-text">
      <div
        className={`group flex items-center gap-1.5 py-0.5 px-1 rounded-md transition-colors hover:bg-slate-100 dark:hover:bg-slate-800/60 ${
          isMatching ? 'bg-amber-50 dark:bg-amber-950/40' : ''
        }`}
      >
        {isObject ? (
          <button
            type="button"
            onClick={handleToggle}
            className="w-4 h-4 flex items-center justify-center text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 cursor-pointer"
            aria-label={isExpanded ? `Collapse ${cleanPath || 'root'}` : `Expand ${cleanPath || 'root'}`}
          >
            {isExpanded ? (
              <ChevronDown className="w-3.5 h-3.5" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5" />
            )}
          </button>
        ) : (
          <span className="w-4" />
        )}

        {nodeKey !== undefined && (
          <span
            onClick={isObject ? handleToggle : undefined}
            className={`font-semibold cursor-pointer ${
              isObject
                ? 'text-purple-700 dark:text-purple-300 hover:underline'
                : 'text-slate-700 dark:text-slate-300'
            }`}
          >
            <HighlightText text={nodeKey} query={searchQuery} />
            <span className="text-slate-400 dark:text-slate-500 font-normal">:</span>
          </span>
        )}

        {/* Value Rendering */}
        {isObject ? (
          <span
            onClick={handleToggle}
            className="text-slate-400 dark:text-slate-500 cursor-pointer"
          >
            {isArray ? `[${value.length} items]` : `{${Object.keys(value).length} keys}`}
          </span>
        ) : typeof value === 'string' ? (
          <span className="text-emerald-600 dark:text-emerald-400 break-all">
            &quot;<HighlightText text={value} query={searchQuery} />&quot;
          </span>
        ) : typeof value === 'number' ? (
          <span className="text-amber-600 dark:text-amber-400 font-semibold">
            <HighlightText text={String(value)} query={searchQuery} />
          </span>
        ) : typeof value === 'boolean' ? (
          <span className="text-cyan-600 dark:text-cyan-400 font-bold">
            <HighlightText text={String(value)} query={searchQuery} />
          </span>
        ) : value === null ? (
          <span className="text-slate-400 italic">null</span>
        ) : (
          <span className="text-slate-400 italic">{String(value)}</span>
        )}

        {/* Action icons on hover */}
        <div className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 flex items-center gap-1 ml-auto transition-opacity">
          {cleanPath && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                onCopyPath(cleanPath)
              }}
              title={`Copy path: ${cleanPath}`}
              className="p-1 rounded text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-200/60 dark:hover:bg-slate-700 cursor-pointer text-[10px]"
              aria-label={`Copy path ${cleanPath}`}
            >
              {copiedPath === cleanPath ? (
                <Check className="w-3 h-3 text-emerald-500" />
              ) : (
                <Copy className="w-3 h-3" />
              )}
            </button>
          )}
        </div>
      </div>

      {/* Render children if object and expanded */}
      {isObject && (isExpanded || (searchQuery && hasMatchingChild)) && (
        <div className="border-l border-slate-200 dark:border-slate-800 ml-2.5 pl-2.5 space-y-0.5 mt-0.5">
          {isArray
            ? value.map((item: any, idx: number) => (
                <TreeNode
                  key={idx}
                  nodeKey={String(idx)}
                  value={item}
                  path={`${path}[${idx}]`}
                  depth={depth + 1}
                  expandedPaths={expandedPaths}
                  togglePath={togglePath}
                  searchQuery={searchQuery}
                  copiedPath={copiedPath}
                  onCopyPath={onCopyPath}
                  onCopyValue={onCopyValue}
                />
              ))
            : Object.entries(value).map(([k, v]) => (
                <TreeNode
                  key={k}
                  nodeKey={k}
                  value={v}
                  path={path === '$' ? k : `${path}.${k}`}
                  depth={depth + 1}
                  expandedPaths={expandedPaths}
                  togglePath={togglePath}
                  searchQuery={searchQuery}
                  copiedPath={copiedPath}
                  onCopyPath={onCopyPath}
                  onCopyValue={onCopyValue}
                />
              ))}
        </div>
      )}
    </div>
  )
}

export const JsonTreeViewer: React.FC<JsonTreeViewerProps> = ({
  data,
  initialExpandedDepth = 2,
  searchQuery: externalSearchQuery,
  onCopyPath,
  className = '',
}) => {
  const [internalSearchQuery, setInternalSearchQuery] = useState('')
  const [copiedPath, setCopiedPath] = useState<string | null>(null)
  const [copiedFull, setCopiedFull] = useState(false)

  const activeSearchQuery = externalSearchQuery !== undefined ? externalSearchQuery : internalSearchQuery

  // Expanded nodes set
  const initialPaths = useMemo(
    () => new Set(collectInitialExpandedPaths(data, initialExpandedDepth)),
    [data, initialExpandedDepth]
  )
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(initialPaths)

  const togglePath = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }, [])

  const handleExpandAll = () => {
    const all = collectAllPaths(data, '$')
    setExpandedPaths(new Set(all))
  }

  const handleCollapseAll = () => {
    setExpandedPaths(new Set())
  }

  const handleCopyPath = async (path: string) => {
    try {
      await navigator.clipboard.writeText(path)
      setCopiedPath(path)
      setTimeout(() => setCopiedPath(null), 2000)
      if (onCopyPath) onCopyPath(path)
    } catch (err) {
      console.error('Failed copying path to clipboard', err)
    }
  }

  const handleCopyValue = async (val: any) => {
    try {
      const text = typeof val === 'string' ? val : JSON.stringify(val, null, 2)
      await navigator.clipboard.writeText(text)
    } catch (err) {
      console.error('Failed copying value to clipboard', err)
    }
  }

  const handleCopyFull = async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(data, null, 2))
      setCopiedFull(true)
      setTimeout(() => setCopiedFull(false), 2000)
    } catch (err) {
      console.error('Failed copying full JSON to clipboard', err)
    }
  }

  const totalMatches = useMemo(
    () => countTotalMatches(data, activeSearchQuery),
    [data, activeSearchQuery]
  )

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Search & Global Controls Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3 bg-slate-50 dark:bg-slate-900/60 rounded-xl border border-slate-200/80 dark:border-slate-800">
        {externalSearchQuery === undefined && (
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={internalSearchQuery}
              onChange={(e) => setInternalSearchQuery(e.target.value)}
              placeholder="Search JSON keys or values..."
              className="w-full pl-9 pr-8 py-1.5 text-xs bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-brand-500 text-slate-900 dark:text-white"
            />
            {internalSearchQuery && (
              <button
                type="button"
                onClick={() => setInternalSearchQuery('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                aria-label="Clear search"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
        )}

        {activeSearchQuery && (
          <Badge variant="outline" className="text-xs font-mono">
            {totalMatches} {totalMatches === 1 ? 'match' : 'matches'}
          </Badge>
        )}

        <div className="flex items-center gap-2 ml-auto">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleExpandAll}
            className="h-8 text-xs gap-1.5 border-slate-200 dark:border-slate-800"
          >
            <Maximize2 className="w-3.5 h-3.5" />
            <span>Expand All</span>
          </Button>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCollapseAll}
            className="h-8 text-xs gap-1.5 border-slate-200 dark:border-slate-800"
          >
            <Minimize2 className="w-3.5 h-3.5" />
            <span>Collapse All</span>
          </Button>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopyFull}
            className="h-8 text-xs gap-1.5 border-slate-200 dark:border-slate-800"
          >
            {copiedFull ? (
              <Check className="w-3.5 h-3.5 text-emerald-500" />
            ) : (
              <Copy className="w-3.5 h-3.5" />
            )}
            <span>Copy JSON</span>
          </Button>
        </div>
      </div>

      {/* JSON Hierarchy Container */}
      <div className="p-4 bg-slate-50/50 dark:bg-slate-950/50 border border-slate-200 dark:border-slate-800 rounded-xl overflow-x-auto max-h-[600px] overflow-y-auto">
        <TreeNode
          value={data}
          path="$"
          depth={0}
          expandedPaths={expandedPaths}
          togglePath={togglePath}
          searchQuery={activeSearchQuery}
          copiedPath={copiedPath}
          onCopyPath={handleCopyPath}
          onCopyValue={handleCopyValue}
        />
      </div>
    </div>
  )
}
