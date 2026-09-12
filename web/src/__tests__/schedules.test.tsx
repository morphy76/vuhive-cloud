import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SchedulesView } from '@/views/SchedulesView'
import { CronBuilder } from '@/components/schedules/CronBuilder'
import { api, FALLBACK_SCHEDULES, FALLBACK_RUNS, FALLBACK_SUITES } from '@/lib/api'
import { RecipeProvider } from '@/context/RecipeContext'
import { TooltipProvider } from '@/components/ui/tooltip'

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={0}>
        <RecipeProvider>{ui}</RecipeProvider>
      </TooltipProvider>
    </QueryClientProvider>
  )
}

describe('CronBuilder Component', () => {
  it('renders presets and updates expression when preset is selected', () => {
    const onChange = vi.fn()
    renderWithProviders(<CronBuilder value="0 2 * * *" onChange={onChange} />)

    expect(screen.getByText(/Nightly/i)).toBeInTheDocument()
    expect(screen.getByText(/Hourly/i)).toBeInTheDocument()
    expect(screen.getByText(/Every Monday morning/i)).toBeInTheDocument()

    // Click Hourly preset
    const hourlyBtn = screen.getByRole('button', { name: /preset hourly/i })
    fireEvent.click(hourlyBtn)

    expect(onChange).toHaveBeenCalledWith('0 * * * *')
  })

  it('displays natural language preview and next run time', () => {
    renderWithProviders(<CronBuilder value="0 2 * * *" onChange={vi.fn()} />)
    expect(screen.getAllByText(/Every day at 02:00 UTC/i).length).toBeGreaterThan(0)
    expect(screen.getByText(/Next scheduled run:/i)).toBeInTheDocument()
  })

  it('validates raw cron input and shows error for invalid expressions', () => {
    renderWithProviders(<CronBuilder value="bad-cron" onChange={vi.fn()} />)
    expect(screen.getByText(/Invalid cron expression/i)).toBeInTheDocument()
  })
})

describe('SchedulesView Component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(api, 'getSchedules').mockResolvedValue([...FALLBACK_SCHEDULES])
    vi.spyOn(api, 'getSuites').mockResolvedValue([...FALLBACK_SUITES])
    vi.spyOn(api, 'getRuns').mockResolvedValue([...FALLBACK_RUNS])
  })

  it('renders schedules table with names, suites, and badges', async () => {
    renderWithProviders(<SchedulesView />)

    expect(screen.getByRole('heading', { name: /Cron Schedules/i })).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText('Nightly Soak Test (2h)')).toBeInTheDocument()
      expect(screen.getByText('Hourly Performance Canary')).toBeInTheDocument()
      expect(screen.getByText('Weekend Massive Concurrency')).toBeInTheDocument()
    })

    // Active and Paused badges
    expect(screen.getAllByText('ACTIVE').length).toBeGreaterThan(0)
    expect(screen.getByText('PAUSED')).toBeInTheDocument()
  })

  it('filters schedules by search query', async () => {
    renderWithProviders(<SchedulesView />)

    await waitFor(() => {
      expect(screen.getByText('Nightly Soak Test (2h)')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search schedules/i)
    fireEvent.change(searchInput, { target: { value: 'Canary' } })

    expect(screen.queryByText('Nightly Soak Test (2h)')).not.toBeInTheDocument()
    expect(screen.getByText('Hourly Performance Canary')).toBeInTheDocument()
  })

  it('filters schedules by status (Active / Paused)', async () => {
    renderWithProviders(<SchedulesView />)

    await waitFor(() => {
      expect(screen.getByText('Nightly Soak Test (2h)')).toBeInTheDocument()
    })

    const statusFilter = screen.getByRole('combobox', { name: /filter by status/i })
    fireEvent.change(statusFilter, { target: { value: 'PAUSED' } })

    expect(screen.queryByText('Nightly Soak Test (2h)')).not.toBeInTheDocument()
    expect(screen.getByText('Weekend Massive Concurrency')).toBeInTheDocument()
  })

  it('handles pause and resume actions', async () => {
    const updateSpy = vi.spyOn(api, 'updateSchedule').mockResolvedValue({
      ...FALLBACK_SCHEDULES[0],
      isActive: false,
    })

    renderWithProviders(<SchedulesView />)

    await waitFor(() => {
      expect(screen.getByText('Nightly Soak Test (2h)')).toBeInTheDocument()
    })

    // Pause first active schedule
    const pauseButtons = screen.getAllByRole('button', { name: /pause schedule/i })
    fireEvent.click(pauseButtons[0])

    await waitFor(() => {
      expect(updateSpy).toHaveBeenCalledWith('sched-nightly-soak', { is_active: false })
    })
  })

  it('handles Run Now ad-hoc trigger action', async () => {
    const triggerSpy = vi.spyOn(api, 'triggerRun').mockResolvedValue(FALLBACK_RUNS[0])

    renderWithProviders(<SchedulesView />)

    await waitFor(() => {
      expect(screen.getByText('Nightly Soak Test (2h)')).toBeInTheDocument()
    })

    const runNowButtons = screen.getAllByRole('button', { name: /run now/i })
    fireEvent.click(runNowButtons[0])

    await waitFor(() => {
      expect(triggerSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          suite_id: 'suite-e2e-checkout',
          artifact_id: 'art-suite-e2e-checkout-arm64',
          runner_profile_id: 'profile-standard-single-node',
        })
      )
    })
  })

  it('opens execution history drawer when clicking history button', async () => {
    renderWithProviders(<SchedulesView />)

    await waitFor(() => {
      expect(screen.getByText('Nightly Soak Test (2h)')).toBeInTheDocument()
    })

    const historyButtons = screen.getAllByRole('button', { name: /view execution history/i })
    fireEvent.click(historyButtons[0])

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /Execution History/i })).toBeInTheDocument()
    })
  })
})
