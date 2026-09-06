import React, { useState } from 'react'
import { RouteId } from '@/types/navigation'
import { Sidebar } from './Sidebar'
import { TopHeader } from './TopHeader'
import { NavDrawer } from './NavDrawer'
import { BottomNav } from './BottomNav'

interface ShellProps {
  currentRoute: RouteId
  onSelectRoute: (route: RouteId) => void
  children: React.ReactNode
}

export const Shell: React.FC<ShellProps> = ({ currentRoute, onSelectRoute, children }) => {
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState<boolean>(false)
  const [isDrawerOpen, setIsDrawerOpen] = useState<boolean>(false)

  return (
    <div className="flex min-h-screen w-full bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 overflow-x-hidden">
      {/* Desktop Collapsible Sidebar (>=1024px) */}
      <Sidebar
        currentRoute={currentRoute}
        onSelectRoute={onSelectRoute}
        isCollapsed={isSidebarCollapsed}
        onToggleCollapse={() => setIsSidebarCollapsed((prev) => !prev)}
      />

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col min-w-0">
        {/* Top Header */}
        <TopHeader
          currentRoute={currentRoute}
          onOpenDrawer={() => setIsDrawerOpen(true)}
        />

        {/* Dynamic Page Content */}
        <main className="flex-1 p-4 sm:p-6 lg:p-8 pb-24 md:pb-8 max-w-7xl w-full mx-auto">
          {children}
        </main>
      </div>

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
    </div>
  )
}
