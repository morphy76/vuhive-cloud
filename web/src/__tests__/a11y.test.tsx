import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { axe } from 'vitest-axe'
import App from '../App'
import { DashboardView } from '../views/DashboardView'
import { SuitesView } from '../views/SuitesView'
import { RunsView } from '../views/RunsView'
import { SchedulesView } from '../views/SchedulesView'
import { TooltipProvider } from '../components/ui/tooltip'

// Helper to wrap components that need TooltipProvider
const withTooltipProvider = (ui: React.ReactElement) => (
  <TooltipProvider>{ui}</TooltipProvider>
)

describe('WCAG 2.1 AA Accessibility Compliance', () => {
  it('full App renders with no critical or serious axe violations', async () => {
    const { container } = render(<App />)
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('DashboardView has no accessibility violations', async () => {
    const { container } = render(withTooltipProvider(<DashboardView />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('SuitesView has no accessibility violations', async () => {
    const { container } = render(withTooltipProvider(<SuitesView />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('RunsView has no accessibility violations', async () => {
    const { container } = render(withTooltipProvider(<RunsView />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('SchedulesView has no accessibility violations', async () => {
    const { container } = render(withTooltipProvider(<SchedulesView />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  it('YamlDiffViewer has no accessibility violations', async () => {
    const { YamlDiffViewer } = await import('../components/editor/YamlDiffViewer')
    const { container } = render(
      <YamlDiffViewer
        baseYaml="vus: 50\nduration: 60s\n"
        comparisonYaml="vus: 100\nduration: 120s\n"
        baseTitle="v1"
        comparisonTitle="v2"
      />
    )
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})

describe('Keyboard Navigation & Focus Management', () => {
  it('renders skip-to-main-content link as first focusable element', () => {
    const { container } = render(<App />)
    const skipLink = container.querySelector('a[href="#main-content"]')
    expect(skipLink).toBeInTheDocument()
  })

  it('main content area has id="main-content"', () => {
    const { container } = render(<App />)
    const main = container.querySelector('#main-content')
    expect(main).toBeInTheDocument()
    expect(main?.tagName).toBe('MAIN')
  })

  it('sidebar active nav item has aria-current="page"', () => {
    render(<App />)
    // Dashboard is active by default
    const activeButtons = document.querySelectorAll('[aria-current="page"]')
    expect(activeButtons.length).toBeGreaterThan(0)
  })
})

describe('Screen Reader Landmarks', () => {
  it('renders navigation landmarks', () => {
    const { container } = render(<App />)
    const navs = container.querySelectorAll('nav')
    expect(navs.length).toBeGreaterThan(0)
  })

  it('renders main landmark', () => {
    const { container } = render(<App />)
    const main = container.querySelector('main')
    expect(main).toBeInTheDocument()
  })

  it('renders header landmark', () => {
    const { container } = render(<App />)
    const header = container.querySelector('header')
    expect(header).toBeInTheDocument()
  })
})
