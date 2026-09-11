import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import App from '../App'

describe('App & Responsive Shell Layout', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.className = ''
  })

  it('renders application brand and default Dashboard view', () => {
    render(<App />)
    expect(screen.getAllByText('vuhive-cloud').length).toBeGreaterThan(0)
    expect(screen.getByText('Control Plane Overview')).toBeInTheDocument()
    expect(screen.getByText('Active Runners')).toBeInTheDocument()
  })

  it('switches navigation between primary views', () => {
    render(<App />)

    // Navigate to Suites
    const suitesButtons = screen.getAllByRole('button', { name: /suites/i })
    fireEvent.click(suitesButtons[0])
    expect(screen.getByText('Test Suites')).toBeInTheDocument()
    expect(screen.getByText(/Source Packages & Artifacts/i)).toBeInTheDocument()

    // Navigate to Runs
    const runsButtons = screen.getAllByRole('button', { name: /runs/i })
    fireEvent.click(runsButtons[0])
    expect(screen.getByText('Execution Runs')).toBeInTheDocument()
    expect(screen.getByText(/Live & Historical Test Runs/i)).toBeInTheDocument()

    // Navigate to Schedules
    const schedulesButtons = screen.getAllByRole('button', { name: /schedules/i })
    fireEvent.click(schedulesButtons[0])
    expect(screen.getByText('Cron Schedules')).toBeInTheDocument()
    expect(screen.getByText(/Native Kubernetes CronJob Schedules/i)).toBeInTheDocument()

    // Navigate to Profiles
    const profilesButtons = screen.getAllByRole('button', { name: /profiles/i })
    fireEvent.click(profilesButtons[0])
    expect(screen.getByRole('heading', { name: /runner profiles/i })).toBeInTheDocument()

    // Back to Dashboard
    const dashboardButtons = screen.getAllByRole('button', { name: /dashboard/i })
    fireEvent.click(dashboardButtons[0])
    expect(screen.getByText('Control Plane Overview')).toBeInTheDocument()
  })

  it('toggles dark and light mode themes', () => {
    render(<App />)
    const themeButton = screen.getByRole('button', { name: /switch to dark mode/i })
    expect(themeButton).toBeInTheDocument()

    // Initially light (or default)
    expect(document.documentElement.classList.contains('dark')).toBe(false)

    // Toggle to dark
    fireEvent.click(themeButton)
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    // Toggle back to light
    const lightButton = screen.getByRole('button', { name: /switch to light mode/i })
    fireEvent.click(lightButton)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('collapses and expands desktop sidebar', () => {
    render(<App />)
    const collapseToggle = screen.getByRole('button', { name: /collapse sidebar/i })
    expect(collapseToggle).toBeInTheDocument()

    // Collapse
    fireEvent.click(collapseToggle)
    expect(screen.getByRole('button', { name: /expand sidebar/i })).toBeInTheDocument()

    // Expand
    const expandToggle = screen.getByRole('button', { name: /expand sidebar/i })
    fireEvent.click(expandToggle)
    expect(screen.getByRole('button', { name: /collapse sidebar/i })).toBeInTheDocument()
  })

  it('opens and closes navigation drawer via Radix Dialog', async () => {
    render(<App />)
    const openMenuButton = screen.getByRole('button', { name: /open navigation menu/i })
    expect(openMenuButton).toBeInTheDocument()

    // Open drawer
    fireEvent.click(openMenuButton)
    
    // Radix Dialog renders via portal, so we need to check the document
    await waitFor(() => {
      // The dialog content with close button should be visible
      expect(screen.getByRole('button', { name: /close navigation menu/i })).toBeInTheDocument()
    })

    // Close drawer
    const closeButton = screen.getByRole('button', { name: /close navigation menu/i })
    fireEvent.click(closeButton)
    
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /close navigation menu/i })).not.toBeInTheDocument()
    })
  })

  it('renders mobile bottom navigation bar', () => {
    render(<App />)
    const bottomNav = screen.getByRole('navigation', { name: /mobile bottom navigation/i })
    expect(bottomNav).toBeInTheDocument()
  })

  it('renders skip-to-main-content link', () => {
    const { container } = render(<App />)
    const skipLink = container.querySelector('a[href="#main-content"]')
    expect(skipLink).toBeInTheDocument()
    expect(skipLink?.textContent).toContain('Skip to main content')
  })
})
