import React, { useState } from 'react'
import { Search, X, ListCollapse, ChevronRight } from 'lucide-react'
import { TocHeading, filterHeadings } from '@/lib/markdown-utils'
import { cn } from '@/lib/utils'

export interface CookbookTocProps {
  headings: TocHeading[]
  activeId?: string
  onSelectHeading?: (id: string) => void
  className?: string
}

export const CookbookToc: React.FC<CookbookTocProps> = ({
  headings,
  activeId,
  onSelectHeading,
  className,
}) => {
  const [query, setQuery] = useState('')

  const filtered = filterHeadings(headings, query)

  const handleHeadingClick = (
    e: React.MouseEvent<HTMLAnchorElement>,
    id: string
  ) => {
    e.preventDefault()
    onSelectHeading?.(id)

    const element = document.getElementById(id)
    if (element) {
      element.scrollIntoView({ behavior: 'smooth' })
      window.history.pushState(null, '', `#${id}`)
    }
  }

  return (
    <nav
      aria-label="Table of contents"
      className={cn(
        'flex flex-col rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-4 shadow-xs',
        className
      )}
    >
      <div className="flex items-center gap-2 mb-3 pb-3 border-b border-slate-100 dark:border-slate-800">
        <ListCollapse className="w-4 h-4 text-brand-600 dark:text-brand-400" />
        <span className="text-sm font-semibold text-slate-900 dark:text-white">
          Table of Contents
        </span>
        <span className="ml-auto text-xs text-slate-400 font-mono">
          {headings.length}
        </span>
      </div>

      {/* Quick-Search Filter */}
      <div className="relative mb-3">
        <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Filter recipes..."
          aria-label="Filter recipes and sections"
          className="w-full rounded-xl bg-slate-50 dark:bg-slate-800/80 pl-8 pr-8 py-1.5 text-xs text-slate-900 dark:text-white border border-slate-200 dark:border-slate-700/60 focus:outline-none focus:ring-2 focus:ring-brand-500/40"
        />
        {query && (
          <button
            type="button"
            onClick={() => setQuery('')}
            aria-label="Clear filter"
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      {/* Heading List */}
      <div className="flex-1 overflow-y-auto space-y-1 pr-1 max-h-[calc(100vh-14rem)]">
        {filtered.length === 0 ? (
          <div className="py-6 text-center text-xs text-slate-400">
            No sections matching &ldquo;{query}&rdquo;
          </div>
        ) : (
          filtered.map((heading) => {
            const isActive = activeId === heading.id
            const indentClass =
              heading.level === 1
                ? 'font-semibold text-xs text-slate-900 dark:text-white mt-2'
                : heading.level === 2
                ? 'pl-2 text-xs font-medium'
                : 'pl-4 text-[11px] text-slate-500 dark:text-slate-400'

            return (
              <a
                key={heading.id}
                href={`#${heading.id}`}
                onClick={(e) => handleHeadingClick(e, heading.id)}
                aria-current={isActive ? 'true' : undefined}
                className={cn(
                  'flex items-center gap-1.5 py-1 px-2 rounded-lg transition-colors leading-normal group',
                  indentClass,
                  isActive
                    ? 'bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400 font-semibold'
                    : 'hover:bg-slate-50 dark:hover:bg-slate-800/60 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
                )}
              >
                {heading.level > 1 && (
                  <ChevronRight
                    className={cn(
                      'w-3 h-3 flex-shrink-0 transition-transform text-slate-400',
                      isActive ? 'rotate-90 text-brand-500' : 'group-hover:translate-x-0.5'
                    )}
                  />
                )}
                <span className="truncate">{heading.text}</span>
              </a>
            )
          })
        )}
      </div>
    </nav>
  )
}
