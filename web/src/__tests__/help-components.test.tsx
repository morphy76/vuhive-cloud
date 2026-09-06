import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { HelpTooltip } from '../components/help/HelpTooltip'
import { InfoBadge } from '../components/help/InfoBadge'
import { CreateSuiteDialog } from '../components/dialogs/CreateSuiteDialog'
import { TriggerRunDialog } from '../components/dialogs/TriggerRunDialog'
import { CreateScheduleDialog } from '../components/dialogs/CreateScheduleDialog'
import { TooltipProvider } from '../components/ui/tooltip'

afterEach(() => {
  cleanup()
})

const renderWithProvider = (ui: React.ReactElement) => {
  return render(<TooltipProvider delayDuration={0}>{ui}</TooltipProvider>)
}

describe('HelpTooltip Component', () => {
  it('renders an accessible trigger button with default aria-label', () => {
    renderWithProvider(
      <HelpTooltip text="Median latency (50th percentile)" />
    )
    const trigger = screen.getByRole('button', { name: /^more information$/i })
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveAttribute('type', 'button')
  })

  it('allows custom aria-label for contextual screen reader announcements', () => {
    renderWithProvider(
      <HelpTooltip text="CPU millicores" label="Help for CPU allocation" />
    )
    const trigger = screen.getByRole('button', { name: /^help for cpu allocation$/i })
    expect(trigger).toBeInTheDocument()
  })

  it('displays tooltip content when opened', () => {
    renderWithProvider(
      <HelpTooltip open={true} text="Target CPU architectures for ephemeral cross-compilation" />
    )
    expect(
      screen.getByText(/Target CPU architectures for ephemeral cross-compilation/i)
    ).toBeInTheDocument()
  })
})

describe('InfoBadge Component', () => {
  it('renders title and descriptive content with note role', () => {
    render(
      <InfoBadge
        variant="warning"
        title="AST Validation Caveats"
      >
        Must import github.com/morphy76/vuhive and use package scenario.
      </InfoBadge>
    )

    expect(screen.getByRole('note')).toBeInTheDocument()
    expect(screen.getByText('AST Validation Caveats')).toBeInTheDocument()
    expect(
      screen.getByText(/Must import github\.com\/morphy76\/vuhive and use package scenario/i)
    ).toBeInTheDocument()
  })

  it('supports all semantic variants (info, warning, error, success)', () => {
    const { rerender } = render(
      <InfoBadge variant="info" title="Info Note">
        Informational content
      </InfoBadge>
    )
    expect(screen.getByText('Info Note')).toBeInTheDocument()

    rerender(
      <InfoBadge variant="error" title="Error Warning">
        Prohibited package imported
      </InfoBadge>
    )
    expect(screen.getByText('Error Warning')).toBeInTheDocument()

    rerender(
      <InfoBadge variant="success" title="Ready to Run">
        Scenario validated successfully
      </InfoBadge>
    )
    expect(screen.getByText('Ready to Run')).toBeInTheDocument()
  })
})

describe('Contextual Documentation Dialogs', () => {
  it('CreateSuiteDialog provides AST validation guidance and architecture help', () => {
    const { unmount, rerender } = renderWithProvider(
      <CreateSuiteDialog open={true} onOpenChange={() => {}} />
    )

    // Dialog title & inputs
    expect(screen.getByText('New Test Suite')).toBeInTheDocument()
    expect(screen.getByLabelText(/^suite name$/i)).toBeInTheDocument()

    // AST validation InfoBadge
    expect(screen.getByText(/AST Static Analysis & Framework Enforcement/i)).toBeInTheDocument()
    expect(screen.getByText(/github\.com\/morphy76\/vuhive/i)).toBeInTheDocument()
    expect(screen.getByText(/os\/exec, syscall, unsafe/i)).toBeInTheDocument()

    // Architecture help tooltip
    expect(screen.getByRole('button', { name: /^help for target architecture$/i })).toBeInTheDocument()

    rerender(<CreateSuiteDialog open={false} onOpenChange={() => {}} />)
    unmount()
  })

  it('TriggerRunDialog provides runner resource sizing and barrier sync guidance', () => {
    const { unmount, rerender } = renderWithProvider(
      <TriggerRunDialog open={true} onOpenChange={() => {}} />
    )

    expect(screen.getByText('Execute Test Run')).toBeInTheDocument()
    expect(screen.getByLabelText(/^runner pods count$/i)).toBeInTheDocument()

    // Resource guidance badge
    expect(screen.getByText(/Kubernetes Runner Resource Allocation/i)).toBeInTheDocument()
    expect(screen.getByText(/Configure memory requests equal to limits/i)).toBeInTheDocument()

    // Help tooltips for technical parameters
    expect(screen.getByRole('button', { name: /^help for runner pods count$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^help for cpu allocation$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^help for memory allocation$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^help for node tolerations$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^help for start barrier$/i })).toBeInTheDocument()

    rerender(<TriggerRunDialog open={false} onOpenChange={() => {}} />)
    unmount()
  })

  it('CreateScheduleDialog provides CRON presets and UTC timezone context', () => {
    const { unmount, rerender } = renderWithProvider(
      <CreateScheduleDialog open={true} onOpenChange={() => {}} />
    )

    expect(screen.getByText('New Cron Schedule')).toBeInTheDocument()
    expect(screen.getByLabelText(/^cron expression$/i)).toBeInTheDocument()

    // Timezone caveat InfoBadge
    expect(screen.getByText(/UTC Timezone Execution/i)).toBeInTheDocument()

    // Help tooltips and presets
    expect(screen.getByRole('button', { name: /^help for cron expression$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^preset hourly/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^preset nightly/i })).toBeInTheDocument()

    rerender(<CreateScheduleDialog open={false} onOpenChange={() => {}} />)
    unmount()
  })
})
