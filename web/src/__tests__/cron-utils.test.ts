import { describe, it, expect } from 'vitest'
import {
  validateCron,
  describeCron,
  getNextCronRun,
  CRON_PRESETS,
} from '@/lib/cron-utils'

describe('Cron Utilities', () => {
  describe('validateCron', () => {
    it('accepts standard 5-field valid expressions', () => {
      expect(validateCron('0 * * * *').isValid).toBe(true)
      expect(validateCron('*/15 * * * *').isValid).toBe(true)
      expect(validateCron('0 2 * * *').isValid).toBe(true)
      expect(validateCron('0 0 1 * *').isValid).toBe(true)
      expect(validateCron('0 4 * * 6').isValid).toBe(true)
      expect(validateCron('30 9-17 * * 1-5').isValid).toBe(true)
      expect(validateCron('0,15,30,45 * * * *').isValid).toBe(true)
    })

    it('rejects invalid number of fields', () => {
      expect(validateCron('').isValid).toBe(false)
      expect(validateCron('* * * *').isValid).toBe(false) // 4 parts
      expect(validateCron('* * * * * *').isValid).toBe(false) // 6 parts
    })

    it('rejects out of range values', () => {
      expect(validateCron('60 * * * *').isValid).toBe(false) // minute > 59
      expect(validateCron('* 24 * * *').isValid).toBe(false) // hour > 23
      expect(validateCron('* * 32 * *').isValid).toBe(false) // day > 31
      expect(validateCron('* * 0 * *').isValid).toBe(false) // day < 1
      expect(validateCron('* * * 13 *').isValid).toBe(false) // month > 12
      expect(validateCron('* * * 0 *').isValid).toBe(false) // month < 1
      expect(validateCron('* * * * 8').isValid).toBe(false) // dow > 7
    })

    it('rejects invalid syntax characters', () => {
      expect(validateCron('invalid-cron').isValid).toBe(false)
      expect(validateCron('a b c d e').isValid).toBe(false)
      expect(validateCron('*/0 * * * *').isValid).toBe(false)
    })
  })

  describe('describeCron', () => {
    it('accurately describes standard presets in natural language', () => {
      expect(describeCron('0 * * * *')).toBe('Every hour at minute 0')
      expect(describeCron('*/15 * * * *')).toBe('Every 15 minutes')
      expect(describeCron('0 0 * * *')).toBe('Every day at midnight (00:00 UTC)')
      expect(describeCron('0 2 * * *')).toBe('Every day at 02:00 UTC')
      expect(describeCron('0 9 * * 1')).toBe('Every Monday at 09:00 UTC')
      expect(describeCron('0 4 * * 6')).toBe('Every Saturday at 04:00 UTC')
      expect(describeCron('0 0 1 * *')).toBe('On the 1st day of every month at 00:00 UTC')
    })

    it('handles custom hours and minutes', () => {
      expect(describeCron('30 14 * * *')).toBe('Every day at 14:30 UTC')
    })

    it('returns fallback error string for invalid cron', () => {
      expect(describeCron('invalid')).toBe('Invalid cron expression')
    })
  })

  describe('getNextCronRun', () => {
    it('calculates the next upcoming run in UTC', () => {
      const fixedBase = new Date('2026-09-11T12:00:00Z') // Friday 12:00 UTC
      const nextRun = getNextCronRun('0 14 * * *', fixedBase) // Every day at 14:00
      expect(nextRun).not.toBeNull()
      expect(nextRun?.toISOString()).toBe('2026-09-11T14:00:00.000Z')
    })

    it('rolls over to next day if scheduled time has passed today', () => {
      const fixedBase = new Date('2026-09-11T16:00:00Z') // Friday 16:00 UTC
      const nextRun = getNextCronRun('0 2 * * *', fixedBase) // Daily 02:00
      expect(nextRun).not.toBeNull()
      expect(nextRun?.toISOString()).toBe('2026-09-12T02:00:00.000Z')
    })

    it('rolls over to specific weekday in future', () => {
      const fixedBase = new Date('2026-09-11T12:00:00Z') // Friday
      const nextRun = getNextCronRun('0 9 * * 1', fixedBase) // Next Monday at 09:00
      expect(nextRun).not.toBeNull()
      expect(nextRun?.getUTCDay()).toBe(1) // Monday
      expect(nextRun?.toISOString()).toBe('2026-09-14T09:00:00.000Z')
    })

    it('returns null for invalid cron', () => {
      expect(getNextCronRun('invalid')).toBeNull()
    })
  })

  describe('CRON_PRESETS', () => {
    it('contains all required standard presets', () => {
      const labels = CRON_PRESETS.map((p) => p.label)
      expect(labels).toContain('Hourly')
      expect(labels).toContain('Nightly')
      expect(labels).toContain('Daily at midnight')
      expect(labels).toContain('Every Monday morning')
      expect(labels).toContain('Weekend soak')

      // Ensure each preset has valid cron expression
      for (const preset of CRON_PRESETS) {
        expect(validateCron(preset.expr).isValid).toBe(true)
      }
    })
  })
})
