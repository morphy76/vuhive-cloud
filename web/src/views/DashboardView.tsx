import React from 'react'
import { Activity, Layers, CalendarClock, CheckCircle2, Play, Cpu, ArrowUpRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'

export const DashboardView: React.FC<{ onNavigate?: (route: any) => void }> = ({ onNavigate }) => {
  const stats = [
    {
      title: 'Active Runners',
      value: '12',
      trend: '+4 from last hour',
      icon: Activity,
      color: 'text-brand-500 bg-brand-50 dark:bg-brand-950/50',
    },
    {
      title: 'Test Suites',
      value: '8',
      trend: 'All artifacts ready',
      icon: Layers,
      color: 'text-indigo-500 bg-indigo-50 dark:bg-indigo-950/50',
    },
    {
      title: 'Cron Schedules',
      value: '4',
      trend: 'Next trigger in 14m',
      icon: CalendarClock,
      color: 'text-amber-500 bg-amber-50 dark:bg-amber-950/50',
    },
    {
      title: 'SLA Pass Rate',
      value: '99.4%',
      trend: 'Last 24 hours',
      icon: CheckCircle2,
      color: 'text-emerald-500 bg-emerald-50 dark:bg-emerald-950/50',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header section */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Control Plane Overview
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Real-time cluster orchestrator health, runner workload status, and performance telemetry.
          </p>
        </div>

        {/* Quick action buttons */}
        <div className="flex items-center gap-3">
          <Button
            onClick={() => onNavigate?.('runs')}
            className="min-h-[44px] gap-2"
          >
            <Play className="w-4 h-4 fill-current" />
            <span>Trigger Run</span>
          </Button>
        </div>
      </div>

      {/* KPI Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, i) => {
          const Icon = stat.icon
          return (
            <div
              key={i}
              className="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-slate-500 dark:text-slate-400">
                  {stat.title}
                </span>
                <div className={`p-2 rounded-xl ${stat.color}`}>
                  <Icon className="w-4 h-4" />
                </div>
              </div>
              <div className="mt-4">
                <div className="text-2xl sm:text-3xl font-bold text-slate-900 dark:text-white">
                  {stat.value}
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  {stat.trend}
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {/* Orchestrator Architecture Status */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kubernetes Engine Card */}
        <div className="lg:col-span-2 p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Cpu className="w-5 h-5 text-brand-500" />
              <h2 className="text-base font-semibold text-slate-900 dark:text-white">
                Cluster Execution Pipeline
              </h2>
            </div>
            <Badge variant="success">Synchronized</Badge>
          </div>

          <div className="space-y-3">
            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/60 dark:border-slate-800 flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-slate-900 dark:text-white">
                  Ephemeral Compilation Subsystem
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  Target architectures: linux/amd64, linux/arm64
                </div>
              </div>
              <Badge variant="success">Ready</Badge>
            </div>

            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/60 dark:border-slate-800 flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-slate-900 dark:text-white">
                  Distributed Start Barrier Rendezvous
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  Zero clock-skew distributed synchronized firing
                </div>
              </div>
              <Badge variant="success">Active</Badge>
            </div>

            <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200/60 dark:border-slate-800 flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-slate-900 dark:text-white">
                  Telemetry & KPI Indexer
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  summary.json parser (p50, p90, p95, p99, TPS, SLA)
                </div>
              </div>
              <Badge variant="success">Listening</Badge>
            </div>
          </div>
        </div>

        {/* Quick Links & Documentation Card */}
        <div className="p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs flex flex-col justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-900 dark:text-white mb-2">
              API & Spec Documentation
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-4">
              Explore interactive endpoints, OpenAPI 3.1 definitions, and control plane recipes.
            </p>

            <TooltipProvider>
              <div className="space-y-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <a
                      href="/openapi.yaml"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center justify-between p-3 rounded-xl hover:bg-slate-50 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-800 transition-colors text-xs font-medium text-slate-700 dark:text-slate-300"
                    >
                      <span>OpenAPI 3.1 Specification</span>
                      <ArrowUpRight className="w-4 h-4 text-slate-400" />
                    </a>
                  </TooltipTrigger>
                  <TooltipContent>
                    View the full OpenAPI 3.1 specification for the Control Plane API
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <a
                      href="https://github.com/morphy76/vuhive-cloud/blob/main/docs/cookbook.md"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center justify-between p-3 rounded-xl hover:bg-slate-50 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-800 transition-colors text-xs font-medium text-slate-700 dark:text-slate-300"
                    >
                      <span>Control Plane Cookbook</span>
                      <ArrowUpRight className="w-4 h-4 text-slate-400" />
                    </a>
                  </TooltipTrigger>
                  <TooltipContent>
                    Read the developer cookbook for advanced configuration recipes
                  </TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          </div>

          <div className="mt-6 pt-4 border-t border-slate-200 dark:border-slate-800 text-[11px] text-slate-400">
            vuhive-cloud • Reactive Go BFF & Embedded React 19 PWA
          </div>
        </div>
      </div>
    </div>
  )
}
