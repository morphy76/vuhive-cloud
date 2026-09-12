import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { YamlEditor } from '@/components/editor/YamlEditor'

describe('YamlEditor', () => {
  const initialYaml = `version: "1.0"
default_scenario: standard_load
scenarios:
  standard_load:
    type: constant_vus
    vus: 50
    run_period: 60s
`

  it('renders editor with validation status and template selector', () => {
    render(
      <YamlEditor
        value={initialYaml}
        onChange={vi.fn()}
        showTemplates={true}
        ariaLabel="Scenario YAML Configuration"
      />
    )

    expect(screen.getByText(/valid yaml/i)).toBeInTheDocument()
    expect(screen.getByText(/load template/i)).toBeInTheDocument()
  })

  it('displays syntax error alert when invalid YAML is provided', () => {
    const invalidYaml = `version: "1.0"
execution:
  vus: 50
    duration: 60s
`
    render(
      <YamlEditor
        value={invalidYaml}
        onChange={vi.fn()}
      />
    )

    expect(screen.getAllByText(/syntax error/i).length).toBeGreaterThan(0)
  })

  it('triggers onSelectTemplate when a template is picked', () => {
    const onSelectTemplate = vi.fn()
    render(
      <YamlEditor
        value={initialYaml}
        onChange={vi.fn()}
        showTemplates={true}
        onSelectTemplate={onSelectTemplate}
      />
    )

    const templateBtn = screen.getByText(/load template/i)
    fireEvent.click(templateBtn)

    // Check template options are displayed
    expect(screen.getByText(/standard load test/i)).toBeInTheDocument()
    fireEvent.click(screen.getByText(/standard load test/i))
    expect(onSelectTemplate).toHaveBeenCalled()
  })
})
