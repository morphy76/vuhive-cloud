import React, { useState, useMemo } from 'react'
import {
  Sliders,
  Plus,
  BookOpen,
  Search,
  Cpu,
  HardDrive,
  Shield,
  Edit2,
  Trash2,
  Copy,
  CheckCircle2,
  Server,
  Zap,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VisuallyHidden } from '@/components/ui/visually-hidden'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import { useRecipe } from '@/context/RecipeContext'
import { useProfiles } from '@/hooks/use-profiles'
import { ProfileBuilderDialog } from '@/components/profiles/ProfileBuilderDialog'
import { DeleteProfileDialog } from '@/components/profiles/DeleteProfileDialog'
import { parseCPUTomilli, parseMemoryToMiB } from '@/lib/k8s-resources'
import type { RunnerProfile } from '@/types/profile'

export const ProfilesView: React.FC = () => {
  const { openRecipe } = useRecipe()
  const { data: profiles = [], isLoading } = useProfiles()

  const [searchQuery, setSearchQuery] = useState('')
  const [isBuilderOpen, setIsBuilderOpen] = useState(false)
  const [selectedProfileForEdit, setSelectedProfileForEdit] = useState<RunnerProfile | null>(null)
  const [selectedProfileForDelete, setSelectedProfileForDelete] = useState<RunnerProfile | null>(null)

  // Filter profiles based on search query
  const filteredProfiles = useMemo(() => {
    const q = searchQuery.toLowerCase().trim()
    if (!q) return profiles
    return profiles.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        (p.description && p.description.toLowerCase().includes(q)) ||
        (p.runner_image && p.runner_image.toLowerCase().includes(q))
    )
  }, [profiles, searchQuery])

  // Summary Metrics
  const totalCount = profiles.length
  const dedicatedCount = profiles.filter(
    (p) =>
      Boolean(p.node_selector && Object.keys(p.node_selector).length > 0) ||
      Boolean(p.tolerations && p.tolerations.length > 0) ||
      Boolean(p.affinity?.node_selector_terms && p.affinity.node_selector_terms.length > 0)
  ).length
  const sandboxedCount = profiles.filter((p) => Boolean(p.runtime_class_name)).length

  const handleEdit = (profile: RunnerProfile) => {
    setSelectedProfileForEdit(profile)
    setIsBuilderOpen(true)
  }

  const handleDuplicate = (profile: RunnerProfile) => {
    // Clone without id to open in create mode with populated fields
    const duplicateProfile: RunnerProfile = {
      ...profile,
      id: '',
      name: `${profile.name}-copy`,
      description: `Copy of ${profile.name}: ${profile.description || ''}`,
    }
    setSelectedProfileForEdit(duplicateProfile)
    setIsBuilderOpen(true)
  }

  const handleDelete = (profile: RunnerProfile) => {
    setSelectedProfileForDelete(profile)
  }

  const handleOpenCreate = () => {
    setSelectedProfileForEdit(null)
    setIsBuilderOpen(true)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white flex items-center gap-2.5">
            <Sliders className="w-7 h-7 text-brand-600 dark:text-brand-400" />
            <span>Runner Profiles</span>
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Reusable Kubernetes compute environments: configure CPU/memory sizing, node affinities, and pod tolerations.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <OfflinePreviewBadge />
          <Button
            variant="outline"
            onClick={() => openRecipe('recipe-3')}
            className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
            aria-label="View Runner Profiles API Recipe"
          >
            <BookOpen className="w-4 h-4 text-brand-600 dark:text-brand-400" />
            <span className="hidden sm:inline">API Recipe</span>
          </Button>
          <Button onClick={handleOpenCreate} className="min-h-[44px] gap-2">
            <Plus className="w-4 h-4" />
            <span>New Profile</span>
          </Button>
        </div>
      </div>

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-4 sm:p-5 shadow-xs flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-50 text-brand-600 dark:bg-brand-950/50 dark:text-brand-400">
            <Server className="h-6 w-6" />
          </div>
          <div>
            <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">
              {totalCount}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">
              Registered Profiles
            </div>
          </div>
        </div>

        <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-4 sm:p-5 shadow-xs flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-50 text-indigo-600 dark:bg-indigo-950/50 dark:text-indigo-400">
            <Zap className="h-6 w-6" />
          </div>
          <div>
            <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">
              {dedicatedCount}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">
              Dedicated Node Schedulers
            </div>
          </div>
        </div>

        <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-4 sm:p-5 shadow-xs flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50 dark:text-emerald-400">
            <Shield className="h-6 w-6" />
          </div>
          <div>
            <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">
              {sandboxedCount}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 font-medium">
              Kernel-Isolated (gVisor)
            </div>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search runner profiles by name, description, or image..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            aria-label="Search profiles"
            className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 pl-10 pr-4 py-2.5 text-xs sm:text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
          />
        </div>
      </div>

      {/* Profiles Table */}
      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
            <caption>
              <VisuallyHidden>List of registered Kubernetes runner profiles</VisuallyHidden>
            </caption>
            <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th scope="col" className="px-6 py-4">Profile Name & Scope</th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>CPU (Req / Lim)</span>
                    <HelpTooltip
                      text="Minimum guaranteed CPU millicores vs maximum throttling limit."
                      label="Help for CPU limits"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Memory (Req / Lim)</span>
                    <HelpTooltip
                      text="Guaranteed memory allocation vs maximum hard eviction boundary."
                      label="Help for memory limits"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Scheduling Rules</span>
                    <HelpTooltip
                      text="Node selectors, affinity match expressions, and pod tolerations."
                      label="Help for scheduling rules"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800 font-normal">
              {isLoading ? (
                <tr>
                  <td colSpan={5} className="px-6 py-8 text-center text-xs text-slate-500">
                    Loading runner profiles...
                  </td>
                </tr>
              ) : filteredProfiles.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center">
                    <div className="flex flex-col items-center justify-center gap-2">
                      <Sliders className="w-8 h-8 text-slate-300 dark:text-slate-600" />
                      <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                        No runner profiles found
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {searchQuery
                          ? 'No profiles match your search criteria.'
                          : 'Create your first runner profile to configure load test environments.'}
                      </p>
                      <Button onClick={handleOpenCreate} size="sm" className="mt-2 gap-1.5">
                        <Plus className="w-3.5 h-3.5" />
                        <span>Build Profile</span>
                      </Button>
                    </div>
                  </td>
                </tr>
              ) : (
                filteredProfiles.map((p) => {
                  const nodeSelectorCount = Object.keys(p.node_selector || {}).length
                  const affinityCount = p.affinity?.node_selector_terms?.length || 0
                  const tolerationsCount = p.tolerations?.length || 0

                  const reqCpu = parseCPUTomilli(p.cpu_request)
                  const limCpu = parseCPUTomilli(p.cpu_limit)
                  const reqMem = parseMemoryToMiB(p.memory_request)
                  const limMem = parseMemoryToMiB(p.memory_limit)
                  const isGuaranteed =
                    reqCpu > 0 &&
                    limCpu > 0 &&
                    reqCpu === limCpu &&
                    reqMem > 0 &&
                    limMem > 0 &&
                    reqMem === limMem

                  return (
                    <tr
                      key={p.id}
                      className="hover:bg-slate-50/70 dark:hover:bg-slate-800/40 transition-colors"
                    >
                      <td className="px-6 py-4">
                        <div className="flex flex-col">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-slate-900 dark:text-slate-100">
                              {p.name}
                            </span>
                            {isGuaranteed && (
                              <Badge variant="success" className="text-[10px] py-0 px-1.5 gap-1">
                                <CheckCircle2 className="w-3 h-3" />
                                <span>Guaranteed</span>
                              </Badge>
                            )}
                            {p.runtime_class_name && (
                              <Badge variant="outline" className="text-[10px] py-0 px-1.5 border-emerald-500/40 text-emerald-600 dark:text-emerald-400">
                                {p.runtime_class_name}
                              </Badge>
                            )}
                          </div>
                          {p.description && (
                            <span className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-1">
                              {p.description}
                            </span>
                          )}
                          <span className="text-[11px] font-mono text-slate-400 dark:text-slate-500 mt-0.5">
                            Image: {p.runner_image || 'alpine:3.20'}
                          </span>
                        </div>
                      </td>

                      <td className="px-6 py-4 font-mono text-xs">
                        <div className="flex items-center gap-1.5">
                          <Cpu className="w-3.5 h-3.5 text-slate-400" />
                          <span className="font-medium text-slate-900 dark:text-slate-100">
                            {p.cpu_request}
                          </span>
                          <span className="text-slate-400">/</span>
                          <span className="text-slate-600 dark:text-slate-400">
                            {p.cpu_limit}
                          </span>
                        </div>
                      </td>

                      <td className="px-6 py-4 font-mono text-xs">
                        <div className="flex items-center gap-1.5">
                          <HardDrive className="w-3.5 h-3.5 text-slate-400" />
                          <span className="font-medium text-slate-900 dark:text-slate-100">
                            {p.memory_request}
                          </span>
                          <span className="text-slate-400">/</span>
                          <span className="text-slate-600 dark:text-slate-400">
                            {p.memory_limit}
                          </span>
                        </div>
                      </td>

                      <td className="px-6 py-4">
                        <div className="flex flex-wrap items-center gap-1.5 text-xs">
                          {nodeSelectorCount > 0 && (
                            <Badge variant="outline" className="text-[11px] font-mono">
                              {nodeSelectorCount} selector{nodeSelectorCount > 1 ? 's' : ''}
                            </Badge>
                          )}
                          {affinityCount > 0 && (
                            <Badge variant="outline" className="text-[11px] font-mono border-indigo-500/40 text-indigo-600 dark:text-indigo-400">
                              {affinityCount} affinity
                            </Badge>
                          )}
                          {tolerationsCount > 0 && (
                            <Badge variant="outline" className="text-[11px] font-mono border-amber-500/40 text-amber-600 dark:text-amber-400">
                              {tolerationsCount} toleration{tolerationsCount > 1 ? 's' : ''}
                            </Badge>
                          )}
                          {nodeSelectorCount === 0 && affinityCount === 0 && tolerationsCount === 0 && (
                            <span className="text-xs text-slate-400 italic">Default scheduler</span>
                          )}
                        </div>
                      </td>

                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => handleDuplicate(p)}
                            aria-label={`Duplicate profile ${p.name}`}
                            className="h-8 w-8 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
                          >
                            <Copy className="w-4 h-4" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => handleEdit(p)}
                            aria-label={`Edit profile ${p.name}`}
                            className="h-8 w-8 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
                          >
                            <Edit2 className="w-4 h-4" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => handleDelete(p)}
                            aria-label={`Delete profile ${p.name}`}
                            className="h-8 w-8 text-slate-400 hover:text-rose-600 dark:hover:text-rose-400"
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Profile Builder Modal */}
      <ProfileBuilderDialog
        open={isBuilderOpen}
        onOpenChange={setIsBuilderOpen}
        initialProfile={selectedProfileForEdit}
      />

      {/* Delete Confirmation Modal */}
      <DeleteProfileDialog
        open={Boolean(selectedProfileForDelete)}
        onOpenChange={(open) => !open && setSelectedProfileForDelete(null)}
        profile={selectedProfileForDelete}
      />
    </div>
  )
}
