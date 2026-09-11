import React, { useState } from 'react'
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client'
import { ThemeProvider } from '@/context/ThemeContext'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Shell } from '@/components/layout/Shell'
import { RouteId } from '@/types/navigation'
import { DashboardView } from '@/views/DashboardView'
import { SuitesView } from '@/views/SuitesView'
import { RunsView } from '@/views/RunsView'
import { SchedulesView } from '@/views/SchedulesView'
import { ProfilesView } from '@/views/ProfilesView'
import { queryClient, idbPersister } from '@/lib/query-client'

import { RecipeProvider } from '@/context/RecipeContext'

const CookbookView = React.lazy(() => import('@/views/CookbookView'))

export const AppContent: React.FC = () => {
  const [currentRoute, setCurrentRoute] = useState<RouteId>('dashboard')

  return (
    <RecipeProvider>
      <Shell currentRoute={currentRoute} onSelectRoute={setCurrentRoute}>
        {currentRoute === 'dashboard' && <DashboardView onNavigate={setCurrentRoute} />}
        {currentRoute === 'suites' && <SuitesView />}
        {currentRoute === 'runs' && <RunsView />}
        {currentRoute === 'schedules' && <SchedulesView />}
        {currentRoute === 'profiles' && <ProfilesView />}
        {currentRoute === 'cookbook' && (
          <React.Suspense
            fallback={
              <div className="flex items-center justify-center p-12 text-slate-400">
                <div className="flex items-center gap-2 text-sm">
                  <span className="w-4 h-4 border-2 border-brand-500 border-t-transparent rounded-full animate-spin" />
                  <span>Loading Cookbook...</span>
                </div>
              </div>
            }
          >
            <CookbookView />
          </React.Suspense>
        )}
      </Shell>
    </RecipeProvider>
  )
}

export const App: React.FC = () => {
  return (
    <PersistQueryClientProvider
      client={queryClient}
      persistOptions={{ persister: idbPersister }}
    >
      <ThemeProvider>
        <TooltipProvider>
          <AppContent />
        </TooltipProvider>
      </ThemeProvider>
    </PersistQueryClientProvider>
  )
}

export default App
