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
import { queryClient, idbPersister } from '@/lib/query-client'

import { RecipeProvider } from '@/context/RecipeContext'

export const AppContent: React.FC = () => {
  const [currentRoute, setCurrentRoute] = useState<RouteId>('dashboard')

  return (
    <RecipeProvider>
      <Shell currentRoute={currentRoute} onSelectRoute={setCurrentRoute}>
        {currentRoute === 'dashboard' && <DashboardView onNavigate={setCurrentRoute} />}
        {currentRoute === 'suites' && <SuitesView />}
        {currentRoute === 'runs' && <RunsView />}
        {currentRoute === 'schedules' && <SchedulesView />}
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
