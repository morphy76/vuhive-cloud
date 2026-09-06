import React from 'react'
import { PRIMARY_NAV_ITEMS, RouteId } from '@/types/navigation'

interface BottomNavProps {
  currentRoute: RouteId
  onSelectRoute: (route: RouteId) => void
}

export const BottomNav: React.FC<BottomNavProps> = ({ currentRoute, onSelectRoute }) => {
  return (
    <nav
      aria-label="Mobile bottom navigation"
      className="md:hidden fixed bottom-0 left-0 right-0 z-30 bg-white/95 dark:bg-slate-900/95 backdrop-blur-md border-t border-slate-200 dark:border-slate-800 pb-safe transition-colors"
    >
      <div className="grid grid-cols-4 h-16 max-w-lg mx-auto">
        {PRIMARY_NAV_ITEMS.map((item) => {
          const Icon = item.icon
          const isActive = currentRoute === item.id

          return (
            <button
              key={item.id}
              type="button"
              onClick={() => onSelectRoute(item.id)}
              aria-label={item.label}
              className={`flex flex-col items-center justify-center gap-1 min-h-[44px] transition-colors ${
                isActive
                  ? 'text-brand-600 dark:text-brand-400 font-semibold'
                  : 'text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
              }`}
            >
              <div className="relative">
                <Icon className={`w-5 h-5 transition-transform ${isActive ? 'scale-110' : ''}`} />
                {isActive && (
                  <span className="absolute -top-1 -right-1 w-1.5 h-1.5 rounded-full bg-brand-600 dark:bg-brand-400" />
                )}
              </div>
              <span className="text-[11px] tracking-tight">{item.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
