import { describe, it, expect } from 'vitest'
import {
  validateYaml,
  validateVuhiveSchema,
  VUHIVE_YAML_TEMPLATES,
  formatYamlError,
} from '@/lib/yaml-validator'

describe('YAML Syntax Validator', () => {
  it('identifies valid YAML without errors', () => {
    const yaml = `
# yaml-language-server: $schema=https://raw.githubusercontent.com/morphy76/vuhive/main/schemas/vuhive.schema.json
version: "1.0"
default_scenario: standard_load

scenarios:
  standard_load:
    type: constant_vus
    vus: 50
    run_period: 60s
    ramp_up: 10s
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p95
        operator: "<"
        target: 250ms
      - metric: vuhive.http.req_failed
        stat: rate
        operator: "<="
        target: "0.01"
`
    const result = validateYaml(yaml)
    expect(result.isValid).toBe(true)
    expect(result.errors).toHaveLength(0)
    expect(result.parsed).toBeDefined()
  })

  it('detects syntax errors and extracts line and column numbers', () => {
    const invalidYaml = `
version: "1.0"
scenarios:
  standard_load:
    vus: 50
      run_period: 60s # bad indentation
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

describe('Vuhive Schema Validator (SDK v1.1.5 specification)', () => {
  it('validates compliant constant_vus scenario', () => {
    const yaml = `
version: "1.0"
default_scenario: standard_load
scenarios:
  standard_load:
    type: constant_vus
    vus: 50
    ramp_up: 10s
    run_period: 60s
    ramp_down: 5s
    vu_timeout: 5s
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p95
        operator: "<"
        target: 250ms
      - metric: vuhive.http.req_failed
        stat: rate
        operator: "<="
        target: "0.01"
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(true)
    expect(schemaResult.errors).toHaveLength(0)
  })

  it('validates compliant arrival_rate scenario', () => {
    const yaml = `
version: "1.0"
default_scenario: api_throughput
scenarios:
  api_throughput:
    type: arrival_rate
    target_tps: 100
    max_vus: 50
    run_period: 2m
    burst_buffer: 20
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p99
        operator: "<="
        target: 500ms
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(true)
    expect(schemaResult.errors).toHaveLength(0)
  })

  it('validates compliant ramping_vus scenario with stages', () => {
    const yaml = `
version: "1.0"
default_scenario: spike_test
scenarios:
  spike_test:
    type: ramping_vus
    stages:
      - target: 10
        duration: 30s
      - target: 50
        duration: 1m
      - target: 0
        duration: 30s
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p95
        operator: "<"
        target: 300ms
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(true)
    expect(schemaResult.errors).toHaveLength(0)
  })

  it('detects legacy custom "execution:" format and provides helpful error message', () => {
    const legacyYaml = `
version: "1.0"
execution:
  vus: 50
  duration: 60s
  ramp_up: 10s
thresholds:
  p95_latency_ms: 250
`
    const validation = validateYaml(legacyYaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.toLowerCase().includes('scenarios'))).toBe(true)
    expect(schemaResult.errors.some((e) => e.includes('execution'))).toBe(true)
  })

  it('rejects missing or empty scenarios section', () => {
    const yaml = `
version: "1.0"
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('scenarios'))).toBe(true)
  })

  it('rejects unknown scenario type', () => {
    const yaml = `
version: "1.0"
scenarios:
  test_scenario:
    type: unknown_mode
    run_period: 30s
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('unknown_mode'))).toBe(true)
  })

  it('reports error when constant_vus is missing vus or run_period', () => {
    const yaml = `
version: "1.0"
scenarios:
  bad_constant:
    type: constant_vus
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('vus'))).toBe(true)
    expect(schemaResult.errors.some((e) => e.includes('run_period'))).toBe(true)
  })

  it('reports error when arrival_rate is missing target_tps or max_vus', () => {
    const yaml = `
version: "1.0"
scenarios:
  bad_arrival:
    type: arrival_rate
    run_period: 30s
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('target_tps'))).toBe(true)
    expect(schemaResult.errors.some((e) => e.includes('max_vus'))).toBe(true)
  })

  it('reports error when ramping_vus is missing stages or has invalid stage targets', () => {
    const yaml = `
version: "1.0"
scenarios:
  bad_ramping:
    type: ramping_vus
    stages:
      - target: -5
        duration: "invalid-duration"
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('target') || e.includes('duration'))).toBe(true)
  })

  it('reports error on invalid duration format (missing units)', () => {
    const yaml = `
version: "1.0"
scenarios:
  invalid_duration:
    type: constant_vus
    vus: 10
    run_period: 60
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('run_period'))).toBe(true)
  })

  it('validates threshold SLA fields and rejects invalid operators or stats', () => {
    const yaml = `
version: "1.0"
scenarios:
  test_thresholds:
    type: constant_vus
    vus: 10
    run_period: 30s
    thresholds:
      - metric: "http.latency"
        stat: "invalid_stat"
        operator: "=="
        target: "200ms"
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('stat') || e.includes('operator'))).toBe(true)
  })

  it('warns when default_scenario does not exist in scenarios mapping', () => {
    const yaml = `
version: "1.0"
default_scenario: non_existent
scenarios:
  load_test:
    type: constant_vus
    vus: 10
    run_period: 30s
`
    const validation = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(validation.parsed)
    expect(schemaResult.warnings.some((w) => w.includes('non_existent'))).toBe(true)
  })

  it('provides predefined valid templates conforming to SDK v1.1.5 specification', () => {
    expect(VUHIVE_YAML_TEMPLATES.length).toBeGreaterThanOrEqual(3)
    const templateIds = VUHIVE_YAML_TEMPLATES.map((t) => t.id)
    expect(templateIds).toContain('standard-load')
    expect(templateIds).toContain('staged-progression')
    expect(templateIds).toContain('stress-peak')

    for (const template of VUHIVE_YAML_TEMPLATES) {
      expect(template.name).toBeTruthy()
      expect(template.description).toBeTruthy()
      const result = validateYaml(template.yaml)
      expect(result.isValid).toBe(true)
      const schema = validateVuhiveSchema(result.parsed)
      expect(schema.isValid).toBe(true)
      expect(schema.errors).toHaveLength(0)
    }
  })
})
