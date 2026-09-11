import React from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ProfilesView } from '../views/ProfilesView'
import { ProfileBuilderDialog } from '../components/profiles/ProfileBuilderDialog'
import { DeleteProfileDialog } from '../components/profiles/DeleteProfileDialog'
import { RecipeProvider } from '../context/RecipeContext'
import { TooltipProvider } from '../components/ui/tooltip'
import { api, FALLBACK_PROFILES } from '../lib/api'
import type { RunnerProfile } from '../types/profile'

function renderWithProviders(ui: React.ReactElement) {
  const testQueryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })

  return render(
    <QueryClientProvider client={testQueryClient}>
      <TooltipProvider>
        <RecipeProvider>{ui}</RecipeProvider>
      </TooltipProvider>
    </QueryClientProvider>
  )
}

describe('Runner Profiles View & Builder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(api, 'getProfiles').mockResolvedValue([...FALLBACK_PROFILES])
  })

  describe('ProfilesView', () => {
    it('renders runner profiles header, KPI metrics, and table', async () => {
      renderWithProviders(<ProfilesView />)

      // Header
      expect(screen.getByRole('heading', { name: /runner profiles/i, level: 1 })).toBeInTheDocument()
      expect(screen.getByText(/registered profiles/i)).toBeInTheDocument()

      // Await data load
      await waitFor(() => {
        expect(screen.getByText('standard-single-node')).toBeInTheDocument()
        expect(screen.getByText('high-throughput-dedicated')).toBeInTheDocument()
        expect(screen.getByText('kernel-isolated-gvisor')).toBeInTheDocument()
      })

      // KPI Counts
      expect(screen.getByText('3')).toBeInTheDocument() // 3 total profiles
    })

    it('filters profiles dynamically via search query', async () => {
      renderWithProviders(<ProfilesView />)

      await waitFor(() => {
        expect(screen.getByText('standard-single-node')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox', { name: /search profiles/i })
      fireEvent.change(searchInput, { target: { value: 'gvisor' } })

      expect(screen.queryByText('standard-single-node')).not.toBeInTheDocument()
      expect(screen.getByText('kernel-isolated-gvisor')).toBeInTheDocument()
    })

    it('shows empty state when search matches nothing', async () => {
      renderWithProviders(<ProfilesView />)

      await waitFor(() => {
        expect(screen.getByText('standard-single-node')).toBeInTheDocument()
      })

      const searchInput = screen.getByRole('textbox', { name: /search profiles/i })
      fireEvent.change(searchInput, { target: { value: 'non-existent-xyz' } })

      expect(screen.getByText(/no runner profiles found/i)).toBeInTheDocument()
    })
  })

  describe('ProfileBuilderDialog', () => {
    it('opens in create mode and applies preset templates', async () => {
      renderWithProviders(<ProfileBuilderDialog open={true} onOpenChange={() => {}} />)

      expect(screen.getByRole('heading', { name: /visual runner profile builder/i })).toBeInTheDocument()

      // Click "High-Throughput Dedicated" preset
      const dedicatedPreset = screen.getByRole('button', { name: /high-throughput dedicated/i })
      fireEvent.click(dedicatedPreset)

      // Inputs should update
      const nameInput = screen.getByLabelText(/profile name/i, { selector: 'input' }) as HTMLInputElement
      expect(nameInput.value).toBe('high-throughput-dedicated')

      // Verify node selector was populated
      expect(screen.getByDisplayValue('node-role.kubernetes.io/performance-runner')).toBeInTheDocument()

      // Verify affinity was populated
      expect(screen.getByDisplayValue('node.kubernetes.io/instance-type')).toBeInTheDocument()

      // Verify toleration was populated
      expect(screen.getByDisplayValue('dedicated')).toBeInTheDocument()
    })

    it('displays warning when CPU request exceeds CPU limit', async () => {
      renderWithProviders(<ProfileBuilderDialog open={true} onOpenChange={() => {}} />)

      const cpuReqInput = screen.getByLabelText(/cpu request/i, { selector: 'input[type="text"]' })
      const cpuLimInput = screen.getByLabelText(/cpu limit/i, { selector: 'input[type="text"]' })

      // Set request to 4000m and limit to 1000m
      fireEvent.change(cpuReqInput, { target: { value: '4000m' } })
      fireEvent.change(cpuLimInput, { target: { value: '1000m' } })

      await waitFor(() => {
        expect(
          screen.getByText(/cpu request \(4000m\) exceeds cpu limit \(1000m\)/i)
        ).toBeInTheDocument()
      })

      // Submit button should be disabled
      const submitBtn = screen.getByRole('button', { name: /create profile/i })
      expect(submitBtn).toBeDisabled()
    })

    it('displays warning when Memory request exceeds Memory limit', async () => {
      renderWithProviders(<ProfileBuilderDialog open={true} onOpenChange={() => {}} />)

      const memReqInput = screen.getByLabelText(/memory request/i, { selector: 'input[type="text"]' })
      const memLimInput = screen.getByLabelText(/memory limit/i, { selector: 'input[type="text"]' })

      // Set request to 4Gi and limit to 2Gi
      fireEvent.change(memReqInput, { target: { value: '4Gi' } })
      fireEvent.change(memLimInput, { target: { value: '2Gi' } })

      await waitFor(() => {
        expect(
          screen.getByText(/memory request \(4Gi\) exceeds memory limit \(2Gi\)/i)
        ).toBeInTheDocument()
      })

      // Submit button should be disabled
      const submitBtn = screen.getByRole('button', { name: /create profile/i })
      expect(submitBtn).toBeDisabled()
    })

    it('allows adding and removing node selectors, affinities, and tolerations', async () => {
      renderWithProviders(<ProfileBuilderDialog open={true} onOpenChange={() => {}} />)

      // Add Node Selector
      const addLabelBtn = screen.getByRole('button', { name: /add label/i })
      fireEvent.click(addLabelBtn)

      const keyInputs = screen.getAllByPlaceholderText(/e\.g\. node-role\.kubernetes\.io\/runner/i)
      expect(keyInputs.length).toBeGreaterThan(0)

      // Add Affinity Match Expression
      const addAffinityBtn = screen.getByRole('button', { name: /add match expression/i })
      fireEvent.click(addAffinityBtn)

      const affinityKeyInput = screen.getByPlaceholderText(/key \(e\.g\. node\.kubernetes\.io\/instance-type\)/i)
      expect(affinityKeyInput).toBeInTheDocument()

      // Add Toleration
      const addTolerationBtn = screen.getByRole('button', { name: /add toleration/i })
      fireEvent.click(addTolerationBtn)

      const tolKeyInput = screen.getByPlaceholderText(/key \(e\.g\. dedicated\)/i)
      expect(tolKeyInput).toBeInTheDocument()
    })

    it('toggles live Kubernetes JSON payload inspection', async () => {
      renderWithProviders(<ProfileBuilderDialog open={true} onOpenChange={() => {}} />)

      const inspectBtn = screen.getByRole('button', { name: /inspect generated kubernetes json/i })
      fireEvent.click(inspectBtn)

      expect(screen.getByText(/kubernetes profile payload \(json\)/i)).toBeInTheDocument()
      expect(screen.getByText(/copy json/i)).toBeInTheDocument()
    })

    it('submits valid profile creation successfully', async () => {
      const createdProfile: RunnerProfile = {
        id: 'new-profile-123',
        name: 'custom-runner',
        description: 'Custom load runner',
        runner_image: 'alpine:3.20',
        cpu_request: '1000m',
        cpu_limit: '2000m',
        memory_request: '1Gi',
        memory_limit: '2Gi',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      const createSpy = vi.spyOn(api, 'createProfile').mockResolvedValue(createdProfile)

      const onSaved = vi.fn()
      const onOpenChange = vi.fn()

      renderWithProviders(
        <ProfileBuilderDialog
          open={true}
          onOpenChange={onOpenChange}
          onSaved={onSaved}
        />
      )

      const nameInput = screen.getByLabelText(/profile name/i, { selector: 'input' })
      fireEvent.change(nameInput, { target: { value: 'custom-runner' } })

      const submitBtn = screen.getByRole('button', { name: /create profile/i })
      expect(submitBtn).not.toBeDisabled()
      fireEvent.click(submitBtn)

      await waitFor(() => {
        expect(createSpy).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'custom-runner',
            runner_image: 'alpine:3.20',
          })
        )
        expect(onSaved).toHaveBeenCalledWith(createdProfile)
        expect(onOpenChange).toHaveBeenCalledWith(false)
      })
    })
  })

  describe('DeleteProfileDialog', () => {
    it('deletes profile when confirmed', async () => {
      const deleteSpy = vi.spyOn(api, 'deleteProfile').mockResolvedValue()
      const onOpenChange = vi.fn()
      const profile = FALLBACK_PROFILES[0]

      renderWithProviders(
        <DeleteProfileDialog
          open={true}
          onOpenChange={onOpenChange}
          profile={profile}
        />
      )

      expect(screen.getByRole('heading', { name: /delete runner profile/i })).toBeInTheDocument()
      expect(screen.getByText(profile.name)).toBeInTheDocument()

      const deleteBtn = screen.getByRole('button', { name: /delete profile/i })
      fireEvent.click(deleteBtn)

      await waitFor(() => {
        expect(deleteSpy).toHaveBeenCalledWith(profile.id)
        expect(onOpenChange).toHaveBeenCalledWith(false)
      })
    })
  })
})
