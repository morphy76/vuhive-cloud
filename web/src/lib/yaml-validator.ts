import YAML from 'yaml'

export interface YamlErrorDetail {
  message: string
  line?: number
  column?: number
}

export interface YamlValidationResult {
  isValid: boolean
  errors: YamlErrorDetail[]
  parsed?: any
}

export interface SchemaValidationResult {
  isValid: boolean
  errors: string[]
  warnings: string[]
}

export interface VuhiveYamlTemplate {
  id: string
  name: string
  description: string
  yaml: string
}

export const VUHIVE_YAML_TEMPLATES: VuhiveYamlTemplate[] = [
  {
    id: 'standard-load',
    name: 'Standard Load Test',
    description: 'Constant virtual users with a smooth ramp-up curve and latency SLA.',
    yaml: `version: "1.0"
execution:
  vus: 50
  duration: 60s
  ramp_up: 10s
thresholds:
  p95_latency_ms: 250
  error_rate_pct: 1.0
`,
  },
  {
    id: 'staged-progression',
    name: 'Staged Step Progression',
    description: 'Multi-stage ramp up transitioning through load plateaus.',
    yaml: `version: "1.0"
execution:
  stages:
    - duration: 30s
      target_vus: 10
    - duration: 60s
      target_vus: 50
    - duration: 30s
      target_vus: 0
thresholds:
  p95_latency_ms: 300
  p99_latency_ms: 500
  error_rate_pct: 0.5
`,
  },
  {
    id: 'stress-peak',
    name: 'Peak Stress Spike',
    description: 'High concurrency burst with stringent failure tolerances.',
    yaml: `version: "1.0"
execution:
  vus: 200
  duration: 120s
  ramp_up: 5s
thresholds:
  p90_latency_ms: 150
  p95_latency_ms: 300
  error_rate_pct: 0.1
`,
  },
]

export function formatYamlError(err: YamlErrorDetail): string {
  if (err.line !== undefined && err.column !== undefined) {
    return `Line ${err.line}, Col ${err.column}: ${err.message}`
  }
  if (err.line !== undefined) {
    return `Line ${err.line}: ${err.message}`
  }
  return err.message
}

export function validateYaml(content: string): YamlValidationResult {
  if (!content || !content.trim()) {
    return {
      isValid: false,
      errors: [{ message: 'Configuration YAML content cannot be empty' }],
    }
  }

  try {
    const doc = YAML.parseDocument(content)
    if (doc.errors && doc.errors.length > 0) {
      const parsedErrors: YamlErrorDetail[] = doc.errors.map((e) => {
        const pos = e.linePos ? e.linePos[0] : undefined
        return {
          message: e.message.split('\n')[0],
          line: pos ? pos.line : undefined,
          column: pos ? pos.col : undefined,
        }
      })
      return {
        isValid: false,
        errors: parsedErrors,
      }
    }

    const parsed = doc.toJSON()
    return {
      isValid: true,
      errors: [],
      parsed,
    }
  } catch (err: any) {
    return {
      isValid: false,
      errors: [
        {
          message: err?.message || 'Invalid YAML format',
        },
      ],
    }
  }
}

export function validateVuhiveSchema(parsed: any): SchemaValidationResult {
  const errors: string[] = []
  const warnings: string[] = []

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return {
      isValid: false,
      errors: ['vuhive.yaml root must be a YAML mapping/object'],
      warnings: [],
    }
  }

  if (!parsed.version) {
    warnings.push('Recommended "version" field is omitted (e.g. version: "1.0")')
  }

  if (!parsed.execution) {
    warnings.push('Missing recommended "execution" configuration block')
  } else if (typeof parsed.execution === 'object') {
    const exec = parsed.execution
    if (exec.vus !== undefined) {
      if (typeof exec.vus !== 'number' || exec.vus <= 0) {
        errors.push('"execution.vus" must be a positive integer')
      }
    }
    if (exec.duration !== undefined && typeof exec.duration !== 'string') {
      errors.push('"execution.duration" must be a string duration format (e.g. "60s", "5m")')
    }
    if (exec.ramp_up !== undefined && typeof exec.ramp_up !== 'string') {
      errors.push('"execution.ramp_up" must be a string duration format (e.g. "10s")')
    }
  }

  if (parsed.thresholds && typeof parsed.thresholds === 'object') {
    const th = parsed.thresholds
    if (th.p95_latency_ms !== undefined && (typeof th.p95_latency_ms !== 'number' || th.p95_latency_ms < 0)) {
      errors.push('"thresholds.p95_latency_ms" must be a non-negative number')
    }
    if (
      th.error_rate_pct !== undefined &&
      (typeof th.error_rate_pct !== 'number' || th.error_rate_pct < 0 || th.error_rate_pct > 100)
    ) {
      errors.push('"thresholds.error_rate_pct" must be a percentage between 0.0 and 100.0')
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
    warnings,
  }
}
