import { describe, it, expect } from 'vitest'
import { formatErrorRate } from '@/lib/format-utils'

describe('formatErrorRate Utility', () => {
  it('formats 0% error rate correctly', () => {
    expect(formatErrorRate(0)).toBe('0.00%')
    expect(formatErrorRate(0.0)).toBe('0.00%')
  })

  it('formats 100% error rate correctly without inflation', () => {
    expect(formatErrorRate(100)).toBe('100.00%')
    expect(formatErrorRate(100.0)).toBe('100.00%')
    expect(formatErrorRate(100)).not.toBe('10000.00%')
  })

  it('formats partial percentages with precision', () => {
    expect(formatErrorRate(50)).toBe('50.00%')
    expect(formatErrorRate(12.345)).toBe('12.35%')
    expect(formatErrorRate(2.4)).toBe('2.40%')
    expect(formatErrorRate(8.2)).toBe('8.20%')
    expect(formatErrorRate(0.5)).toBe('0.50%')
  })

  it('supports custom decimal places', () => {
    expect(formatErrorRate(12.345, 1)).toBe('12.3%')
    expect(formatErrorRate(100, 1)).toBe('100.0%')
    expect(formatErrorRate(0, 0)).toBe('0%')
  })

  it('clamps out-of-bound percentages to [0.0, 100.0]', () => {
    expect(formatErrorRate(150)).toBe('100.00%')
    expect(formatErrorRate(10000)).toBe('100.00%')
    expect(formatErrorRate(-10)).toBe('0.00%')
  })

  it('returns "-" for null or undefined values', () => {
    expect(formatErrorRate(undefined)).toBe('-')
    expect(formatErrorRate(null)).toBe('-')
  })

  it('handles NaN or non-finite numbers safely', () => {
    expect(formatErrorRate(Number.NaN)).toBe('-')
    expect(formatErrorRate(Number.POSITIVE_INFINITY)).toBe('-')
  })
})
