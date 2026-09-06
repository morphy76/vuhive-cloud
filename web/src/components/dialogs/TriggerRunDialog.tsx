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
import { Switch } from '@/components/ui/switch'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { InfoBadge } from '@/components/help/InfoBadge'

export interface TriggerRunDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const TriggerRunDialog: React.FC<TriggerRunDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const [pods, setPods] = React.useState('8')
  const [cpu, setCpu] = React.useState('1000m')
  const [memory, setMemory] = React.useState('1Gi')
  const [tolerations, setTolerations] = React.useState('dedicated=loadgen:NoSchedule')
  const [barrierEnabled, setBarrierEnabled] = React.useState(true)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-lg"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>Execute Test Run</DialogTitle>
          <DialogDescription>
            Dispatch an on-demand distributed load test run across Kubernetes worker pods.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-pods"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Runner Pods Count
                </label>
                <HelpTooltip
                  text="Number of parallel worker pods to spawn in Kubernetes."
                  label="Help for runner pods count"
                />
              </div>
              <input
                id="runner-pods"
                type="number"
                value={pods}
                onChange={(e) => setPods(e.target.value)}
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-cpu"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  CPU Millicores
                </label>
                <HelpTooltip
                  text="Kubernetes CPU allocation in millicores (e.g., 500m = 0.5 CPU, 1000m = 1 vCPU). Limits prevent noisy-neighbor throttling."
                  label="Help for cpu allocation"
                />
              </div>
              <input
                id="runner-cpu"
                type="text"
                value={cpu}
                onChange={(e) => setCpu(e.target.value)}
                placeholder="1000m"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-memory"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Memory Units
                </label>
                <HelpTooltip
                  text="Kubernetes memory allocation using binary SI units (e.g., 512Mi, 1Gi, 2Gi). Exceeding limits triggers OOMKill."
                  label="Help for memory allocation"
                />
              </div>
              <input
                id="runner-memory"
                type="text"
                value={memory}
                onChange={(e) => setMemory(e.target.value)}
                placeholder="1Gi"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center gap-1.5">
                <label
                  htmlFor="runner-tolerations"
                  className="text-xs font-semibold text-slate-700 dark:text-slate-300"
                >
                  Node Tolerations
                </label>
                <HelpTooltip
                  text="Kubernetes tolerations syntax (key=value:NoSchedule) allowing runner pods to schedule onto dedicated tainted load-generation worker nodes."
                  label="Help for node tolerations"
                />
              </div>
              <input
                id="runner-tolerations"
                type="text"
                value={tolerations}
                onChange={(e) => setTolerations(e.target.value)}
                placeholder="dedicated=loadgen:NoSchedule"
                className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3.5 py-2 text-sm text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 font-mono text-xs"
              />
            </div>
          </div>

          <div className="flex items-center justify-between p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-800">
            <div className="space-y-0.5">
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-semibold text-slate-800 dark:text-slate-200">
                  Distributed Start Barrier
                </span>
                <HelpTooltip
                  text="Zero clock-skew distributed rendezvous barrier coordinating simultaneous test execution across all pods."
                  label="Help for start barrier"
                />
              </div>
              <p className="text-[11px] text-slate-500 dark:text-slate-400">
                Hold load generators until all pods reach rendezvous readiness.
              </p>
            </div>
            <Switch
              checked={barrierEnabled}
              onCheckedChange={setBarrierEnabled}
              aria-label="Toggle distributed start barrier"
            />
          </div>

          <InfoBadge
            variant="info"
            title="Kubernetes Runner Resource Allocation"
          >
            Configure memory requests equal to limits for Guaranteed Quality of Service (QoS).
            Allocate sufficient CPU millicores (e.g. 1000m) to minimize garbage collection latency
            impacts during high-throughput benchmark runs.
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
            Dispatch Run
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
