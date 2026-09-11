import { describe, it, expect } from 'vitest'
import {
  parseCPUTomilli,
  formatMillitoCPU,
  parseMemoryToMiB,
  formatMiBToMemory,
  validateResourceRequirements,
} from '../lib/k8s-resources'

describe('Kubernetes Resource Parsing & Conversions', () => {
  describe('CPU parsing and formatting', () => {
    it('parses millicores and plain cores correctly', () => {
      expect(parseCPUTomilli('500m')).toBe(500)
      expect(parseCPUTomilli('1000m')).toBe(1000)
      expect(parseCPUTomilli('1')).toBe(1000)
      expect(parseCPUTomilli('2.5')).toBe(2500)
      expect(parseCPUTomilli('0.25')).toBe(250)
      expect(parseCPUTomilli('')).toBe(0)
    })

    it('formats millicores to appropriate CPU representation', () => {
      expect(formatMillitoCPU(500)).toBe('500m')
      expect(formatMillitoCPU(1000)).toBe('1000m')
      expect(formatMillitoCPU(2000, true)).toBe('2')
      expect(formatMillitoCPU(2500, true)).toBe('2.5')
      expect(formatMillitoCPU(2500, false)).toBe('2500m')
    })
  })

  describe('Memory parsing and formatting', () => {
    it('parses Mi, Gi, and plain numbers correctly', () => {
      expect(parseMemoryToMiB('512Mi')).toBe(512)
      expect(parseMemoryToMiB('1Gi')).toBe(1024)
      expect(parseMemoryToMiB('2Gi')).toBe(2048)
      expect(parseMemoryToMiB('4Gi')).toBe(4096)
      expect(parseMemoryToMiB('1024M')).toBe(1024)
      expect(parseMemoryToMiB('1G')).toBe(1024)
      expect(parseMemoryToMiB('')).toBe(0)
    })

    it('formats MiB to human readable memory units', () => {
      expect(formatMiBToMemory(512)).toBe('512Mi')
      expect(formatMiBToMemory(1024)).toBe('1Gi')
      expect(formatMiBToMemory(2048)).toBe('2Gi')
      expect(formatMiBToMemory(1536)).toBe('1.5Gi')
      expect(formatMiBToMemory(768)).toBe('768Mi')
    })
  })

  describe('Resource Requirements Validation', () => {
    it('validates matching and valid requests and limits', () => {
      const result = validateResourceRequirements({
        cpuRequest: '500m',
        cpuLimit: '1000m',
        memoryRequest: '512Mi',
        memoryLimit: '1Gi',
      })

      expect(result.isValid).toBe(true)
      expect(result.cpuWarning).toBeUndefined()
      expect(result.memWarning).toBeUndefined()
    })

    it('flags warning when CPU request exceeds CPU limit', () => {
      const result = validateResourceRequirements({
        cpuRequest: '2000m',
        cpuLimit: '1000m',
        memoryRequest: '512Mi',
        memoryLimit: '1Gi',
      })

      expect(result.isValid).toBe(false)
      expect(result.cpuWarning).toBe('CPU request (2000m) exceeds CPU limit (1000m)')
    })

    it('flags warning when Memory request exceeds Memory limit', () => {
      const result = validateResourceRequirements({
        cpuRequest: '500m',
        cpuLimit: '1000m',
        memoryRequest: '2Gi',
        memoryLimit: '1Gi',
      })

      expect(result.isValid).toBe(false)
      expect(result.memWarning).toBe('Memory request (2Gi) exceeds Memory limit (1Gi)')
    })

    it('validates equal request and limit (Guaranteed QoS)', () => {
      const result = validateResourceRequirements({
        cpuRequest: '2000m',
        cpuLimit: '2000m',
        memoryRequest: '4Gi',
        memoryLimit: '4Gi',
      })

      expect(result.isValid).toBe(true)
      expect(result.isGuaranteedQoS).toBe(true)
    })
  })
})
