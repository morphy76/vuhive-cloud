import React, { useState, useMemo } from 'react'
import {
  Layers,
  Plus,
  BookOpen,
  Search,
  X,
  ChevronRight,
  ChevronLeft,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VisuallyHidden } from '@/components/ui/visually-hidden'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { CreateSuiteDialog } from '@/components/dialogs/CreateSuiteDialog'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import { useRecipe } from '@/context/RecipeContext'
import { useSuites } from '@/hooks/use-suites'
import { SuiteDetailView } from '@/views/SuiteDetailView'
import type { TestSuite } from '@/types/suite'

export interface SuitesViewProps {
  initialSuites?: TestSuite[]
}

type SortOption = 'created-desc' | 'created-asc' | 'name-asc' | 'name-desc' | 'updated-desc'

export const SuitesView: React.FC<SuitesViewProps> = ({ initialSuites }) => {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [selectedSuiteId, setSelectedSuiteId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<SortOption>('created-desc')
  const [filterState, setFilterState] = useState<string>('ALL')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(5)

  const { openRecipe } = useRecipe()
  const { data: suites = [], isLoading } = useSuites(initialSuites)

  // Filtered and sorted suites
  const filteredSuites = useMemo(() => {
    return suites
      .filter((s) => {
        // State filter
        if (filterState !== 'ALL' && s.state !== filterState) {
          return false
        }
        // Search filter (name or description)
        if (!searchQuery.trim()) return true
        const q = searchQuery.toLowerCase()
        const matchesName = s.name.toLowerCase().includes(q)
        const matchesDesc = s.description?.toLowerCase().includes(q)
        const matchesId = s.id.toLowerCase().includes(q)
        return matchesName || matchesDesc || matchesId
      })
      .sort((a, b) => {
        switch (sortBy) {
          case 'name-asc':
            return a.name.localeCompare(b.name)
          case 'name-desc':
            return b.name.localeCompare(a.name)
          case 'created-asc':
            return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
          case 'created-desc':
            return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
          case 'updated-desc':
          default:
            return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
        }
      })
  }, [suites, searchQuery, filterState, sortBy])

  // Pagination calculations
  const totalItems = filteredSuites.length
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize))
  const safeCurrentPage = Math.min(currentPage, totalPages)
  const startIndex = (safeCurrentPage - 1) * pageSize
  const paginatedSuites = filteredSuites.slice(startIndex, startIndex + pageSize)

  // Selected suite for detail view
  const selectedSuite = useMemo(() => {
    if (!selectedSuiteId) return null
    return suites.find((s) => s.id === selectedSuiteId) || null
  }, [selectedSuiteId, suites])

  if (selectedSuite) {
    return (
      <SuiteDetailView
        suite={selectedSuite}
        onBack={() => setSelectedSuiteId(null)}
      />
    )
  }

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Test Suites
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Source Packages & Artifacts: browse, search, and manage scenario aggregates.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <OfflinePreviewBadge />
          <Button
            variant="outline"
            onClick={() => openRecipe('recipe-1')}
            className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
            aria-label="View Suites API Recipe"
          >
            <BookOpen className="w-4 h-4 text-brand-600 dark:text-brand-400" />
            <span className="hidden sm:inline">API Recipe</span>
          </Button>
          <Button
            onClick={() => setIsCreateOpen(true)}
            className="min-h-[44px] gap-2"
          >
            <Plus className="w-4 h-4" />
            <span>New Suite</span>
          </Button>
        </div>
      </div>

      {/* Search & Filter Controls */}
      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-4 shadow-xs space-y-3">
        <div className="flex flex-col md:flex-row gap-3">
          {/* Search Box */}
          <div className="relative flex-1">
            <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 pointer-events-none" />
            <input
              type="search"
              role="searchbox"
              aria-label="Search test suites"
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value)
                setCurrentPage(1)
              }}
              placeholder="Search by suite name, description, or ID..."
              className="w-full pl-10 pr-9 py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 min-h-[44px]"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => setSearchQuery('')}
                aria-label="Clear search input"
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 p-1"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>

          {/* Sort & State Filters */}
          <div className="flex flex-wrap items-center gap-2.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="sort-select"
                className="text-xs font-medium text-slate-500 dark:text-slate-400"
              >
                Sort:
              </label>
              <select
                id="sort-select"
                aria-label="Sort test suites"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as SortOption)}
                className="rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 min-h-[44px]"
              >
                <option value="created-desc">Newest Created</option>
                <option value="created-asc">Oldest Created</option>
                <option value="name-asc">Name (A to Z)</option>
                <option value="name-desc">Name (Z to A)</option>
                <option value="updated-desc">Recently Updated</option>
              </select>
            </div>

            <div className="flex items-center gap-1.5">
              <label
                htmlFor="state-filter"
                className="text-xs font-medium text-slate-500 dark:text-slate-400"
              >
                State:
              </label>
              <select
                id="state-filter"
                aria-label="Filter by state"
                value={filterState}
                onChange={(e) => {
                  setFilterState(e.target.value)
                  setCurrentPage(1)
                }}
                className="rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50 px-3 py-2 text-xs font-medium text-slate-700 dark:text-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 min-h-[44px]"
              >
                <option value="ALL">All States</option>
                <option value="ACTIVE">Active</option>
                <option value="DRAFT">Draft</option>
                <option value="ARCHIVED">Archived</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* Catalog Display */}
      {isLoading ? (
        <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-12 text-center text-sm text-slate-500">
          Loading test suites catalog...
        </div>
      ) : paginatedSuites.length === 0 ? (
        <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-12 text-center">
          <Layers className="w-12 h-12 mx-auto text-slate-300 dark:text-slate-700 mb-3" />
          <h2 className="text-base font-semibold text-slate-800 dark:text-slate-200">
            No Test Suites Found
          </h2>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1 max-w-md mx-auto">
            {searchQuery
              ? `No test suites matched "${searchQuery}". Try revising your search or clearing filters.`
              : 'No test suites configured in the cluster. Create your first suite to get started.'}
          </p>
          {searchQuery && (
            <Button
              variant="outline"
              onClick={() => {
                setSearchQuery('')
                setFilterState('ALL')
              }}
              className="mt-4 min-h-[44px]"
            >
              Clear Search
            </Button>
          )}
        </div>
      ) : (
        <div className="space-y-4">
          {/* Desktop & Tablet Table (Hidden on small mobile screens) */}
          <div className="hidden md:block rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
                <caption>
                  <VisuallyHidden>Test suites with build artifacts and target architectures</VisuallyHidden>
                </caption>
                <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
                  <tr>
                    <th scope="col" className="px-6 py-4">Suite Name</th>
                    <th scope="col" className="px-6 py-4">
                      <div className="flex items-center gap-1.5">
                        <span>Status</span>
                        <HelpTooltip
                          text="Lifecycle state of the test suite (ACTIVE or DRAFT)."
                          label="Help for status column"
                        />
                      </div>
                    </th>
                    <th scope="col" className="px-6 py-4">
                      <div className="flex items-center gap-1.5">
                        <span>Build Artifact</span>
                        <HelpTooltip
                          text="Cross-compilation status of scenario binary."
                          label="Help for build artifact column"
                        />
                      </div>
                    </th>
                    <th scope="col" className="px-6 py-4">
                      <div className="flex items-center gap-1.5">
                        <span>Target Arch</span>
                        <HelpTooltip
                          text="Target CPU architectures compiled statically for runner container execution."
                          label="Help for target architecture column"
                        />
                      </div>
                    </th>
                    <th scope="col" className="px-6 py-4">Updated</th>
                    <th scope="col" className="px-6 py-4 text-right">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {paginatedSuites.map((s) => (
                    <tr
                      key={s.id}
                      onClick={() => setSelectedSuiteId(s.id)}
                      className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors cursor-pointer"
                    >
                      <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                        <div className="flex items-center gap-3">
                          <Layers className="w-5 h-5 text-brand-500 flex-shrink-0" />
                          <div>
                            <div className="font-semibold">{s.name}</div>
                            {s.description && (
                              <div className="text-xs text-slate-400 truncate max-w-sm mt-0.5">
                                {s.description}
                              </div>
                            )}
                            <div className="text-[11px] text-slate-400 font-mono mt-0.5">{s.id}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant={s.state === 'ACTIVE' ? 'success' : 'default'}>
                          {s.state}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant={s.buildStatus === 'READY' ? 'info' : 'default'}>
                          {s.buildStatus || 'PENDING'}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex gap-1.5 flex-wrap">
                          {(s.platforms || ['linux/amd64']).map((p) => (
                            <span
                              key={p}
                              className="px-2 py-0.5 rounded-md text-xs font-mono bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300"
                            >
                              {p}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-6 py-4 text-xs font-mono">
                        {new Date(s.updatedAt).toLocaleDateString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation()
                            setSelectedSuiteId(s.id)
                          }}
                          className="min-h-[36px] text-brand-600 dark:text-brand-400"
                          aria-label={`View details for ${s.name}`}
                        >
                          <span>Inspect</span>
                          <ChevronRight className="w-4 h-4 ml-1" />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Mobile Card List (Visible on mobile screens) */}
          <div className="block md:hidden space-y-3">
            {paginatedSuites.map((s) => (
              <div
                key={s.id}
                data-testid="suite-mobile-card"
                onClick={() => setSelectedSuiteId(s.id)}
                className="p-4 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs space-y-3 cursor-pointer active:scale-[0.99] transition-transform"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-2.5">
                    <div className="p-2 rounded-xl bg-brand-50 dark:bg-brand-950 text-brand-600 dark:text-brand-400">
                      <Layers className="w-5 h-5" />
                    </div>
                    <div>
                      <h2 className="font-semibold text-base text-slate-900 dark:text-white">
                        {s.name}
                      </h2>
                      <span className="text-xs font-mono text-slate-400">{s.id}</span>
                    </div>
                  </div>
                  <Badge variant={s.state === 'ACTIVE' ? 'success' : 'default'}>
                    {s.state}
                  </Badge>
                </div>

                {s.description && (
                  <p className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2">
                    {s.description}
                  </p>
                )}

                <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-slate-100 dark:border-slate-800 text-xs">
                  <div className="flex gap-1.5 flex-wrap">
                    {(s.platforms || ['linux/amd64']).map((p) => (
                      <span
                        key={p}
                        className="px-2 py-0.5 rounded-md text-[11px] font-mono bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300"
                      >
                        {p}
                      </span>
                    ))}
                  </div>

                  <div className="flex items-center text-brand-600 dark:text-brand-400 font-medium min-h-[44px]">
                    <span>View details</span>
                    <ChevronRight className="w-4 h-4 ml-0.5" />
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Pagination Controls */}
          <div className="flex flex-col sm:flex-row items-center justify-between gap-3 px-2 py-3">
            <div className="text-xs text-slate-500 dark:text-slate-400">
              Showing <span className="font-medium text-slate-900 dark:text-white">{startIndex + 1}</span> to{' '}
              <span className="font-medium text-slate-900 dark:text-white">
                {Math.min(startIndex + pageSize, totalItems)}
              </span>{' '}
              of <span className="font-medium text-slate-900 dark:text-white">{totalItems}</span> suites
            </div>

            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1 mr-2">
                <span className="text-xs text-slate-500">Per page:</span>
                <select
                  aria-label="Items per page"
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value))
                    setCurrentPage(1)
                  }}
                  className="rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-2 py-1 text-xs text-slate-700 dark:text-slate-300 min-h-[36px]"
                >
                  <option value={5}>5</option>
                  <option value={10}>10</option>
                  <option value={20}>20</option>
                </select>
              </div>

              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                disabled={safeCurrentPage <= 1}
                className="min-h-[44px] px-3"
                aria-label="Previous page"
              >
                <ChevronLeft className="w-4 h-4 mr-1" />
                <span>Prev</span>
              </Button>
              <span className="text-xs text-slate-500 px-2">
                {safeCurrentPage} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                disabled={safeCurrentPage >= totalPages}
                className="min-h-[44px] px-3"
                aria-label="Next page"
              >
                <span>Next</span>
                <ChevronRight className="w-4 h-4 ml-1" />
              </Button>
            </div>
          </div>
        </div>
      )}

      <CreateSuiteDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
        existingSuites={suites}
      />
    </div>
  )
}
