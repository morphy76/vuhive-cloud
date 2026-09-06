import React from 'react'
import { ChevronLeft, ChevronRight, Activity, Github } from 'lucide-react'
import { PRIMARY_NAV_ITEMS, RouteId } from '@/types/navigation'

interface SidebarProps {
  currentRoute: RouteId
  onSelectRoute: (route: RouteId) => void
  isCollapsed: boolean
  onToggleCollapse: () => void
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentRoute,
  onSelectRoute,
  isCollapsed,
  onToggleCollapse,
}) => {
  return (
    <aside
      className={`hidden lg:flex flex-col border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 transition-all duration-300 select-none ${
        isCollapsed ? 'w-20' : 'w-64'
      }`}
    >
      {/* Brand Header */}
      <div className="flex h-16 items-center justify-between px-4 border-b border-slate-200 dark:border-slate-800">
        <div className="flex items-center gap-3 overflow-hidden">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-brand-600 to-indigo-500 flex-shrink-0 flex items-center justify-center text-white font-black shadow-md shadow-brand-500/20">
            <Activity className="w-5 h-5 text-white" />
          </div>
          {!isCollapsed && (
            <div className="flex flex-col truncate">
              <span className="font-bold text-slate-900 dark:text-white tracking-tight">
                vuhive-cloud
              </span>
              <span className="text-[11px] text-slate-400 dark:text-slate-500 font-mono">
                Control Plane v0.1.0
              </span>
            </div>
          )}
        </div>

        {/* Collapse / Expand Toggle */}
        <button
          type="button"
          onClick={onToggleCollapse}
          aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="inline-flex items-center justify-center p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          {isCollapsed ? (
            <ChevronRight className="w-5 h-5" />
          ) : (
            <ChevronLeft className="w-5 h-5" />
          )}
        </button>
      </div>

      {/* Nav items list */}
      <nav aria-label="Desktop sidebar navigation" className="flex-1 space-y-1.5 p-3 overflow-y-auto">
        {PRIMARY_NAV_ITEMS.map((item) => {
          const Icon = item.icon
          const isActive = currentRoute === item.id

          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onSelectRoute(item.id)}
              aria-label={item.label}
              title={isCollapsed ? item.label : undefined}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl font-medium text-sm transition-all ${
                isActive
                  ? 'bg-brand-50 text-brand-700 dark:bg-brand-950/60 dark:text-brand-300 shadow-xs'
                  : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100 hover:bg-slate-100/80 dark:hover:bg-slate-800/80'
              } ${isCollapsed ? 'justify-center px-2' : ''}`}
            >
              <Icon className={`w-5 h-5 flex-shrink-0 ${isActive ? 'text-brand-600 dark:text-brand-400' : ''}`} />
              {!isCollapsed && <span className="truncate">{item.label}</span>}
            </button>
          )
        })}
      </nav>

      {/* Footer / External Links */}
      <div className="p-3 border-t border-slate-200 dark:border-slate-800">
        <a
          href="https://github.com/morphy76/vuhive-cloud"
          target="_blank"
          rel="noreferrer"
          className={`flex items-center gap-3 px-3 py-2 rounded-xl text-xs text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors ${
            isCollapsed ? 'justify-center px-2' : ''
          }`}
          title="GitHub Repository"
        >
          <Github className="w-4 h-4 flex-shrink-0" />
          {!isCollapsed && <span>GitHub Project</span>}
        </a>
      </div>
    </aside>
  )
}
