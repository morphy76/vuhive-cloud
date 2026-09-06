import React, { useRef } from 'react'
import { X, Activity, Github, ExternalLink } from 'lucide-react'
import { PRIMARY_NAV_ITEMS, RouteId } from '@/types/navigation'
import { ThemeToggle } from './ThemeToggle'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { cn } from '@/lib/utils'

interface NavDrawerProps {
  isOpen: boolean
  onClose: () => void
  currentRoute: RouteId
  onSelectRoute: (route: RouteId) => void
}

export const NavDrawer: React.FC<NavDrawerProps> = ({
  isOpen,
  onClose,
  currentRoute,
  onSelectRoute,
}) => {
  const touchStartX = useRef<number | null>(null)

  // Touch swipe handling to close on left swipe
  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
  }

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartX.current === null) return
    const touchEndX = e.changedTouches[0].clientX
    const deltaX = touchEndX - touchStartX.current
    // Swiped left by at least 50px
    if (deltaX < -50) {
      onClose()
    }
    touchStartX.current = null
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent 
        className="lg:hidden fixed inset-y-0 left-0 z-50 flex w-72 max-w-[80vw] flex-col gap-0 border-r border-slate-200 bg-white p-0 dark:border-slate-800 dark:bg-slate-900 shadow-2xl sm:rounded-none"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        aria-label="Navigation drawer"
      >
        <VisuallyHidden>
          <DialogTitle>Navigation Menu</DialogTitle>
          <DialogDescription>Access all main areas of the application.</DialogDescription>
        </VisuallyHidden>

        {/* Header */}
        <div className="flex h-16 items-center justify-between px-4 border-b border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-brand-600 to-indigo-500 flex items-center justify-center text-white shadow-md shadow-brand-500/20">
              <Activity className="w-5 h-5 text-white" />
            </div>
            <div className="flex flex-col">
              <span className="font-bold text-slate-900 dark:text-white tracking-tight">
                vuhive-cloud
              </span>
              <span className="text-[11px] text-slate-400 dark:text-slate-500 font-mono">
                Control Plane v0.1.0
              </span>
            </div>
          </div>

          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            aria-label="Close navigation menu"
            className="text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 min-h-[44px] min-w-[44px]"
          >
            <X className="w-5 h-5" />
          </Button>
        </div>

        {/* Navigation links */}
        <nav aria-label="Tablet drawer navigation" className="flex-1 space-y-1.5 p-4 overflow-y-auto">
          {PRIMARY_NAV_ITEMS.map((item) => {
            const Icon = item.icon
            const isActive = currentRoute === item.id

            return (
              <Button
                key={item.id}
                variant="ghost"
                onClick={() => {
                  onSelectRoute(item.id)
                  onClose()
                }}
                aria-label={item.label}
                className={cn(
                  "w-full flex items-center justify-start gap-3 px-3.5 py-3 h-auto rounded-xl font-medium text-sm transition-all min-h-[44px]",
                  isActive
                    ? "bg-brand-50 text-brand-700 dark:bg-brand-950/60 dark:text-brand-300 shadow-xs hover:bg-brand-100 dark:hover:bg-brand-900/60"
                    : "text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800"
                )}
              >
                <Icon className={cn("w-5 h-5", isActive ? "text-brand-600 dark:text-brand-400" : "")} />
                <div className="flex flex-col text-left">
                  <span>{item.label}</span>
                  <span className="text-xs text-slate-400 dark:text-slate-500 font-normal">
                    {item.description}
                  </span>
                </div>
              </Button>
            )
          })}
        </nav>

        {/* Footer actions */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-800 space-y-2">
          <div className="flex items-center justify-between px-2">
            <span className="text-xs text-slate-500 dark:text-slate-400">Theme</span>
            <ThemeToggle />
          </div>

          <a
            href="https://github.com/morphy76/vuhive-cloud"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center justify-between px-3 py-2 rounded-xl text-xs text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Github className="w-4 h-4" />
              <span>GitHub Repository</span>
            </div>
            <ExternalLink className="w-3.5 h-3.5 text-slate-400" />
          </a>
        </div>
      </DialogContent>
    </Dialog>
  )
}
