import React, { useState } from 'react'
import { ThemeProvider } from '@/context/ThemeContext'
import { Shell } from '@/components/layout/Shell'
import { RouteId } from '@/types/navigation'
import { DashboardView } from '@/views/DashboardView'
import { SuitesView } from '@/views/SuitesView'
import { RunsView } from '@/views/RunsView'
import { SchedulesView } from '@/views/SchedulesView'

export const AppContent: React.FC = () => {
  const [currentRoute, setCurrentRoute] = useState<RouteId>('dashboard')

  return (
    <Shell currentRoute={currentRoute} onSelectRoute={setCurrentRoute}>
      {currentRoute === 'dashboard' && <DashboardView onNavigate={setCurrentRoute} />}
      {currentRoute === 'suites' && <SuitesView />}
      {currentRoute === 'runs' && <RunsView />}
      {currentRoute === 'schedules' && <SchedulesView />}
    </Shell>
  )
}

export const App: React.FC = () => {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  )
}

export default App
