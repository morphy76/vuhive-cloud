import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Play, CalendarClock, Clock, CheckCircle2, XCircle, Loader2 } from 'lucide-react'
import { useRuns, useTriggerRun } from '@/hooks/use-runs'
import type { Schedule } from '@/types/schedule'
import { describeCron } from '@/lib/cron-utils'

export interface ScheduleRunsDrawerProps {
  schedule: Schedule | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const ScheduleRunsDrawer: React.FC<ScheduleRunsDrawerProps> = ({
  schedule,
  open,
  onOpenChange,
}) => {
  const { data: runs = [], isLoading } = useRuns(
    schedule ? { scheduleId: schedule.id } : undefined
  )
  const triggerRunMutation = useTriggerRun()

  const handleRunNow = async () => {
    if (!schedule) return
    try {
      await triggerRunMutation.mutateAsync({
        suite_id: schedule.suiteId,
        artifact_id: schedule.artifactId,
        configuration_id: schedule.configurationId,
        runner_profile_id: schedule.runnerProfileId,
      })
    } catch (err) {
      console.error('Failed to trigger run from schedule:', err)
    }
  }

  if (!schedule) return null

  const naturalCadence = describeCron(schedule.cronExpression)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-2xl max-h-[85vh] flex flex-col"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader className="border-b border-slate-200 dark:border-slate-800 pb-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <DialogTitle className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2">
                <CalendarClock className="w-5 h-5 text-brand-500 flex-shrink-0" />
                <span>Execution History</span>
              </DialogTitle>
              <DialogDescription className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Historical test runs spawned by schedule{' '}
                <span className="font-semibold text-slate-700 dark:text-slate-300">
                  {schedule.name}
                </span>
              </DialogDescription>
            </div>

            <Button
              size="sm"
              disabled={triggerRunMutation.isPending}
              onClick={handleRunNow}
              className="gap-1.5 text-xs min-h-[36px]"
              aria-label="Trigger immediate run"
            >
              {triggerRunMutation.isPending ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5 fill-current" />
              )}
              <span>Run Now</span>
            </Button>
          </div>

          <div className="flex items-center gap-2 flex-wrap pt-2">
            <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded text-slate-800 dark:text-slate-200">
              {schedule.cronExpression}
            </code>
            <span className="text-xs text-slate-500 dark:text-slate-400">
              • {naturalCadence}
            </span>
            <Badge variant={schedule.isActive ? 'success' : 'outline'}>
              {schedule.isActive ? 'ACTIVE' : 'PAUSED'}
            </Badge>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4">
          {isLoading ? (
            <div className="py-12 flex flex-col items-center justify-center text-slate-400 gap-2">
              <Loader2 className="w-6 h-6 animate-spin text-brand-500" />
              <span className="text-xs">Loading execution history...</span>
            </div>
          ) : runs.length === 0 ? (
            <div className="py-12 text-center rounded-xl border border-dashed border-slate-200 dark:border-slate-800">
              <Clock className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
              <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                No runs executed yet
              </h4>
              <p className="text-xs text-slate-500 dark:text-slate-400 max-w-sm mx-auto mt-1">
                No executions have been recorded for this schedule. Click &quot;Run Now&quot; above to trigger an immediate execution or wait for the next scheduled trigger.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-200 dark:divide-slate-800 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
              {runs.map((r) => {
                const durationSec = r.durationMs ? Math.round(r.durationMs / 1000) : 0
                return (
                  <div
                    key={r.id}
                    className="p-3.5 hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-medium text-slate-900 dark:text-slate-100">
                          {r.id}
                        </span>
                        <Badge
                          variant={
                            r.status === 'COMPLETED'
                              ? 'success'
                              : r.status === 'RUNNING'
                              ? 'info'
                              : r.status === 'FAILED' || r.status === 'ABORTED'
                              ? 'error'
                              : 'outline'
                          }
                        >
                          {r.status}
                        </Badge>
                        {r.slaPassed !== undefined && (
                          <span
                            className={`inline-flex items-center gap-1 text-[11px] font-medium ${
                              r.slaPassed
                                ? 'text-emerald-600 dark:text-emerald-400'
                                : 'text-red-600 dark:text-red-400'
                            }`}
                          >
                            {r.slaPassed ? (
                              <CheckCircle2 className="w-3.5 h-3.5" />
                            ) : (
                              <XCircle className="w-3.5 h-3.5" />
                            )}
                            SLA {r.slaPassed ? 'Passed' : 'Failed'}
                          </span>
                        )}
                      </div>

                      <div className="text-[11px] text-slate-500 dark:text-slate-400">
                        {r.startedAt ? new Date(r.startedAt).toLocaleString() : 'Not started'}
                        {durationSec > 0 && ` • Duration: ${durationSec}s`}
                      </div>
                    </div>

                    {r.metrics && (
                      <div className="flex items-center gap-3 text-[11px] font-mono text-slate-600 dark:text-slate-300 bg-slate-50 dark:bg-slate-800 px-2.5 py-1.5 rounded-lg">
                        {r.metrics.avgTps !== undefined && (
                          <span>{r.metrics.avgTps} TPS</span>
                        )}
                        {r.metrics.p95DurationMs !== undefined && (
                          <span>p95: {r.metrics.p95DurationMs}ms</span>
                        )}
                        {r.metrics.errorRatePct !== undefined && (
                          <span>err: {(r.metrics.errorRatePct * 100).toFixed(1)}%</span>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
