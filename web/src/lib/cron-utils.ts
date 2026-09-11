export interface CronPreset {
  label: string
  expr: string
  desc: string
}

export const CRON_PRESETS: CronPreset[] = [
  {
    label: 'Hourly',
    expr: '0 * * * *',
    desc: 'At minute 0 of every hour',
  },
  {
    label: 'Nightly',
    expr: '0 2 * * *',
    desc: 'Every day at 02:00 UTC',
  },
  {
    label: 'Daily at midnight',
    expr: '0 0 * * *',
    desc: 'Every day at midnight (00:00 UTC)',
  },
  {
    label: 'Every Monday morning',
    expr: '0 9 * * 1',
    desc: 'Every Monday at 09:00 UTC',
  },
  {
    label: 'Weekend soak',
    expr: '0 4 * * 6',
    desc: 'Every Saturday at 04:00 UTC',
  },
]

interface FieldRule {
  min: number
  max: number
  name: string
}

const FIELD_RULES: FieldRule[] = [
  { min: 0, max: 59, name: 'minute' },
  { min: 0, max: 23, name: 'hour' },
  { min: 1, max: 31, name: 'day of month' },
  { min: 1, max: 12, name: 'month' },
  { min: 0, max: 7, name: 'day of week' },
]

function validateSubPart(part: string, rule: FieldRule): boolean {
  if (part === '*') return true

  // Handle step expressions like */15 or 1-5/2
  if (part.includes('/')) {
    const [base, stepStr] = part.split('/')
    if (!stepStr || !/^\d+$/.test(stepStr)) return false
    const step = parseInt(stepStr, 10)
    if (step <= 0 || step > rule.max) return false
    if (base === '*') return true
    return validateSubPart(base, rule)
  }

  // Handle range like 1-5
  if (part.includes('-')) {
    const [startStr, endStr] = part.split('-')
    if (!startStr || !endStr || !/^\d+$/.test(startStr) || !/^\d+$/.test(endStr)) return false
    const start = parseInt(startStr, 10)
    const end = parseInt(endStr, 10)
    return start >= rule.min && start <= rule.max && end >= rule.min && end <= rule.max && start <= end
  }

  // Single number
  if (!/^\d+$/.test(part)) return false
  const num = parseInt(part, 10)
  return num >= rule.min && num <= rule.max
}

function validateField(field: string, rule: FieldRule): boolean {
  if (!field) return false
  const subParts = field.split(',')
  for (const part of subParts) {
    if (!validateSubPart(part, rule)) return false
  }
  return true
}

export function validateCron(expr: string): { isValid: boolean; error?: string } {
  const trimmed = expr.trim()
  if (!trimmed) {
    return { isValid: false, error: 'Cron expression cannot be empty' }
  }

  const parts = trimmed.split(/\s+/)
  if (parts.length !== 5) {
    return {
      isValid: false,
      error: `Expected 5 fields (minute, hour, day-of-month, month, day-of-week), found ${parts.length}`,
    }
  }

  for (let i = 0; i < 5; i++) {
    const rule = FIELD_RULES[i]
    if (!validateField(parts[i], rule)) {
      return {
        isValid: false,
        error: `Invalid ${rule.name} field: "${parts[i]}" (allowed range: ${rule.min}-${rule.max})`,
      }
    }
  }

  return { isValid: true }
}

const DOW_NAMES: Record<number, string> = {
  0: 'Sunday',
  1: 'Monday',
  2: 'Tuesday',
  3: 'Wednesday',
  4: 'Thursday',
  5: 'Friday',
  6: 'Saturday',
  7: 'Sunday',
}

function padZero(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

export function describeCron(expr: string): string {
  const validation = validateCron(expr)
  if (!validation.isValid) {
    return 'Invalid cron expression'
  }

  const [min, hour, dom, mon, dow] = expr.trim().split(/\s+/)

  // Exact preset match
  const foundPreset = CRON_PRESETS.find((p) => p.expr === expr.trim())
  if (foundPreset) {
    if (foundPreset.expr === '0 * * * *') return 'Every hour at minute 0'
    return foundPreset.desc
  }

  // Every N minutes: */N * * * *
  if (min.startsWith('*/') && hour === '*' && dom === '*' && mon === '*' && dow === '*') {
    const step = min.replace('*/', '')
    return `Every ${step} minutes`
  }

  // Hourly at specific minute: M * * * *
  if (/^\d+$/.test(min) && hour === '*' && dom === '*' && mon === '*' && dow === '*') {
    return `Every hour at minute ${min}`
  }

  // Daily at specific time: M H * * *
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && dom === '*' && mon === '*' && dow === '*') {
    const m = parseInt(min, 10)
    const h = parseInt(hour, 10)
    if (h === 0 && m === 0) {
      return 'Every day at midnight (00:00 UTC)'
    }
    return `Every day at ${padZero(h)}:${padZero(m)} UTC`
  }

  // Weekly at specific time and day: M H * * D
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && dom === '*' && mon === '*' && /^\d+$/.test(dow)) {
    const m = parseInt(min, 10)
    const h = parseInt(hour, 10)
    const d = parseInt(dow, 10)
    const dayName = DOW_NAMES[d] || 'day'
    return `Every ${dayName} at ${padZero(h)}:${padZero(m)} UTC`
  }

  // Monthly on specific day: M H DOM * *
  if (/^\d+$/.test(min) && /^\d+$/.test(hour) && /^\d+$/.test(dom) && mon === '*' && dow === '*') {
    const m = parseInt(min, 10)
    const h = parseInt(hour, 10)
    const d = parseInt(dom, 10)
    const suffix = d === 1 ? '1st' : d === 2 ? '2nd' : d === 3 ? '3rd' : `${d}th`
    return `On the ${suffix} day of every month at ${padZero(h)}:${padZero(m)} UTC`
  }

  return `Runs on schedule: ${expr} (UTC)`
}

