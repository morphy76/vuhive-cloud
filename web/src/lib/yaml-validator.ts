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
    yaml: `# yaml-language-server: $schema=https://raw.githubusercontent.com/morphy76/vuhive/main/schemas/vuhive.schema.json
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
        target: "250ms"
      - metric: vuhive.http.req_failed
        stat: rate
        operator: "<="
        target: "0.01"
`,
  },
  {
    id: 'staged-progression',
    name: 'Staged Step Progression',
    description: 'Multi-stage ramp up transitioning through load plateaus.',
    yaml: `# yaml-language-server: $schema=https://raw.githubusercontent.com/morphy76/vuhive/main/schemas/vuhive.schema.json
version: "1.0"
default_scenario: staged_progression

scenarios:
  staged_progression:
    type: ramping_vus
    stages:
      - target: 10
        duration: 30s
      - target: 50
        duration: 60s
      - target: 0
        duration: 30s
    ramp_down: 5s
    vu_timeout: 5s
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p95
        operator: "<"
        target: "300ms"
      - metric: vuhive.http.req_duration
        stat: p99
        operator: "<"
        target: "500ms"
      - metric: vuhive.http.req_failed
        stat: rate
        operator: "<="
        target: "0.005"
`,
  },
  {
    id: 'stress-peak',
    name: 'Peak Stress Spike',
    description: 'High concurrency burst with stringent failure tolerances.',
    yaml: `# yaml-language-server: $schema=https://raw.githubusercontent.com/morphy76/vuhive/main/schemas/vuhive.schema.json
version: "1.0"
default_scenario: stress_peak

scenarios:
  stress_peak:
    type: constant_vus
    vus: 200
    ramp_up: 5s
    run_period: 120s
    ramp_down: 10s
    vu_timeout: 5s
    thresholds:
      - metric: vuhive.http.req_duration
        stat: p90
        operator: "<"
        target: "150ms"
      - metric: vuhive.http.req_duration
        stat: p95
        operator: "<"
        target: "300ms"
      - metric: vuhive.http.req_failed
        stat: rate
        operator: "<="
        target: "0.001"
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

const GO_DURATION_REGEX = /^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$/
const VALID_SCENARIO_TYPES = ['constant_vus', 'arrival_rate', 'ramping_vus'] as const
const VALID_THRESHOLD_STATS = [
  'p50',
  'p90',
  'p95',
  'p99',
  'mean',
  'max',
  'count',
  'rate',
  'value',
] as const
const VALID_THRESHOLD_OPERATORS = ['<', '<=', '>', '>='] as const
const VALID_ON_NO_DATA = ['zero', 'fail', 'pass', 'ignore', 'skip'] as const
const VALID_THINK_TIME_TYPES = ['fixed', 'range', 'expo', 'gaussian'] as const
const VALID_STRICT_MODES = ['off', 'warn', 'fatal'] as const

function isValidDuration(val: unknown): boolean {
  return typeof val === 'string' && GO_DURATION_REGEX.test(val.trim())
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

  // Detect deprecated/legacy format
  if (parsed.execution !== undefined) {
    errors.push(
      'Detected legacy root "execution" block. In vuhive SDK v1.1.5, scenarios must be declared under the "scenarios" mapping (e.g. scenarios.<name>.type: constant_vus).'
    )
  }

  // Version validation
  if (!parsed.version) {
    errors.push('Missing required "version" field (must be "1.0")')
  } else if (parsed.version !== '1.0') {
    errors.push(`Invalid version "${parsed.version}". vuhive configuration version must be "1.0"`)
  }

  // Scenarios validation
  if (!parsed.scenarios || typeof parsed.scenarios !== 'object' || Array.isArray(parsed.scenarios)) {
    errors.push('Missing required "scenarios" configuration block')
    return { isValid: false, errors, warnings }
  }

  const scenarioNames = Object.keys(parsed.scenarios)
  if (scenarioNames.length === 0) {
    errors.push('"scenarios" must define at least one named scenario')
    return { isValid: false, errors, warnings }
  }

  // Default scenario warning
  if (parsed.default_scenario) {
    if (typeof parsed.default_scenario !== 'string') {
      errors.push('"default_scenario" must be a string')
    } else if (!scenarioNames.includes(parsed.default_scenario)) {
      warnings.push(
        `"default_scenario: ${parsed.default_scenario}" does not match any declared scenario in "scenarios" (${scenarioNames.join(', ')})`
      )
    }
  }

  for (const name of scenarioNames) {
    const scenario = parsed.scenarios[name]
    if (!scenario || typeof scenario !== 'object' || Array.isArray(scenario)) {
      errors.push(`Scenario "${name}" must be a configuration object`)
      continue
    }

    // Type validation
    if (!scenario.type) {
      errors.push(
        `Scenario "${name}" requires a "type" field ("constant_vus", "arrival_rate", or "ramping_vus")`
      )
      continue
    }

    if (!VALID_SCENARIO_TYPES.includes(scenario.type)) {
      errors.push(
        `Scenario "${name}" has invalid type "${scenario.type}". Expected "constant_vus", "arrival_rate", or "ramping_vus"`
      )
      continue
    }

    // Common duration fields
    const optionalDurations: (keyof typeof scenario)[] = [
      'ramp_up',
      'ramp_down',
      'drain',
      'drain_period',
      'vu_timeout',
      'startup_grace_period',
      'watchdog_stall_threshold',
      'watchdog_interval',
    ]
    for (const field of optionalDurations) {
      if (scenario[field] !== undefined && !isValidDuration(scenario[field])) {
        errors.push(
          `Scenario "${name}.${String(field)}" must be a valid Go duration string (e.g. "10s", "500ms")`
        )
      }
    }

    if (
      scenario.max_pretest_retries !== undefined &&
      (typeof scenario.max_pretest_retries !== 'number' ||
        scenario.max_pretest_retries < 0 ||
        !Number.isInteger(scenario.max_pretest_retries))
    ) {
      errors.push(`Scenario "${name}.max_pretest_retries" must be a non-negative integer`)
    }

    if (
      scenario.min_ready_ratio !== undefined &&
      (typeof scenario.min_ready_ratio !== 'number' ||
        scenario.min_ready_ratio < 0 ||
        scenario.min_ready_ratio > 1.0)
    ) {
      errors.push(`Scenario "${name}.min_ready_ratio" must be a float between 0.0 and 1.0`)
    }

    if (
      scenario.strict !== undefined &&
      !VALID_STRICT_MODES.includes(scenario.strict)
    ) {
      errors.push(
        `Scenario "${name}.strict" must be one of "off", "warn", or "fatal"`
      )
    }

    // Type-specific requirements
    if (scenario.type === 'constant_vus') {
      if (scenario.vus === undefined) {
        errors.push(`Scenario "${name}" (constant_vus) requires "vus"`)
      } else if (
        typeof scenario.vus !== 'number' ||
        scenario.vus <= 0 ||
        !Number.isInteger(scenario.vus)
      ) {
        errors.push(`Scenario "${name}.vus" must be a positive integer`)
      }

      if (scenario.run_period === undefined) {
        errors.push(`Scenario "${name}" (constant_vus) requires "run_period"`)
      } else if (!isValidDuration(scenario.run_period)) {
        errors.push(
          `Scenario "${name}.run_period" must be a valid Go duration string (e.g. "60s", "2m")`
        )
      }
    } else if (scenario.type === 'arrival_rate') {
      if (scenario.target_tps === undefined) {
        errors.push(`Scenario "${name}" (arrival_rate) requires "target_tps"`)
      } else if (
        typeof scenario.target_tps !== 'number' ||
        scenario.target_tps <= 0 ||
        !Number.isInteger(scenario.target_tps)
      ) {
        errors.push(`Scenario "${name}.target_tps" must be a positive integer`)
      }

      if (scenario.max_vus === undefined) {
        errors.push(`Scenario "${name}" (arrival_rate) requires "max_vus"`)
      } else if (
        typeof scenario.max_vus !== 'number' ||
        scenario.max_vus <= 0 ||
        !Number.isInteger(scenario.max_vus)
      ) {
        errors.push(`Scenario "${name}.max_vus" must be a positive integer`)
      }

      if (scenario.run_period === undefined) {
        errors.push(`Scenario "${name}" (arrival_rate) requires "run_period"`)
      } else if (!isValidDuration(scenario.run_period)) {
        errors.push(
          `Scenario "${name}.run_period" must be a valid Go duration string (e.g. "60s", "2m")`
        )
      }

      if (
        scenario.burst_buffer !== undefined &&
        (typeof scenario.burst_buffer !== 'number' ||
          scenario.burst_buffer < 0 ||
          !Number.isInteger(scenario.burst_buffer))
      ) {
        errors.push(`Scenario "${name}.burst_buffer" must be a non-negative integer`)
      }
    } else if (scenario.type === 'ramping_vus') {
      if (!Array.isArray(scenario.stages) || scenario.stages.length === 0) {
        errors.push(`Scenario "${name}" (ramping_vus) requires a non-empty "stages" array`)
      } else {
        scenario.stages.forEach((stage: any, idx: number) => {
          if (!stage || typeof stage !== 'object' || Array.isArray(stage)) {
            errors.push(`Scenario "${name}.stages[${idx}]" must be an object`)
            return
          }
          if (stage.target === undefined) {
            errors.push(`Scenario "${name}.stages[${idx}]" requires "target"`)
          } else if (
            typeof stage.target !== 'number' ||
            stage.target < 0 ||
            !Number.isInteger(stage.target)
          ) {
            errors.push(`Scenario "${name}.stages[${idx}].target" must be a non-negative integer`)
          }
          if (stage.duration === undefined) {
            errors.push(`Scenario "${name}.stages[${idx}]" requires "duration"`)
          } else if (!isValidDuration(stage.duration)) {
            errors.push(
              `Scenario "${name}.stages[${idx}].duration" must be a valid Go duration string (e.g. "30s", "1m")`
            )
          }
        })
      }
    }

    // Thresholds validation
    if (scenario.thresholds !== undefined) {
      if (!Array.isArray(scenario.thresholds)) {
        errors.push(`Scenario "${name}.thresholds" must be an array`)
      } else {
        scenario.thresholds.forEach((th: any, idx: number) => {
          if (!th || typeof th !== 'object' || Array.isArray(th)) {
            errors.push(`Scenario "${name}.thresholds[${idx}]" must be an object`)
            return
          }

          if (!th.metric || typeof th.metric !== 'string' || !th.metric.trim()) {
            errors.push(`Scenario "${name}.thresholds[${idx}]" requires a non-empty "metric" string`)
          }

          if (!th.stat || !VALID_THRESHOLD_STATS.includes(th.stat)) {
            errors.push(
              `Scenario "${name}.thresholds[${idx}].stat" must be one of: ${VALID_THRESHOLD_STATS.join(', ')}`
            )
          }

          if (!th.operator || !VALID_THRESHOLD_OPERATORS.includes(th.operator)) {
            errors.push(
              `Scenario "${name}.thresholds[${idx}].operator" must be one of: ${VALID_THRESHOLD_OPERATORS.join(', ')}`
            )
          }

          if (
            th.target === undefined ||
            th.target === null ||
            (typeof th.target === 'string' && !th.target.trim())
          ) {
            errors.push(`Scenario "${name}.thresholds[${idx}]" requires a non-empty "target" value`)
          }

          if (th.on_no_data !== undefined && !VALID_ON_NO_DATA.includes(th.on_no_data)) {
            errors.push(
              `Scenario "${name}.thresholds[${idx}].on_no_data" must be one of: ${VALID_ON_NO_DATA.join(', ')}`
            )
          }

          if (th.abort_on_fail !== undefined && typeof th.abort_on_fail !== 'boolean') {
            errors.push(`Scenario "${name}.thresholds[${idx}].abort_on_fail" must be a boolean`)
          }

          if (th.delay_abort_eval !== undefined && !isValidDuration(th.delay_abort_eval)) {
            errors.push(
              `Scenario "${name}.thresholds[${idx}].delay_abort_eval" must be a valid Go duration string`
            )
          }
        })
      }
    }

    // Think time / interaction delay validation
    const thinkTimeFields = ['think_time', 'interaction_delay'] as const
    for (const ttField of thinkTimeFields) {
      if (scenario[ttField] !== undefined) {
        const tt = scenario[ttField]
        if (!tt || typeof tt !== 'object' || Array.isArray(tt)) {
          errors.push(`Scenario "${name}.${ttField}" must be an object`)
          continue
        }
        if (!tt.type || !VALID_THINK_TIME_TYPES.includes(tt.type)) {
          errors.push(
            `Scenario "${name}.${ttField}.type" must be one of: ${VALID_THINK_TIME_TYPES.join(', ')}`
          )
        }
        if (tt.duration !== undefined && !isValidDuration(tt.duration)) {
          errors.push(`Scenario "${name}.${ttField}.duration" must be a valid Go duration string`)
        }
        if (tt.min !== undefined && !isValidDuration(tt.min)) {
          errors.push(`Scenario "${name}.${ttField}.min" must be a valid Go duration string`)
        }
        if (tt.max !== undefined && !isValidDuration(tt.max)) {
          errors.push(`Scenario "${name}.${ttField}.max" must be a valid Go duration string`)
        }
        if (tt.mean !== undefined && !isValidDuration(tt.mean)) {
          errors.push(`Scenario "${name}.${ttField}.mean" must be a valid Go duration string`)
        }
        if (tt.std_dev !== undefined && !isValidDuration(tt.std_dev)) {
          errors.push(`Scenario "${name}.${ttField}.std_dev" must be a valid Go duration string`)
        }
      }
    }
  }

  return {
    isValid: errors.length === 0,
    errors,
    warnings,
  }
}
