import { describe, it, expect } from 'vitest'
import { validateYaml, validateVuhiveSchema } from '@/lib/yaml-validator'

describe('YAML Schema Validator with ${secrets.KEY} placeholders', () => {
  it('validates constant_vus scenario using secret placeholders for vus and durations without false positives', () => {
    const yaml = `
version: "1.0"
default_scenario: secret_load
scenarios:
  secret_load:
    type: constant_vus
    vus: "\${secrets.MAX_VUS}"
    run_period: "\${secrets.DURATION}"
    ramp_up: "\${secrets.RAMP_UP}"
    vu_timeout: "\${secrets.TIMEOUT}"
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p95
        operator: "<"
        target: "\${secrets.MAX_LATENCY}"
`
    const valResult = validateYaml(yaml)
    expect(valResult.isValid).toBe(true)

    const schemaResult = validateVuhiveSchema(valResult.parsed)
    expect(schemaResult.errors).toEqual([])
    expect(schemaResult.isValid).toBe(true)
  })

  it('validates arrival_rate scenario using secret placeholders for tps, max_vus, and burst_buffer', () => {
    const yaml = `
version: "1.0"
scenarios:
  api_stress:
    type: arrival_rate
    target_tps: "\${secrets.TARGET_TPS}"
    max_vus: "\${secrets.BURST_VUS}"
    burst_buffer: "\${secrets.BURST_BUFFER}"
    run_period: "60s"
`
    const valResult = validateYaml(yaml)
    expect(valResult.isValid).toBe(true)

    const schemaResult = validateVuhiveSchema(valResult.parsed)
    expect(schemaResult.errors).toEqual([])
    expect(schemaResult.isValid).toBe(true)
  })

  it('validates ramping_vus stages with secret placeholders for targets and durations', () => {
    const yaml = `
version: "1.0"
scenarios:
  staged:
    type: ramping_vus
    stages:
      - target: "\${secrets.STAGE_1_TARGET}"
        duration: "\${secrets.STAGE_1_DURATION}"
      - target: 50
        duration: "30s"
`
    const valResult = validateYaml(yaml)
    expect(valResult.isValid).toBe(true)

    const schemaResult = validateVuhiveSchema(valResult.parsed)
    expect(schemaResult.errors).toEqual([])
    expect(schemaResult.isValid).toBe(true)
  })

  it('still rejects truly invalid configs (e.g. invalid type or missing required fields)', () => {
    const yaml = `
version: "1.0"
scenarios:
  invalid_scenario:
    type: unknown_type
`
    const valResult = validateYaml(yaml)
    const schemaResult = validateVuhiveSchema(valResult.parsed)
    expect(schemaResult.isValid).toBe(false)
    expect(schemaResult.errors.some((e) => e.includes('invalid type'))).toBe(true)
  })
})