export function getNextCronRun(expr: string, fromDate: Date = new Date()): Date | null {
  const validation = validateCron(expr)
  if (!validation.isValid) return null

  const [minStr, hourStr, domStr, monStr, dowStr] = expr.trim().split(/\s+/)

  // Parse candidate values for each field
  const parseFieldCandidates = (field: string, rule: FieldRule): number[] => {
    const set = new Set<number>()
    const subParts = field.split(',')
    for (const part of subParts) {
      if (part === '*') {
        for (let i = rule.min; i <= rule.max; i++) set.add(i)
      } else if (part.startsWith('*/')) {
        const step = parseInt(part.replace('*/', ''), 10)
        for (let i = rule.min; i <= rule.max; i += step) set.add(i)
      } else if (part.includes('-')) {
        const [start, end] = part.split('-').map((v) => parseInt(v, 10))
        for (let i = start; i <= end; i++) set.add(i)
      } else if (/^\d+$/.test(part)) {
        set.add(parseInt(part, 10))
      }
    }
    return Array.from(set).sort((a, b) => a - b)
  }

  const validMinutes = parseFieldCandidates(minStr, FIELD_RULES[0])
  const validHours = parseFieldCandidates(hourStr, FIELD_RULES[1])
  const validDoms = parseFieldCandidates(domStr, FIELD_RULES[2])
  const validMonths = parseFieldCandidates(monStr, FIELD_RULES[3])
  const validDows = parseFieldCandidates(dowStr, FIELD_RULES[4])
  // Normalize dow 7 to 0 (Sunday)
  const normalizedDows = new Set(validDows.map((d) => (d === 7 ? 0 : d)))

  // Look up to 366 days ahead, minute-by-minute or skipping smart intervals
  const cursor = new Date(fromDate.getTime())
  // Start from the next whole minute
  cursor.setUTCSeconds(0, 0)
  cursor.setUTCMinutes(cursor.getUTCMinutes() + 1)

  const maxLookaheadMs = 366 * 24 * 60 * 60 * 1000
  const endMs = fromDate.getTime() + maxLookaheadMs

  while (cursor.getTime() <= endMs) {
    const month = cursor.getUTCMonth() + 1
    if (!validMonths.includes(month)) {
      // Jump to next month
      cursor.setUTCMonth(cursor.getUTCMonth() + 1, 1)
      cursor.setUTCHours(0, 0, 0, 0)
      continue
    }

    const dom = cursor.getUTCDate()
    const dow = cursor.getUTCDay()

    const domMatches = domStr === '*' || validDoms.includes(dom)
    const dowMatches = dowStr === '*' || normalizedDows.has(dow)

    // Standard cron rule: if either dom or dow is restricted (not *),
    // and both are specified, either matching or both matching depending on spec.
    // If one is '*', other governs. If both not '*', standard is OR.
    const dayMatches =
      domStr === '*' && dowStr === '*'
        ? true
        : domStr === '*'
        ? dowMatches
        : dowStr === '*'
        ? domMatches
        : domMatches || dowMatches

    if (!dayMatches) {
      // Advance to next day
      cursor.setUTCDate(cursor.getUTCDate() + 1)
      cursor.setUTCHours(0, 0, 0, 0)
      continue
    }

    const hour = cursor.getUTCHours()
    if (!validHours.includes(hour)) {
      // Advance to next hour
      cursor.setUTCHours(cursor.getUTCHours() + 1, 0, 0, 0)
      continue
    }

    const minute = cursor.getUTCMinutes()
    if (!validMinutes.includes(minute)) {
      cursor.setUTCMinutes(cursor.getUTCMinutes() + 1, 0, 0)
      continue
    }

    // Found match
    return new Date(cursor.getTime())
  }

  return null
}
