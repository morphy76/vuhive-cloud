import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { YamlDiffViewer } from '@/components/editor/YamlDiffViewer'

describe('YamlDiffViewer', () => {
  const baseYaml = `version: "1.0"
execution:
  vus: 50
  duration: 60s
`

  const comparisonYaml = `version: "1.0"
execution:
  vus: 100
  duration: 120s
`

  it('renders diff viewer with side-by-side titles and diff statistics', () => {
    render(
      <YamlDiffViewer
        baseYaml={baseYaml}
        comparisonYaml={comparisonYaml}
        baseTitle="v1 (Original)"
        comparisonTitle="v2 (Modified)"
      />
    )

    expect(screen.getByText('v1 (Original)')).toBeInTheDocument()
    expect(screen.getByText('v2 (Modified)')).toBeInTheDocument()
    expect(screen.getByLabelText('YAML Difference Viewer')).toBeInTheDocument()
  })

  it('identifies identical content with no changes alert', () => {
    render(
      <YamlDiffViewer
        baseYaml={baseYaml}
        comparisonYaml={baseYaml}
        baseTitle="Base"
        comparisonTitle="Target"
      />
    )

    expect(screen.getByText(/configurations are identical/i)).toBeInTheDocument()
  })

  it('renders added and removed line indicators', () => {
    render(
      <YamlDiffViewer
        baseYaml={baseYaml}
        comparisonYaml={comparisonYaml}
      />
    )

    // Should display added and removed lines
    expect(screen.getByText(/vus: 50/i)).toBeInTheDocument()
    expect(screen.getByText(/vus: 100/i)).toBeInTheDocument()
  })
})
