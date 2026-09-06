import React, { useState } from 'react'
import { CalendarClock, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VisuallyHidden } from '@/components/ui/visually-hidden'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { CreateScheduleDialog } from '@/components/dialogs/CreateScheduleDialog'

export const SchedulesView: React.FC = () => {
  const [isCreateOpen, setIsCreateOpen] = useState(false)

  const sampleSchedules = [
    {
      id: 'sched-nightly-soak',
      name: 'Nightly Soak Test (2h)',
      cron: '0 2 * * *',
      suite: 'Checkout & Payment Stress Test',
      nextRun: 'Tonight at 02:00 UTC',
      status: 'ACTIVE',
    },
    {
      id: 'sched-hourly-health',
      name: 'Hourly Performance Canary',
      cron: '0 * * * *',
      suite: 'Product Catalog High Throughput',
      nextRun: 'In 48 minutes',
      status: 'ACTIVE',
    },
    {
      id: 'sched-weekend-scale',
      name: 'Weekend Massive Concurrency',
      cron: '0 4 * * 6',
      suite: 'OAuth2 Token Grant Barrier Test',
      nextRun: 'Saturday at 04:00 UTC',
      status: 'ACTIVE',
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Cron Schedules
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Native Kubernetes CronJob Schedules: automate recurring performance verification runs.
          </p>
        </div>

        <Button
          onClick={() => setIsCreateOpen(true)}
          className="min-h-[44px] gap-2"
        >
          <Plus className="w-4 h-4" />
          <span>New Schedule</span>
        </Button>
      </div>

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
                      text="State of the native Kubernetes CronJob (ACTIVE or SUSPENDED)."
                      label="Help for status column"
                    />
                  </div>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
              {sampleSchedules.map((s) => (
                <tr key={s.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors">
                  <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                    <div className="flex items-center gap-3">
                      <CalendarClock className="w-5 h-5 text-brand-500 flex-shrink-0" />
                      <div>
                        <div>{s.name}</div>
                        <div className="text-xs text-slate-400 font-mono">{s.id}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <code className="px-2.5 py-1 rounded-md text-xs font-mono bg-slate-100 dark:bg-slate-800 text-slate-800 dark:text-slate-200">
                      {s.cron}
                    </code>
                  </td>
                  <td className="px-6 py-4 text-xs font-medium text-slate-700 dark:text-slate-300">
                    {s.suite}
                  </td>
                  <td className="px-6 py-4 text-xs font-mono">{s.nextRun}</td>
                  <td className="px-6 py-4">
                    <Badge variant="success">{s.status}</Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <CreateScheduleDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />
    </div>
  )
}
