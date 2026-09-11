/**
 * Utilities for client-side export of telemetry data to CSV and JSON files.
 */

export interface CsvColumn<T = any> {
  key: string
  header: string
  formatter?: (val: any, row: T) => any
}

/**
 * Escapes an individual cell value according to RFC 4180 CSV standard.
 */
export function escapeCsvCell(val: any): string {
  if (val === null || val === undefined) {
    return ''
  }

  if (typeof val === 'number' || typeof val === 'boolean') {
    return String(val)
  }

  let str: string
  if (typeof val === 'object') {
    try {
      str = JSON.stringify(val)
    } catch {
      str = String(val)
    }
  } else {
    str = String(val)
  }

  if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
    return `"${str.replace(/"/g, '""')}"`
  }

  return str
}

/**
 * Builds a RFC 4180 compliant CSV string from tabular column definitions and row objects.
 */
export function buildCsvString<T = any>(columns: CsvColumn<T>[], data: T[]): string {
  const headerLine = columns.map((col) => escapeCsvCell(col.header)).join(',')
  if (!data || data.length === 0) {
    return headerLine
  }

  const rowLines = data.map((row) =>
    columns
      .map((col) => {
        const rawVal = (row as any)?.[col.key]
        const formattedVal = col.formatter ? col.formatter(rawVal, row) : rawVal
        return escapeCsvCell(formattedVal)
      })
      .join(',')
  )

  return [headerLine, ...rowLines].join('\r\n')
}

/**
 * Triggers a browser file download using a Blob and ephemeral anchor element.
 */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.style.display = 'none'

  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)

  window.URL.revokeObjectURL(url)
}

/**
 * Formats data as pretty-printed JSON and triggers download.
 */
export function downloadJsonFile(data: any, filename: string): void {
  const jsonStr = typeof data === 'string' ? data : JSON.stringify(data, null, 2)
  const blob = new Blob([jsonStr], { type: 'application/json;charset=utf-8;' })
  downloadBlob(blob, filename)
}

/**
 * Formats tabular data as CSV and triggers download.
 */
export function exportToCsvFile<T = any>(
  columns: CsvColumn<T>[],
  data: T[],
  filename: string
): void {
  const csvStr = buildCsvString(columns, data)
  const blob = new Blob([csvStr], { type: 'text/csv;charset=utf-8;' })
  downloadBlob(blob, filename)
}
