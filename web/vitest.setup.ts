import '@testing-library/jest-dom'
import { configure } from '@testing-library/react'
import { expect } from 'vitest'
import * as matchers from 'vitest-axe/matchers'

// Set global Testing Library timeout to 10s to prevent flakiness in slow CI environments
configure({ asyncUtilTimeout: 10000 })

// Register vitest-axe matchers for toHaveNoViolations
expect.extend(matchers)

// Mock window.matchMedia for jsdom environment
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

// Mock ResizeObserver for Radix UI primitives and Recharts in jsdom
global.ResizeObserver = class ResizeObserver {
  callback: (entries: any[]) => void
  constructor(callback: any) {
    this.callback = callback
  }
  observe(target: any) {
    if (this.callback) {
      this.callback([{ target, contentRect: { width: 800, height: 400 } }])
    }
  }
  unobserve() {}
  disconnect() {}
}

// Mock SVGElement.prototype.getBBox for Recharts in jsdom
if (typeof SVGElement !== 'undefined' && !(SVGElement.prototype as any).getBBox) {
  ;(SVGElement.prototype as any).getBBox = () =>
    ({
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      bottom: 0,
      left: 0,
      right: 0,
      top: 0,
      toJSON: () => {},
    }) as any
}

// Mock Range.prototype.getClientRects for CodeMirror in jsdom
if (typeof Range !== 'undefined') {
  Range.prototype.getClientRects = () =>
    [
      {
        bottom: 0,
        height: 0,
        left: 0,
        right: 0,
        top: 0,
        width: 0,
        x: 0,
        y: 0,
        toJSON: () => {},
      },
    ] as any
  Range.prototype.getBoundingClientRect = () =>
    ({
      bottom: 0,
      height: 0,
      left: 0,
      right: 0,
      top: 0,
      width: 0,
      x: 0,
      y: 0,
      toJSON: () => {},
    }) as any
}
