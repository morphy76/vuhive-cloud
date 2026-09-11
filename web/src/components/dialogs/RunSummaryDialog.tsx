import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { RunSummaryDashboard } from '@/components/runs/RunSummaryDashboard'
import type { HistoricalRun } from '@/types/suite'

export interface RunSummaryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  run: HistoricalRun | null
}

export const RunSummaryDialog: React.FC<RunSummaryDialogProps> = ({
  open,
  onOpenChange,
  run,
}) => {
  if (!run) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-4xl max-h-[90vh] overflow-y-auto p-6"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader className="sr-only">
          <DialogTitle>Run Summary Dashboard</DialogTitle>
          <DialogDescription>
            Performance and SLA compliance executive summary for run {run.id}
          </DialogDescription>
        </DialogHeader>

        <RunSummaryDashboard run={run} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  )
}
