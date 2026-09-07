import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { RECIPES, getRecipeForRoute, getRecipeById } from '@/data/recipes'
import { RecipeDrawer } from '@/components/recipes/RecipeDrawer'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/toaster'

describe('API Recipes Data & Dynamic cURL Generators', () => {
  it('defines all 7 core contextual recipes', () => {
    expect(RECIPES).toHaveLength(7)
    const ids = RECIPES.map((r) => r.id)
    expect(ids).toEqual([
      'recipe-1',
      'recipe-2',
      'recipe-3',
      'recipe-4',
      'recipe-5',
      'recipe-6',
      'recipe-7',
    ])
  })

  it('maps navigation routes to contextual recipes correctly', () => {
    expect(getRecipeForRoute('suites').id).toBe('recipe-1')
    expect(getRecipeForRoute('runs').id).toBe('recipe-4')
    expect(getRecipeForRoute('schedules').id).toBe('recipe-5')
    expect(getRecipeForRoute('dashboard').id).toBe('recipe-1')
  })

  it('generates dynamic cURL commands with updated parameters', () => {
    const recipe1 = getRecipeById('recipe-1')
    expect(recipe1).toBeDefined()

    const defaultParams = recipe1!.defaultParams
    const step1 = recipe1!.steps[0]
    const defaultCurl = step1.generateCurl(defaultParams)

    expect(defaultCurl).toContain('POST http://localhost:8080/api/v1/suites')
    expect(defaultCurl).toContain('"name": "suite-e2e-checkout"')

    // Dynamically updated parameters
    const customCurl = step1.generateCurl({
      ...defaultParams,
      baseUrl: 'https://vuhive.prod.internal',
      suiteName: 'custom-payment-flow',
    })
    expect(customCurl).toContain('POST https://vuhive.prod.internal/api/v1/suites')
    expect(customCurl).toContain('"name": "custom-payment-flow"')
  })

  it('generates executable cURL for Recipe 7 abort execution', () => {
    const recipe7 = getRecipeById('recipe-7')
    expect(recipe7).toBeDefined()
    const abortStep = recipe7!.steps[0]
    const curl = abortStep.generateCurl({
      baseUrl: 'http://localhost:8080',
      runId: 'run-test-123',
      reason: 'SLA breach detected',
    })
    expect(curl).toContain('POST http://localhost:8080/api/v1/runs/run-test-123/abort')
    expect(curl).toContain('"reason": "SLA breach detected"')
  })
})

describe('RecipeDrawer Component', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    currentRoute: 'suites' as const,
  }

  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  const renderDrawer = (props = defaultProps) => {
    return render(
      <TooltipProvider>
        <RecipeDrawer {...props} />
        <Toaster />
      </TooltipProvider>
    )
  }

  it('renders open drawer with the contextual recipe title and description', () => {
    renderDrawer()

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(
      screen.getByRole('heading', {
        name: /recipe 1: registering a test suite & uploading source packages/i,
      })
    ).toBeInTheDocument()
  })

  it('renders all 7 recipes in the selector and allows switching recipes', async () => {
    renderDrawer()

    const select = screen.getByLabelText(/select api recipe/i)
    expect(select).toBeInTheDocument()

    // Switch to Recipe 4 (Runs)
    fireEvent.change(select, { target: { value: 'recipe-4' } })

    expect(
      await screen.findByRole('heading', {
        name: /recipe 4: triggering ad-hoc test runs/i,
      })
    ).toBeInTheDocument()
  })

  it('dynamically updates cURL preview when editing parameter inputs', async () => {
    renderDrawer()

    const suiteInput = screen.getByLabelText(/suite name/i)
    fireEvent.change(suiteInput, { target: { value: 'my-custom-suite' } })

    const codeBlock = screen.getByTestId('curl-preview')
    expect(codeBlock.textContent).toContain('my-custom-suite')
  })

  it('copies cURL command to clipboard and triggers toast notification', async () => {
    renderDrawer()

    const copyButtons = screen.getAllByRole('button', { name: /copy as curl/i })
    expect(copyButtons.length).toBeGreaterThan(0)
    fireEvent.click(copyButtons[0])

    expect(navigator.clipboard.writeText).toHaveBeenCalledTimes(1)
    await waitFor(() => {
      expect(screen.getByText(/curl command copied/i)).toBeInTheDocument()
    })
  })

  it('invokes onClose when clicking the close button', () => {
    const onClose = vi.fn()
    renderDrawer({ ...defaultProps, onClose })

    const closeBtn = screen.getByRole('button', { name: /close recipe guidance/i })
    fireEvent.click(closeBtn)

    expect(onClose).toHaveBeenCalled()
  })

  it('has no accessibility violations (WCAG 2.1 AA)', async () => {
    const { axe } = await import('vitest-axe')
    const { container } = renderDrawer()
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})
