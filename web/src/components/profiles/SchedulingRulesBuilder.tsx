import React from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import type {
  NodeAffinityTerm,
  TolerationSpec,
  AffinityOperator,
  TolerationOperator,
  TolerationEffect,
} from '@/types/profile'

interface SchedulingRulesBuilderProps {
  nodeSelector: Record<string, string>
  onNodeSelectorChange: (selector: Record<string, string>) => void
  affinityTerms: NodeAffinityTerm[]
  onAffinityTermsChange: (terms: NodeAffinityTerm[]) => void
  tolerations: TolerationSpec[]
  onTolerationsChange: (tolerations: TolerationSpec[]) => void
}

export const SchedulingRulesBuilder: React.FC<SchedulingRulesBuilderProps> = ({
  nodeSelector,
  onNodeSelectorChange,
  affinityTerms,
  onAffinityTermsChange,
  tolerations,
  onTolerationsChange,
}) => {
  // Node Selector helper handlers
  const nodeSelectorEntries = Object.entries(nodeSelector)

  const handleAddNodeSelector = () => {
    onNodeSelectorChange({
      ...nodeSelector,
      [`label-${Date.now()}`]: 'true',
    })
  }

  const handleUpdateNodeSelectorKey = (oldKey: string, newKey: string) => {
    const updated: Record<string, string> = {}
    for (const [k, v] of Object.entries(nodeSelector)) {
      if (k === oldKey) {
        updated[newKey] = v
      } else {
        updated[k] = v
      }
    }
    onNodeSelectorChange(updated)
  }

  const handleUpdateNodeSelectorValue = (key: string, value: string) => {
    onNodeSelectorChange({
      ...nodeSelector,
      [key]: value,
    })
  }

  const handleRemoveNodeSelector = (key: string) => {
    const updated = { ...nodeSelector }
    delete updated[key]
    onNodeSelectorChange(updated)
  }

  // Affinity Terms helper handlers
  const handleAddAffinityTerm = () => {
    onAffinityTermsChange([
      ...affinityTerms,
      {
        key: '',
        operator: 'In',
        values: [''],
      },
    ])
  }

  const handleUpdateAffinityTerm = (index: number, updatedTerm: NodeAffinityTerm) => {
    const updated = [...affinityTerms]
    updated[index] = updatedTerm
    onAffinityTermsChange(updated)
  }

  const handleRemoveAffinityTerm = (index: number) => {
    onAffinityTermsChange(affinityTerms.filter((_, i) => i !== index))
  }

  // Tolerations helper handlers
  const handleAddToleration = () => {
    onTolerationsChange([
      ...tolerations,
      {
        key: '',
        operator: 'Equal',
        value: '',
        effect: 'NoSchedule',
      },
    ])
  }

  const handleUpdateToleration = (index: number, updatedTol: TolerationSpec) => {
    const updated = [...tolerations]
    updated[index] = updatedTol
    onTolerationsChange(updated)
  }

  const handleRemoveToleration = (index: number) => {
    onTolerationsChange(tolerations.filter((_, i) => i !== index))
  }

  return (
    <div className="space-y-6">
      {/* 1. Node Selector Section */}
      <div className="space-y-3 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              Node Selectors
            </h4>
            <HelpTooltip
              text="Direct key-value label selectors that candidate Kubernetes worker nodes must possess to schedule runner pods."
              label="Help for node selectors"
            />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAddNodeSelector}
            className="h-8 gap-1.5 text-xs border-slate-200 dark:border-slate-800"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Add Label</span>
          </Button>
        </div>

        {nodeSelectorEntries.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-slate-400 italic py-2">
            No node selectors configured. Runner pods can schedule on any eligible worker node.
          </p>
        ) : (
          <div className="space-y-2">
            {nodeSelectorEntries.map(([key, value], idx) => (
              <div key={idx} className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="e.g. node-role.kubernetes.io/runner"
                  value={key}
                  onChange={(e) => handleUpdateNodeSelectorKey(key, e.target.value)}
                  aria-label={`Node selector key ${idx + 1}`}
                  className="flex-1 rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
                <span className="text-xs text-slate-400 font-bold">=</span>
                <input
                  type="text"
                  placeholder="e.g. true"
                  value={value}
                  onChange={(e) => handleUpdateNodeSelectorValue(key, e.target.value)}
                  aria-label={`Node selector value ${idx + 1}`}
                  className="flex-1 rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => handleRemoveNodeSelector(key)}
                  aria-label={`Remove node selector ${key}`}
                  className="h-8 w-8 text-slate-400 hover:text-rose-600 dark:hover:text-rose-400"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 2. Node Affinity MatchExpressions Section */}
      <div className="space-y-3 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              Node Affinity (`matchExpressions`)
            </h4>
            <HelpTooltip
              text="Advanced Kubernetes node affinity match expressions (e.g. instance type In [c5.4xlarge, c6i.4xlarge], zone In [us-east-1a])."
              label="Help for node affinity matchExpressions"
            />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAddAffinityTerm}
            className="h-8 gap-1.5 text-xs border-slate-200 dark:border-slate-800"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Add Match Expression</span>
          </Button>
        </div>

        {affinityTerms.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-slate-400 italic py-2">
            No node affinity match expressions configured.
          </p>
        ) : (
          <div className="space-y-3">
            {affinityTerms.map((term, idx) => {
              const operatorNeedsValues = ['In', 'NotIn', 'Gt', 'Lt'].includes(term.operator)
              return (
                <div
                  key={idx}
                  className="rounded-lg border border-slate-200 dark:border-slate-800/80 bg-slate-50/50 dark:bg-slate-900/40 p-3 space-y-2"
                >
                  <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                    <input
                      type="text"
                      placeholder="Key (e.g. node.kubernetes.io/instance-type)"
                      value={term.key}
                      onChange={(e) =>
                        handleUpdateAffinityTerm(idx, { ...term, key: e.target.value })
                      }
                      aria-label={`Affinity term key ${idx + 1}`}
                      className="flex-1 rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                    />
                    <select
                      value={term.operator}
                      onChange={(e) =>
                        handleUpdateAffinityTerm(idx, {
                          ...term,
                          operator: e.target.value as AffinityOperator,
                          values: ['Exists', 'DoesNotExist'].includes(e.target.value)
                            ? []
                            : term.values || [''],
                        })
                      }
                      aria-label={`Affinity term operator ${idx + 1}`}
                      className="w-full sm:w-36 rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-2.5 py-1.5 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                    >
                      <option value="In">In</option>
                      <option value="NotIn">NotIn</option>
                      <option value="Exists">Exists</option>
                      <option value="DoesNotExist">DoesNotExist</option>
                      <option value="Gt">Gt</option>
                      <option value="Lt">Lt</option>
                    </select>

                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveAffinityTerm(idx)}
                      aria-label={`Remove affinity term ${idx + 1}`}
                      className="h-8 w-8 text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 self-end sm:self-center"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </div>

                  {operatorNeedsValues && (
                    <div className="pt-1">
                      <input
                        type="text"
                        placeholder="Values (comma-separated, e.g. c5.4xlarge, c6i.4xlarge)"
                        value={(term.values || []).join(', ')}
                        onChange={(e) => {
                          const raw = e.target.value
                          const values = raw
                            .split(',')
                            .map((v) => v.trim())
                            .filter(Boolean)
                          handleUpdateAffinityTerm(idx, { ...term, values })
                        }}
                        aria-label={`Affinity term values ${idx + 1}`}
                        className="w-full rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                      />
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* 3. Tolerations Section */}
      <div className="space-y-3 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-950 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <h4 className="text-sm font-bold text-slate-900 dark:text-slate-100">
              Pod Tolerations
            </h4>
            <HelpTooltip
              text="Allow runner pods to schedule onto tainted nodes (e.g. dedicated=loadgen:NoSchedule or sandbox.gvisor.io/runtime:Exists:NoSchedule)."
              label="Help for tolerations"
            />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAddToleration}
            className="h-8 gap-1.5 text-xs border-slate-200 dark:border-slate-800"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Add Toleration</span>
          </Button>
        </div>

        {tolerations.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-slate-400 italic py-2">
            No tolerations configured. Runner pods will respect all node taints.
          </p>
        ) : (
          <div className="space-y-3">
            {tolerations.map((tol, idx) => (
              <div
                key={idx}
                className="rounded-lg border border-slate-200 dark:border-slate-800/80 bg-slate-50/50 dark:bg-slate-900/40 p-3 space-y-2"
              >
                <div className="grid grid-cols-1 sm:grid-cols-4 gap-2 items-center">
                  <input
                    type="text"
                    placeholder="Key (e.g. dedicated)"
                    value={tol.key || ''}
                    onChange={(e) =>
                      handleUpdateToleration(idx, { ...tol, key: e.target.value })
                    }
                    aria-label={`Toleration key ${idx + 1}`}
                    className="rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                  />

                  <select
                    value={tol.operator || 'Equal'}
                    onChange={(e) => {
                      const op = e.target.value as TolerationOperator
                      handleUpdateToleration(idx, {
                        ...tol,
                        operator: op,
                        value: op === 'Exists' ? '' : tol.value,
                      })
                    }}
                    aria-label={`Toleration operator ${idx + 1}`}
                    className="rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-2.5 py-1.5 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                  >
                    <option value="Equal">Equal</option>
                    <option value="Exists">Exists</option>
                  </select>

                  <input
                    type="text"
                    placeholder={tol.operator === 'Exists' ? 'Value (n/a)' : 'Value (e.g. loadgen)'}
                    disabled={tol.operator === 'Exists'}
                    value={tol.value || ''}
                    onChange={(e) =>
                      handleUpdateToleration(idx, { ...tol, value: e.target.value })
                    }
                    aria-label={`Toleration value ${idx + 1}`}
                    className="rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 disabled:opacity-50 disabled:bg-slate-100 dark:disabled:bg-slate-800"
                  />

                  <div className="flex items-center gap-1.5">
                    <select
                      value={tol.effect || 'NoSchedule'}
                      onChange={(e) =>
                        handleUpdateToleration(idx, {
                          ...tol,
                          effect: e.target.value as TolerationEffect,
                        })
                      }
                      aria-label={`Toleration effect ${idx + 1}`}
                      className="flex-1 rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-2.5 py-1.5 text-xs font-medium text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                    >
                      <option value="NoSchedule">NoSchedule</option>
                      <option value="PreferNoSchedule">PreferNoSchedule</option>
                      <option value="NoExecute">NoExecute</option>
                    </select>

                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveToleration(idx)}
                      aria-label={`Remove toleration ${idx + 1}`}
                      className="h-8 w-8 text-slate-400 hover:text-rose-600 dark:hover:text-rose-400"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                </div>

                {tol.effect === 'NoExecute' && (
                  <div className="flex items-center gap-2 pt-1 text-xs">
                    <span className="text-slate-500 dark:text-slate-400">
                      Toleration Seconds:
                    </span>
                    <input
                      type="number"
                      min={0}
                      placeholder="e.g. 300"
                      value={tol.toleration_seconds ?? ''}
                      onChange={(e) =>
                        handleUpdateToleration(idx, {
                          ...tol,
                          toleration_seconds: e.target.value ? Number(e.target.value) : undefined,
                        })
                      }
                      aria-label={`Toleration seconds ${idx + 1}`}
                      className="w-28 rounded-lg border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 px-2.5 py-1 text-xs text-slate-900 dark:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
