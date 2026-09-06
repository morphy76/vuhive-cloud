import React from 'react'
import { Menu, FileCode, CheckCircle2 } from 'lucide-react'
import { ThemeToggle } from './ThemeToggle'
import { PRIMARY_NAV_ITEMS, RouteId } from '@/types/navigation'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { Badge } from '@/components/ui/badge'

interface TopHeaderProps {
  currentRoute: RouteId
  onOpenDrawer: () => void
}

export const TopHeader: React.FC<TopHeaderProps> = ({ currentRoute, onOpenDrawer }) => {
  const currentNav = PRIMARY_NAV_ITEMS.find((item) => item.id === currentRoute)

  return (
    <header className="sticky top-0 z-20 flex h-16 w-full items-center justify-between border-b border-slate-200 bg-white/80 px-4 sm:px-6 backdrop-blur-md dark:border-slate-800 dark:bg-slate-900/80 transition-colors">
      {/* Left side: Mobile/Tablet menu button + Breadcrumbs */}
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          onClick={onOpenDrawer}
          aria-label="Open navigation menu"
          className="lg:hidden text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100 min-h-[44px] min-w-[44px]"
        >
          <Menu className="w-5 h-5" />
        </Button>

        <div className="flex items-center gap-2">
          <div className="lg:hidden flex items-center gap-2 font-bold text-slate-900 dark:text-white mr-2">
            <div className="w-7 h-7 rounded-lg bg-brand-600 flex items-center justify-center text-white font-black text-sm">
              vh
            </div>
          </div>
          <nav aria-label="Breadcrumbs" className="flex items-center text-sm font-medium">
            <span className="text-slate-400 dark:text-slate-500">vuhive</span>
            <span className="mx-2 text-slate-300 dark:text-slate-600">/</span>
            <span className="text-slate-900 dark:text-slate-100 capitalize">
              {currentNav ? currentNav.label : currentRoute}
            </span>
          </nav>
        </div>
      </div>

      {/* Right side: Cluster badge, API doc link, Theme Toggle */}
      <div className="flex items-center gap-2 sm:gap-3">
        {/* Cluster status badge */}
        <Badge variant="success" className="hidden sm:inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium">
          <CheckCircle2 className="w-3.5 h-3.5" />
          <span>Control Plane Connected</span>
        </Badge>

        {/* OpenAPI Link */}
        <Tooltip>
          <TooltipTrigger asChild>
            <a
              href="/openapi.yaml"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="OpenAPI Specification"
              className="hidden md:inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors border border-slate-200 dark:border-slate-800"
            >
              <FileCode className="w-3.5 h-3.5 text-brand-500" />
              <span>OpenAPI</span>
            </a>
          </TooltipTrigger>
          <TooltipContent>OpenAPI 3.1 Specification</TooltipContent>
        </Tooltip>

        {/* Theme Toggle */}
        <ThemeToggle />
      </div>
    </header>
  )
}
