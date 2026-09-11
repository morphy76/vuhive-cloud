import { describe, it, expect } from 'vitest'
import {
  validateYaml,
  validateVuhiveSchema,
  VUHIVE_YAML_TEMPLATES,
  formatYamlError,
} from '@/lib/yaml-validator'

describe('YAML Validator', () => {
  it('identifies valid YAML without errors', () => {
    const yaml = `
version: "1.0"
execution:
  vus: 50
  duration: 60s
  ramp_up: 10s
thresholds:
  p95_latency_ms: 250
  error_rate_pct: 1.0
`
    const result = validateYaml(yaml)
    expect(result.isValid).toBe(true)
    expect(result.errors).toHaveLength(0)
    expect(result.parsed).toBeDefined()
  })

  it('detects syntax errors and extracts line and column numbers', () => {
    const invalidYaml = `
version: "1.0"
execution:
  vus: 50
    duration: 60s # bad indentation
`
    const result = validateYaml(invalidYaml)
    expect(result.isValid).toBe(false)
    expect(result.errors.length).toBeGreaterThan(0)
    expect(result.errors[0].line).toBeGreaterThan(0)
    expect(result.errors[0].message).toBeTruthy()
  })

  it('handles empty input gracefully', () => {
    const result = validateYaml('')
    expect(result.isValid).toBe(false)
    expect(result.errors[0].message).toContain('empty')
  })

  it('formats error message cleanly for UI display', () => {
    const formatted = formatYamlError({
      message: 'Bad indentation',
      line: 4,
      column: 5,
    })
    expect(formatted).toBe('Line 4, Col 5: Bad indentation')
  })
})

describe('Vuhive Schema Validator', () => {
  it('validates compliant vuhive.yaml schema', () => {
    const yaml = `
version: "1.0"
execution:
  vus: 50
  duration: 60s
  ramp_up: 10s
thresholds:
  p95_latency_ms: 250
  error_rate_pct: 1.0
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(true)
    expect(schemaResult.errors).toHaveLength(0)
  })

  it('warns when execution section is missing', () => {
    const yaml = `
version: "1.0"
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.warnings).toContain('Missing recommended "execution" configuration block')
  })

  it('reports error when vus is non-numeric or negative', () => {
    const yaml = `
version: "1.0"
execution:
  vus: -5
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('vus'))).toBe(true)
  })

  it('provides predefined valid templates', () => {
    expect(VUHIVE_YAML_TEMPLATES.length).toBeGreaterThanOrEqual(3)
    for (const template of VUHIVE_YAML_TEMPLATES) {
      expect(template.name).toBeTruthy()
      expect(template.description).toBeTruthy()
      const result = validateYaml(template.yaml)
      expect(result.isValid).toBe(true)
      const schema = validateVuhiveSchema(result.parsed)
      expect(schema.isValid).toBe(true)
    }
  })
})
