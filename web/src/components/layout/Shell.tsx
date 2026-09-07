import React, { useState } from 'react'
import { BookOpen } from 'lucide-react'
import { RouteId } from '@/types/navigation'
import { Sidebar } from './Sidebar'
import { TopHeader } from './TopHeader'
import { NavDrawer } from './NavDrawer'
import { BottomNav } from './BottomNav'
import { SkipLink } from '@/components/ui/skip-link'
import { TooltipProvider, Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/toaster'
import { OfflineBanner } from '@/components/ui/offline-banner'
import { Button } from '@/components/ui/button'
import { RecipeDrawer } from '@/components/recipes/RecipeDrawer'

import { useRecipe } from '@/context/RecipeContext'

interface ShellProps {
  currentRoute: RouteId
  onSelectRoute: (route: RouteId) => void
  children: React.ReactNode
}

export const Shell: React.FC<ShellProps> = ({ currentRoute, onSelectRoute, children }) => {
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState<boolean>(false)
  const [isDrawerOpen, setIsDrawerOpen] = useState<boolean>(false)
  const { isOpen: isRecipeDrawerOpen, openRecipe, closeRecipe, activeRecipeId } = useRecipe()

  return (
    <div className="flex min-h-screen w-full bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 overflow-x-hidden">
      <SkipLink />
      <TooltipProvider>
        {/* Desktop Collapsible Sidebar (>=1024px) */}
        <Sidebar
          currentRoute={currentRoute}
          onSelectRoute={onSelectRoute}
          isCollapsed={isSidebarCollapsed}
          onToggleCollapse={() => setIsSidebarCollapsed((prev) => !prev)}
        />

        {/* Main Content Area */}
        <div className="flex flex-1 flex-col min-w-0">
          {/* Offline Indicator Banner */}
          <OfflineBanner />

          {/* Top Header */}
          <TopHeader
            currentRoute={currentRoute}
            onOpenDrawer={() => setIsDrawerOpen(true)}
            onOpenRecipes={() => openRecipe()}
          />

          {/* Dynamic Page Content */}
          <main id="main-content" className="flex-1 p-4 sm:p-6 lg:p-8 pb-24 md:pb-8 max-w-7xl w-full mx-auto">
            {children}
          </main>
        </div>

        {/* Floating Action Button for Recipe Guidance on any view */}
        <div className="fixed bottom-20 right-4 sm:bottom-6 sm:right-6 md:bottom-8 md:right-8 z-30">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                onClick={() => openRecipe()}
                aria-label="Open Recipe Guidance"
                className="h-12 w-12 sm:h-14 sm:w-14 rounded-full shadow-lg shadow-brand-500/25 bg-gradient-to-tr from-brand-600 to-indigo-600 hover:from-brand-500 hover:to-indigo-500 text-white p-0 flex items-center justify-center transition-transform hover:scale-105 active:scale-95"
              >
                <BookOpen className="w-5 h-5 sm:w-6 sm:h-6" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="left">API Recipes & cURL Generator</TooltipContent>
          </Tooltip>
        </div>

        {/* Slide-over Contextual Recipe Drawer */}
        <RecipeDrawer
          isOpen={isRecipeDrawerOpen}
          onClose={closeRecipe}
          currentRoute={currentRoute}
          initialRecipeId={activeRecipeId}
        />

        {/* Tablet Slide-out Navigation Drawer (768px - 1023px) */}
        <NavDrawer
          isOpen={isDrawerOpen}
          onClose={() => setIsDrawerOpen(false)}
          currentRoute={currentRoute}
          onSelectRoute={onSelectRoute}
        />

        {/* Mobile Bottom Navigation (<768px) */}
        <BottomNav
          currentRoute={currentRoute}
          onSelectRoute={onSelectRoute}
        />
      </TooltipProvider>
      <Toaster />
    </div>
  )
}
