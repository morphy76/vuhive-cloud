import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { VirtualizedLogViewer } from '@/components/logs/VirtualizedLogViewer'
import { useRunLogs } from '@/hooks/use-runs'
import type { HistoricalRun } from '@/types/suite'

export interface ExecutionLogDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  run: HistoricalRun | null
}

export const ExecutionLogDialog: React.FC<ExecutionLogDialogProps> = ({
  open,
  onOpenChange,
  run,
}) => {
  if (!run) return null

  const isActive = run.status === 'RUNNING' || run.status === 'QUEUED'
  // Auto-poll logs every 3 seconds while test is active
  const { data: logs = '', isLoading } = useRunLogs(open ? run.id : undefined, isActive ? 3000 : false)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-6xl w-[95vw] h-[85vh] max-h-[90vh] p-0 flex flex-col overflow-hidden bg-slate-950 border-slate-800"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader className="sr-only">
          <DialogTitle>Execution Log Viewer</DialogTitle>
          <DialogDescription>
            Live ANSI execution log output for test run {run.id}
          </DialogDescription>
        </DialogHeader>

        <VirtualizedLogViewer
          logs={logs}
          runId={run.id}
          title="Container Execution Logs (run.log)"
          subtitle={`Status: ${run.status} • Job: ${run.k8sJobName || 'vuhive-run'}`}
          isLoading={isLoading}
          onClose={() => onOpenChange(false)}
          className="h-full border-0 rounded-none"
        />
      </DialogContent>
    </Dialog>
  )
}
