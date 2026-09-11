/**
 * Parses CPU quantity strings (e.g., "500m", "1", "2.5", "1000m") into millicores integer.
 */
export function parseCPUTomilli(cpuStr: string): number {
  if (!cpuStr) return 0
  const trimmed = cpuStr.trim()
  if (!trimmed) return 0

  if (trimmed.endsWith('m')) {
    const val = parseInt(trimmed.slice(0, -1), 10)
    return isNaN(val) ? 0 : Math.max(0, val)
  }

  const val = parseFloat(trimmed)
  if (isNaN(val)) return 0
  return Math.round(Math.max(0, val) * 1000)
}

/**
 * Formats millicores into a Kubernetes CPU quantity string.
 * When preferCores is true, integers or clean decimals are formatted as whole cores (e.g. 1000 -> "1", 2500 -> "2.5").
 */
export function formatMillitoCPU(milli: number, preferCores: boolean = false): string {
  if (milli <= 0) return '0m'
  if (preferCores) {
    const cores = milli / 1000
    // If it formats nicely (e.g. 1, 2, 0.5, 2.5)
    if (milli % 100 === 0) {
      return `${cores}`
    }
  }
  return `${milli}m`
}

/**
 * Parses Memory quantity strings (e.g., "512Mi", "1Gi", "2048M", "4G") into MiB integer.
 */
export function parseMemoryToMiB(memStr: string): number {
  if (!memStr) return 0
  const trimmed = memStr.trim()
  if (!trimmed) return 0

  const match = trimmed.match(/^(\d+(?:\.\d+)?)\s*([a-zA-Z]*)$/)
  if (!match) return 0

  const num = parseFloat(match[1])
  const unit = match[2]

  switch (unit) {
    case 'Ki':
      return Math.round(num / 1024)
    case 'Mi':
    case 'M':
      return Math.round(num)
    case 'Gi':
    case 'G':
      return Math.round(num * 1024)
    case 'Ti':
    case 'T':
      return Math.round(num * 1024 * 1024)
    case '':
      // Assumed bytes if purely numeric
      return Math.round(num / (1024 * 1024))
    default:
      return Math.round(num)
  }
}

/**
 * Formats MiB into human readable Kubernetes memory quantity string.
 */
export function formatMiBToMemory(mib: number): string {
  if (mib <= 0) return '0Mi'
  if (mib >= 1024 && mib % 512 === 0) {
    const gib = mib / 1024
    return `${gib}Gi`
  }
  return `${mib}Mi`
}

export interface ResourceValidationResult {
  isValid: boolean
  cpuWarning?: string
  memWarning?: string
  isGuaranteedQoS?: boolean
}

export interface ResourceConfig {
  cpuRequest?: string
  cpuLimit?: string
  memoryRequest?: string
  memoryLimit?: string
}

/**
 * Validates resource requirements against Kubernetes rules:
 * - CPU Request cannot exceed CPU Limit
 * - Memory Request cannot exceed Memory Limit
 * - Checks for Guaranteed QoS (Request == Limit for both CPU and Memory)
 */
export function validateResourceRequirements(config: ResourceConfig): ResourceValidationResult {
  const reqCpu = parseCPUTomilli(config.cpuRequest || '')
  const limCpu = parseCPUTomilli(config.cpuLimit || '')
  const reqMem = parseMemoryToMiB(config.memoryRequest || '')
  const limMem = parseMemoryToMiB(config.memoryLimit || '')

  let cpuWarning: string | undefined
  let memWarning: string | undefined

  if (reqCpu > 0 && limCpu > 0 && reqCpu > limCpu) {
    cpuWarning = `CPU request (${config.cpuRequest}) exceeds CPU limit (${config.cpuLimit})`
  }

  if (reqMem > 0 && limMem > 0 && reqMem > limMem) {
    memWarning = `Memory request (${config.memoryRequest}) exceeds Memory limit (${config.memoryLimit})`
  }

  const isGuaranteedQoS =
    reqCpu > 0 &&
    limCpu > 0 &&
    reqCpu === limCpu &&
    reqMem > 0 &&
    limMem > 0 &&
    reqMem === limMem

  return {
    isValid: !cpuWarning && !memWarning,
    cpuWarning,
    memWarning,
    isGuaranteedQoS,
  }
}
