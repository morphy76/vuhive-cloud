import React, { useState, useMemo } from 'react'
import {
  CalendarClock,
  Plus,
  BookOpen,
  Search,
  Play,
  Pause,
  Trash2,
  History,
  Clock,
  Loader2,
  Copy,
  Check,
  AlertTriangle,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VisuallyHidden } from '@/components/ui/visually-hidden'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { CreateScheduleDialog } from '@/components/dialogs/CreateScheduleDialog'
import { ScheduleRunsDrawer } from '@/components/schedules/ScheduleRunsDrawer'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { useRecipe } from '@/context/RecipeContext'
import { useSchedules, useUpdateSchedule, useDeleteSchedule } from '@/hooks/use-schedules'
import { useSuites } from '@/hooks/use-suites'
import { useTriggerRun } from '@/hooks/use-runs'
import { useToast } from '@/hooks/use-toast'
import { describeCron, getNextCronRun } from '@/lib/cron-utils'
import type { Schedule } from '@/types/schedule'

export const SchedulesView: React.FC = () => {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [selectedScheduleForHistory, setSelectedScheduleForHistory] = useState<Schedule | null>(null)
  const [isHistoryOpen, setIsHistoryOpen] = useState(false)
  const [deleteScheduleTarget, setDeleteScheduleTarget] = useState<Schedule | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<'ALL' | 'ACTIVE' | 'PAUSED'>('ALL')
  const [pendingActionId, setPendingActionId] = useState<string | null>(null)

  const { openRecipe } = useRecipe()
  const { toast } = useToast()

  const { data: schedules = [], isLoading } = useSchedules()
  const { data: suites = [] } = useSuites()

  const updateScheduleMutation = useUpdateSchedule()
  const deleteScheduleMutation = useDeleteSchedule()
  const triggerRunMutation = useTriggerRun()

  // Map suite IDs to suite names for fast lookup
  const suiteMap = useMemo(() => {
    const map = new Map<string, string>()
    for (const s of suites) {
      map.set(s.id, s.name)
    }
    return map
  }, [suites])

  // Filtered schedules
  const filteredSchedules = useMemo(() => {
    return schedules.filter((s) => {
      // Status filter
      if (statusFilter === 'ACTIVE' && !s.isActive) return false
      if (statusFilter === 'PAUSED' && s.isActive) return false

      // Search filter
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase()
        const nameMatch = s.name.toLowerCase().includes(query)
        const cronMatch = s.cronExpression.toLowerCase().includes(query)
        const suiteName = suiteMap.get(s.suiteId)?.toLowerCase() || ''
        const suiteMatch = suiteName.includes(query)
        return nameMatch || cronMatch || suiteMatch
      }

      return true
    })
  }, [schedules, statusFilter, searchQuery, suiteMap])

  // Copy helper
  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
    toast({
      title: 'Copied to clipboard',
      description: text,
    })
  }

  // Handle Pause / Resume toggle
  const handleToggleActive = async (schedule: Schedule) => {
    setPendingActionId(schedule.id)
    const nextActive = !schedule.isActive
    try {
      await updateScheduleMutation.mutateAsync({
        id: schedule.id,
        data: { is_active: nextActive },
      })
      toast({
        title: nextActive ? 'Schedule Resumed' : 'Schedule Paused',
        description: `${schedule.name} has been ${nextActive ? 'resumed' : 'suspended'}.`,
      })
    } catch (err: any) {
      toast({
        title: 'Update failed',
        description: err.message || 'Failed to update schedule status',
        variant: 'destructive',
      })
    } finally {
      setPendingActionId(null)
    }
  }

  // Handle Run Now trigger
  const handleRunNow = async (schedule: Schedule) => {
    setPendingActionId(schedule.id)
    try {
      const run = await triggerRunMutation.mutateAsync({
        suite_id: schedule.suiteId,
        artifact_id: schedule.artifactId,
        configuration_id: schedule.configurationId,
        runner_profile_id: schedule.runnerProfileId,
      })
      toast({
        title: 'Ad-hoc run triggered',
        description: `Dispatched test run ${run.id.slice(0, 8)}... from schedule ${schedule.name}`,
      })
    } catch (err: any) {
      toast({
        title: 'Trigger run failed',
        description: err.message || 'Failed to dispatch ad-hoc test run',
        variant: 'destructive',
      })
    } finally {
      setPendingActionId(null)
    }
  }

  // Handle Confirm Delete
  const handleConfirmDelete = async () => {
    if (!deleteScheduleTarget) return
    setPendingActionId(deleteScheduleTarget.id)
    try {
      await deleteScheduleMutation.mutateAsync(deleteScheduleTarget.id)
      toast({
        title: 'Schedule Deleted',
        description: `Schedule ${deleteScheduleTarget.name} and native CronJob removed.`,
      })
      setDeleteScheduleTarget(null)
    } catch (err: any) {
      toast({
        title: 'Deletion failed',
        description: err.message || 'Failed to delete schedule',
        variant: 'destructive',
      })
    } finally {
      setPendingActionId(null)
    }
  }

  // Open History Drawer
  const handleOpenHistory = (schedule: Schedule) => {
    setSelectedScheduleForHistory(schedule)
    setIsHistoryOpen(true)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Cron Schedules
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Native Kubernetes CronJob Schedules: automate recurring performance verification runs with human-readable cadence builders.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <OfflinePreviewBadge />
          <Button
            variant="outline"
            onClick={() => openRecipe('recipe-5')}
            className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
            aria-label="View Schedules API Recipe"
          >
            <BookOpen className="w-4 h-4 text-brand-600 dark:text-brand-400" />
            <span className="hidden sm:inline">API Recipe</span>
          </Button>
          <Button
            onClick={() => setIsCreateOpen(true)}
            className="min-h-[44px] gap-2"
          >
            <Plus className="w-4 h-4" />
            <span>New Schedule</span>
          </Button>
        </div>
      </div>

      {/* Filter & Search Toolbar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search schedules by name, suite, or cron..."
            aria-label="Search schedules"
            className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 pl-10 pr-4 py-2 text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
          />
        </div>

        <div className="flex items-center gap-2">
          <label htmlFor="status-filter" className="text-xs text-slate-500 dark:text-slate-400 font-medium">
            Status:
          </label>
          <select
            id="status-filter"
            aria-label="Filter by status"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as any)}
            className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-2 text-xs text-slate-700 dark:text-slate-300 font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
          >
            <option value="ALL">All Schedules</option>
            <option value="ACTIVE">Active Only</option>
            <option value="PAUSED">Paused Only</option>
          </select>
        </div>
      </div>

      {/* Table Container */}
      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
            <caption>
              <VisuallyHidden>Kubernetes CronJob schedules with cron expressions and trigger times</VisuallyHidden>
            </caption>
            <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th scope="col" className="px-6 py-4">Schedule Name</th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Cron Expression</span>
                    <HelpTooltip
                      text="Standard 5-field CRON expression (minute, hour, day, month, weekday) executed in UTC timezone."
                      label="Help for cron expression column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">Target Suite</th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Next Trigger</span>
                    <HelpTooltip
                      text="Calculated next trigger timestamp according to the cluster UTC clock."
                      label="Help for next trigger column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Status</span>
                    <HelpTooltip
                      text="State of the native Kubernetes CronJob (ACTIVE or PAUSED/SUSPENDED)."
                      label="Help for status column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-slate-400">
                    <Loader2 className="w-6 h-6 animate-spin mx-auto text-brand-500 mb-2" />
                    <span className="text-xs">Loading schedules...</span>
                  </td>
                </tr>
              ) : filteredSchedules.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-slate-400">
                    <CalendarClock className="w-8 h-8 mx-auto text-slate-300 dark:text-slate-600 mb-2" />
                    <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                      No schedules found
                    </p>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                      {searchQuery
                        ? 'Try adjusting your search query or status filter.'
                        : 'Create your first recurring Kubernetes CronJob schedule.'}
                    </p>
                  </td>
                </tr>
              ) : (
                filteredSchedules.map((s) => {
                  const suiteName = suiteMap.get(s.suiteId) || s.suiteId
                  const naturalCadence = describeCron(s.cronExpression)
                  const nextRun = getNextCronRun(s.cronExpression)
                  const formattedNextRun = nextRun
                    ? nextRun.toUTCString().replace('GMT', 'UTC')
                    : 'Invalid schedule'

                  const isRowPending = pendingActionId === s.id

                  return (
                    <tr
                      key={s.id}
                      className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
                    >
                      {/* Name & ID */}
                      <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                        <div className="flex items-center gap-3">
                          <CalendarClock className="w-5 h-5 text-brand-500 flex-shrink-0" />
                          <div>
                            <div className="font-semibold text-slate-900 dark:text-white">
                              {s.name}
                            </div>
                            <div className="flex items-center gap-1.5 text-xs text-slate-400 font-mono">
                              <span>{s.id}</span>
                              <button
                                type="button"
                                aria-label={`Copy ID for ${s.name}`}
                                onClick={() => handleCopy(s.id, s.id)}
                                className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                              >
                                {copiedId === s.id ? (
                                  <Check className="w-3 h-3 text-emerald-500" />
                                ) : (
                                  <Copy className="w-3 h-3" />
                                )}
                              </button>
                            </div>
                          </div>
                        </div>
                      </td>

                      {/* Cron Expression & Natural Cadence */}
                      <td className="px-6 py-4">
                        <div className="space-y-1">
                          <div className="flex items-center gap-1.5">
                            <code className="px-2 py-0.5 rounded-md text-xs font-mono bg-slate-100 dark:bg-slate-800 text-slate-800 dark:text-slate-200">
                              {s.cronExpression}
                            </code>
                          </div>
                          <div className="text-[11px] text-slate-500 dark:text-slate-400">
                            {naturalCadence}
                          </div>
                        </div>
                      </td>

                      {/* Target Suite */}
                      <td className="px-6 py-4 text-xs font-medium text-slate-700 dark:text-slate-300">
                        {suiteName}
                      </td>

                      {/* Next Trigger */}
                      <td className="px-6 py-4 text-xs font-mono">
                        {s.isActive ? (
                          <div className="flex items-center gap-1 text-slate-700 dark:text-slate-300">
                            <Clock className="w-3.5 h-3.5 text-slate-400" />
                            <span>{formattedNextRun}</span>
                          </div>
                        ) : (
                          <span className="text-amber-600 dark:text-amber-400 font-sans italic text-[11px]">
                            Paused (suspended)
                          </span>
                        )}
                      </td>

                      {/* Status Badge */}
                      <td className="px-6 py-4">
                        <Badge variant={s.isActive ? 'success' : 'outline'}>
                          {s.isActive ? 'ACTIVE' : 'PAUSED'}
                        </Badge>
                      </td>

                      {/* Quick Actions */}
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {/* Run Now (Ad-Hoc Trigger) */}
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={isRowPending}
                            onClick={() => handleRunNow(s)}
                            className="h-8 px-2 text-xs gap-1 text-brand-600 dark:text-brand-400 hover:text-brand-700 hover:bg-brand-50 dark:hover:bg-brand-950/40"
                            aria-label={`Run now for schedule ${s.name}`}
                          >
                            <Play className="w-3.5 h-3.5 fill-current" />
                            <span className="hidden md:inline">Run Now</span>
                          </Button>

                          {/* Pause / Resume */}
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={isRowPending}
                            onClick={() => handleToggleActive(s)}
                            className={`h-8 px-2 text-xs gap-1 ${
                              s.isActive
                                ? 'text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-950/40'
                                : 'text-emerald-600 dark:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-950/40'
                            }`}
                            aria-label={
                              s.isActive
                                ? `Pause schedule ${s.name}`
                                : `Resume schedule ${s.name}`
                            }
                          >
                            {s.isActive ? (
                              <>
                                <Pause className="w-3.5 h-3.5" />
                                <span className="hidden md:inline">Pause</span>
                              </>
                            ) : (
                              <>
                                <Play className="w-3.5 h-3.5" />
                                <span className="hidden md:inline">Resume</span>
                              </>
                            )}
                          </Button>

                          {/* Execution History */}
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleOpenHistory(s)}
                            className="h-8 px-2 text-xs gap-1 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800"
                            aria-label={`View execution history for ${s.name}`}
                          >
                            <History className="w-3.5 h-3.5" />
                            <span className="hidden lg:inline">History</span>
                          </Button>

                          {/* Delete */}
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={isRowPending}
                            onClick={() => setDeleteScheduleTarget(s)}
                            className="h-8 px-2 text-xs text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/40"
                            aria-label={`Delete schedule ${s.name}`}
                          >
                            <Trash2 className="w-3.5 h-3.5" />
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

      {/* Create Schedule Dialog */}
      <CreateScheduleDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />

      {/* Schedule Runs History Drawer */}
      <ScheduleRunsDrawer
        schedule={selectedScheduleForHistory}
        open={isHistoryOpen}
        onOpenChange={setIsHistoryOpen}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={Boolean(deleteScheduleTarget)}
        onOpenChange={(open) => !open && setDeleteScheduleTarget(null)}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-red-600 dark:text-red-400">
              <AlertTriangle className="w-5 h-5" />
              <span>Delete Cron Schedule</span>
            </DialogTitle>
            <DialogDescription className="text-xs text-slate-600 dark:text-slate-400 pt-2">
              Are you sure you want to delete schedule{' '}
              <span className="font-semibold text-slate-900 dark:text-slate-100">
                &quot;{deleteScheduleTarget?.name}&quot;
              </span>
              ? This action will remove the schedule from the database and immediately delete the
              underlying Kubernetes CronJob{' '}
              <code className="font-mono text-slate-800 dark:text-slate-200">
                {deleteScheduleTarget?.k8sCronJobName}
              </code>
              . Historical test runs will not be deleted.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter className="pt-3">
            <Button
              variant="outline"
              onClick={() => setDeleteScheduleTarget(null)}
              className="min-h-[40px]"
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deleteScheduleMutation.isPending}
              onClick={handleConfirmDelete}
              className="min-h-[40px] gap-2"
            >
              {deleteScheduleMutation.isPending && (
                <Loader2 className="w-4 h-4 animate-spin" />
              )}
              <span>Confirm Delete</span>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
