/**
 * Formats an error rate percentage value safely.
 *
 * The canonical contract across vuhive-cloud is that error_rate_pct is already
 * a percentage value in the range [0.0, 100.0] (e.g. 0.5 represents 0.5%, 100.0 represents 100%).
 *
 * This function guarantees:
 * - Returns '-' for undefined, null, NaN, or non-finite numbers.
 * - Clamps out-of-bounds numbers to [0.0, 100.0] (preventing inflated values like 10000.00%).
 * - Formats output with the specified decimal places (default: 2) and appends '%'.
 */
export function formatErrorRate(ratePct?: number | null, decimals = 2): string {
  if (ratePct === undefined || ratePct === null) {
    return '-'
  }

  const num = typeof ratePct === 'number' ? ratePct : Number(ratePct)
  if (!Number.isFinite(num)) {
    return '-'
  }

  const clamped = Math.min(100.0, Math.max(0.0, num))
  return `${clamped.toFixed(decimals)}%`
}
