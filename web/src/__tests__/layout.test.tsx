import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Sidebar } from '../components/layout/Sidebar'
import { NavDrawer } from '../components/layout/NavDrawer'
import { BottomNav } from '../components/layout/BottomNav'
import { ThemeProvider } from '../context/ThemeContext'

describe('Layout Components Unit Tests', () => {
  it('Sidebar renders in expanded state by default and handles route selection', () => {
    const handleSelectRoute = vi.fn()
    const handleToggleCollapse = vi.fn()

    const { rerender } = render(
      <Sidebar
        currentRoute="dashboard"
        onSelectRoute={handleSelectRoute}
        isCollapsed={false}
        onToggleCollapse={handleToggleCollapse}
      />
    )

    expect(screen.getByText('vuhive-cloud')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /collapse sidebar/i })).toBeInTheDocument()

    // Click on Suites
    fireEvent.click(screen.getByRole('button', { name: 'Suites' }))
    expect(handleSelectRoute).toHaveBeenCalledWith('suites')

    // Toggle collapse
    fireEvent.click(screen.getByRole('button', { name: /collapse sidebar/i }))
    expect(handleToggleCollapse).toHaveBeenCalledTimes(1)

    // In collapsed state
    rerender(
      <Sidebar
        currentRoute="dashboard"
        onSelectRoute={handleSelectRoute}
        isCollapsed={true}
        onToggleCollapse={handleToggleCollapse}
      />
    )
    expect(screen.getByRole('button', { name: /expand sidebar/i })).toBeInTheDocument()
    expect(screen.queryByText('Control Plane v0.1.0')).not.toBeInTheDocument()
  })

  it('NavDrawer closes on Escape key press and left touch swipe', () => {
    const handleClose = vi.fn()
    const handleSelectRoute = vi.fn()

    const { rerender } = render(
      <ThemeProvider>
        <NavDrawer
          isOpen={true}
          onClose={handleClose}
          currentRoute="dashboard"
          onSelectRoute={handleSelectRoute}
        />
      </ThemeProvider>
    )

    expect(screen.getByRole('dialog', { name: /navigation drawer/i })).toBeInTheDocument()

    // Press Escape
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(handleClose).toHaveBeenCalledTimes(1)

    // Touch swipe left simulation
    const drawerDialog = screen.getByRole('dialog', { name: /navigation drawer/i })
    const drawerPanel = drawerDialog.querySelector('div.relative.z-10')!
    expect(drawerPanel).toBeInTheDocument()

    fireEvent.touchStart(drawerPanel, {
      touches: [{ clientX: 200, clientY: 100 }],
    })
    fireEvent.touchEnd(drawerPanel, {
      changedTouches: [{ clientX: 100, clientY: 100 }], // Swiped 100px left
    })
    expect(handleClose).toHaveBeenCalledTimes(2)

    // Closed state
    rerender(
      <ThemeProvider>
        <NavDrawer
          isOpen={false}
          onClose={handleClose}
          currentRoute="dashboard"
          onSelectRoute={handleSelectRoute}
        />
      </ThemeProvider>
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('BottomNav provides 4 primary route buttons', () => {
    const handleSelectRoute = vi.fn()

    render(
      <BottomNav
        currentRoute="runs"
        onSelectRoute={handleSelectRoute}
      />
    )

    const nav = screen.getByRole('navigation', { name: /mobile bottom navigation/i })
    expect(nav).toBeInTheDocument()

    const schedulesBtn = screen.getByRole('button', { name: 'Schedules' })
    fireEvent.click(schedulesBtn)
    expect(handleSelectRoute).toHaveBeenCalledWith('schedules')
  })
})
