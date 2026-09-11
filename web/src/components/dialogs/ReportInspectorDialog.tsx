import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { SummaryReportInspector } from '@/components/runs/SummaryReportInspector'
import type { HistoricalRun } from '@/types/suite'

export interface ReportInspectorDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  run: HistoricalRun | null
}

export const ReportInspectorDialog: React.FC<ReportInspectorDialogProps> = ({
  open,
  onOpenChange,
  run,
}) => {
  if (!run) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-6xl w-[95vw] max-h-[90vh] p-6 overflow-y-auto"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader className="sr-only">
          <DialogTitle>Telemetry Report Inspector</DialogTitle>
          <DialogDescription>
            Detailed summary.json telemetry breakdown and interactive JSON tree for run {run.id}
          </DialogDescription>
        </DialogHeader>

        <SummaryReportInspector
          runId={run.id}
          onClose={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  )
}
