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
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { InfoBadge } from '@/components/help/InfoBadge'

export interface CreateScheduleDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const CreateScheduleDialog: React.FC<CreateScheduleDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const [scheduleName, setScheduleName] = React.useState('')
  const [cronExpr, setCronExpr] = React.useState('0 2 * * *')

  const presets = [
    { label: 'Hourly', expr: '0 * * * *', desc: 'At minute 0 of every hour' },
    { label: 'Nightly', expr: '0 2 * * *', desc: 'Every day at 02:00 UTC' },
    { label: 'Weekly', expr: '0 4 * * 6', desc: 'Saturdays at 04:00 UTC' },
  ]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-lg"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>New Cron Schedule</DialogTitle>
          <DialogDescription>
            Schedule recurring load test executions as native Kubernetes CronJobs.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="schedule-name"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Schedule Name
              </label>
              <HelpTooltip
                text="A descriptive name for your recurring schedule (e.g. Nightly Soak Test)."
                label="Help for schedule name"
              />
            </div>
            <input
              id="schedule-name"
              type="text"
              value={scheduleName}
              onChange={(e) => setScheduleName(e.target.value)}
              placeholder="e.g. Nightly Soak Test"
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            />
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <label
                htmlFor="cron-expression"
                className="text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                Cron Expression
              </label>
              <HelpTooltip
                text="Standard 5-field CRON expression (minute, hour, day-of-month, month, day-of-week) executed in UTC timezone by native Kubernetes CronJobs."
                label="Help for cron expression"
              />
            </div>
            <input
              id="cron-expression"
              type="text"
              value={cronExpr}
              onChange={(e) => setCronExpr(e.target.value)}
              className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 font-mono focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
            />
          </div>

          <div className="space-y-1.5">
            <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
              Quick Presets
            </span>
            <div className="flex gap-2 flex-wrap">
              {presets.map((p) => (
                <button
                  key={p.label}
                  type="button"
                  aria-label={`Preset ${p.label}`}
                  onClick={() => setCronExpr(p.expr)}
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors border ${
                    cronExpr === p.expr
                      ? 'bg-brand-50 border-brand-300 text-brand-700 dark:bg-brand-950/60 dark:border-brand-800 dark:text-brand-300'
                      : 'bg-slate-50 border-slate-200 text-slate-600 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
                  }`}
                >
                  {p.label} ({p.expr})
                </button>
              ))}
            </div>
          </div>

          <InfoBadge
            variant="info"
            title="UTC Timezone Execution"
          >
            Native Kubernetes CronJobs evaluate schedules strictly in UTC. Account for local
            daylight saving time (DST) shifts when defining launch windows.
          </InfoBadge>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="min-h-[40px]"
          >
            Cancel
          </Button>
          <Button
            onClick={() => onOpenChange(false)}
            className="min-h-[40px]"
          >
            Save Schedule
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
