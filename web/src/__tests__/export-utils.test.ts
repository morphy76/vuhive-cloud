import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  escapeCsvCell,
  buildCsvString,
  downloadBlob,
  downloadJsonFile,
  exportToCsvFile,
} from '@/lib/export-utils'

describe('export-utils', () => {
  describe('escapeCsvCell', () => {
    it('returns empty string for null and undefined', () => {
      expect(escapeCsvCell(null)).toBe('')
      expect(escapeCsvCell(undefined)).toBe('')
    })

    it('formats numbers and booleans directly', () => {
      expect(escapeCsvCell(42)).toBe('42')
      expect(escapeCsvCell(3.1415)).toBe('3.1415')
      expect(escapeCsvCell(true)).toBe('true')
      expect(escapeCsvCell(false)).toBe('false')
    })

    it('leaves simple strings unquoted if no special characters', () => {
      expect(escapeCsvCell('hello_world')).toBe('hello_world')
      expect(escapeCsvCell('Step 1')).toBe('Step 1')
    })

    it('quotes strings containing commas', () => {
      expect(escapeCsvCell('GET /api/v1,v2')).toBe('"GET /api/v1,v2"')
    })

    it('escapes quotes and wraps in quotes when quotes present', () => {
      expect(escapeCsvCell('He said "hello"')).toBe('"He said ""hello"""')
    })

    it('quotes strings containing newlines', () => {
      expect(escapeCsvCell('Line 1\nLine 2')).toBe('"Line 1\nLine 2"')
    })

    it('stringifies objects as JSON and escapes', () => {
      expect(escapeCsvCell({ a: 1 })).toBe('"{""a"":1}"')
    })
  })

  describe('buildCsvString', () => {
    const columns = [
      { key: 'name', header: 'Step Name' },
      { key: 'requests', header: 'Requests' },
      { key: 'tps', header: 'TPS' },
      { key: 'errorRate', header: 'Error Rate %', formatter: (val: number) => `${val.toFixed(2)}%` },
    ]

    const data = [
      { name: 'GET /items', requests: 1000, tps: 50.5, errorRate: 0.05 },
      { name: 'POST /items,new', requests: 500, tps: 25.25, errorRate: 0.0 },
    ]

    it('generates standard CSV with header and data rows', () => {
      const csv = buildCsvString(columns, data)
      const lines = csv.split('\r\n')

      expect(lines[0]).toBe('Step Name,Requests,TPS,Error Rate %')
      expect(lines[1]).toBe('GET /items,1000,50.5,0.05%')
      expect(lines[2]).toBe('"POST /items,new",500,25.25,0.00%')
    })

    it('handles empty dataset returning only header row', () => {
      const csv = buildCsvString(columns, [])
      expect(csv).toBe('Step Name,Requests,TPS,Error Rate %')
    })
  })

  describe('downloadBlob and file exporters', () => {
    let createObjectURLSpy: any
    let revokeObjectURLSpy: any
    let clickSpy: any

    beforeEach(() => {
      createObjectURLSpy = vi.fn().mockReturnValue('blob:mock-url')
      revokeObjectURLSpy = vi.fn()
      window.URL.createObjectURL = createObjectURLSpy
      window.URL.revokeObjectURL = revokeObjectURLSpy

      clickSpy = vi.fn()
      vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(clickSpy)
    })

    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('downloadBlob creates anchor, clicks it, and cleans up URL', () => {
      const blob = new Blob(['test content'], { type: 'text/plain' })
      downloadBlob(blob, 'test.txt')

      expect(createObjectURLSpy).toHaveBeenCalledWith(blob)
      expect(clickSpy).toHaveBeenCalled()
      expect(revokeObjectURLSpy).toHaveBeenCalledWith('blob:mock-url')
    })

    it('downloadJsonFile serializes object and triggers download', () => {
      const payload = { suite: 'ecommerce', passed: true }
      downloadJsonFile(payload, 'report-123.json')

      expect(createObjectURLSpy).toHaveBeenCalled()
      const blobArg = createObjectURLSpy.mock.calls[0][0] as Blob
      expect(blobArg.type).toContain('application/json')
      expect(clickSpy).toHaveBeenCalled()
    })

    it('exportToCsvFile builds CSV and triggers download', () => {
      const columns = [{ key: 'id', header: 'ID' }, { key: 'status', header: 'Status' }]
      const rows = [{ id: '1', status: 'PASS' }]

      exportToCsvFile(columns, rows, 'summary-steps.csv')

      expect(createObjectURLSpy).toHaveBeenCalled()
      const blobArg = createObjectURLSpy.mock.calls[0][0] as Blob
      expect(blobArg.type).toContain('text/csv')
      expect(clickSpy).toHaveBeenCalled()
    })
  })
})
