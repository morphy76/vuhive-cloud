import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { Sidebar } from '../components/layout/Sidebar'
import { NavDrawer } from '../components/layout/NavDrawer'
import { BottomNav } from '../components/layout/BottomNav'
import { ThemeProvider } from '../context/ThemeContext'
import { TooltipProvider } from '../components/ui/tooltip'

const withProviders = (ui: React.ReactElement) => (
  <ThemeProvider>
    <TooltipProvider>{ui}</TooltipProvider>
  </ThemeProvider>
)

describe('Layout Components Unit Tests', () => {
  it('Sidebar renders in expanded state by default and handles route selection', () => {
    const handleSelectRoute = vi.fn()
    const handleToggleCollapse = vi.fn()

    const { rerender } = render(
      withProviders(
        <Sidebar
          currentRoute="dashboard"
          onSelectRoute={handleSelectRoute}
          isCollapsed={false}
          onToggleCollapse={handleToggleCollapse}
        />
      )
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
      withProviders(
        <Sidebar
          currentRoute="dashboard"
          onSelectRoute={handleSelectRoute}
          isCollapsed={true}
          onToggleCollapse={handleToggleCollapse}
        />
      )
    )
    expect(screen.getByRole('button', { name: /expand sidebar/i })).toBeInTheDocument()
    expect(screen.queryByText('Control Plane v0.1.0')).not.toBeInTheDocument()
  })

  it('Sidebar marks active item with aria-current="page"', () => {
    const handleSelectRoute = vi.fn()
    const handleToggleCollapse = vi.fn()

    render(
      withProviders(
        <Sidebar
          currentRoute="suites"
          onSelectRoute={handleSelectRoute}
          isCollapsed={false}
          onToggleCollapse={handleToggleCollapse}
        />
      )
    )

    const activeItem = screen.getByRole('button', { name: 'Suites' })
    expect(activeItem).toHaveAttribute('aria-current', 'page')

    const inactiveItem = screen.getByRole('button', { name: 'Dashboard' })
    expect(inactiveItem).not.toHaveAttribute('aria-current')
  })

  it('NavDrawer opens via Radix Dialog and closes on button click', async () => {
    const handleClose = vi.fn()
    const handleSelectRoute = vi.fn()

    render(
      withProviders(
        <NavDrawer
          isOpen={true}
          onClose={handleClose}
          currentRoute="dashboard"
          onSelectRoute={handleSelectRoute}
        />
      )
    )

    // Radix Dialog should render the close button
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /close navigation menu/i })).toBeInTheDocument()
    })

    // Click close button
    fireEvent.click(screen.getByRole('button', { name: /close navigation menu/i }))
    expect(handleClose).toHaveBeenCalledTimes(1)
  })

  it('NavDrawer renders Cookbook navigation button and triggers route change', async () => {
    const handleClose = vi.fn()
    const handleSelectRoute = vi.fn()

    render(
      withProviders(
        <NavDrawer
          isOpen={true}
          onClose={handleClose}
          currentRoute="dashboard"
          onSelectRoute={handleSelectRoute}
        />
      )
    )

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /cookbook/i })).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: /cookbook/i }))
    expect(handleSelectRoute).toHaveBeenCalledWith('cookbook')
    expect(handleClose).toHaveBeenCalledTimes(1)
  })

  it('BottomNav provides primary route buttons including Cookbook', () => {
    const handleSelectRoute = vi.fn()

    render(
      <BottomNav
        currentRoute="runs"
        onSelectRoute={handleSelectRoute}
      />
    )

    const nav = screen.getByRole('navigation', { name: /mobile bottom navigation/i })
    expect(nav).toBeInTheDocument()

    const cookbookBtn = screen.getByRole('button', { name: 'Cookbook' })
    fireEvent.click(cookbookBtn)
    expect(handleSelectRoute).toHaveBeenCalledWith('cookbook')
  })

  it('BottomNav marks active item with aria-current="page"', () => {
    const handleSelectRoute = vi.fn()

    render(
      <BottomNav
        currentRoute="runs"
        onSelectRoute={handleSelectRoute}
      />
    )

    const activeBtn = screen.getByRole('button', { name: 'Runs' })
    expect(activeBtn).toHaveAttribute('aria-current', 'page')

    const inactiveBtn = screen.getByRole('button', { name: 'Dashboard' })
    expect(inactiveBtn).not.toHaveAttribute('aria-current')
  })
})
