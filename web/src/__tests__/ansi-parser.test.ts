import { describe, it, expect } from 'vitest'
import { parseAnsi, stripAnsi } from '../lib/ansi'

describe('ANSI Parser', () => {
  describe('stripAnsi', () => {
    it('returns unmodified string if no ANSI sequences are present', () => {
      expect(stripAnsi('hello world')).toBe('hello world')
    })

    it('strips standard color and style escape sequences', () => {
      const input = '\u001b[31mRed text\u001b[0m and \u001b[1mBold text\u001b[0m'
      expect(stripAnsi(input)).toBe('Red text and Bold text')
    })

    it('strips 256-color and 24-bit RGB sequences', () => {
      const input = '\u001b[38;5;208mOrange\u001b[0m and \u001b[38;2;255;100;50mCustom\u001b[0m'
      expect(stripAnsi(input)).toBe('Orange and Custom')
    })

    it('strips non-SGR control codes and cursor sequences', () => {
      const input = '\u001b[2K\u001b[1G\u001b[?25hClear and move'
      expect(stripAnsi(input)).toBe('Clear and move')
    })
  })

  describe('parseAnsi', () => {
    it('parses plain text without styles', () => {
      const segments = parseAnsi('Plain log message')
      expect(segments).toEqual([
        { text: 'Plain log message' },
      ])
    })

    it('parses basic foreground colors', () => {
      const input = '\u001b[31mError occurred\u001b[0m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(1)
      expect(segments[0].text).toBe('Error occurred')
      expect(segments[0].style?.color).toBeDefined()
    })

    it('parses standard background colors', () => {
      const input = '\u001b[42mSuccess background\u001b[0m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(1)
      expect(segments[0].text).toBe('Success background')
      expect(segments[0].style?.backgroundColor).toBeDefined()
    })

    it('parses bright/high-intensity colors', () => {
      const input = '\u001b[91mBright red\u001b[0m \u001b[104mBright blue bg\u001b[0m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(3)
      expect(segments[0].text).toBe('Bright red')
      expect(segments[0].style?.color).toBeDefined()
      expect(segments[1].text).toBe(' ')
      expect(segments[2].text).toBe('Bright blue bg')
      expect(segments[2].style?.backgroundColor).toBeDefined()
    })

    it('parses text styles: bold, dim, italic, underline', () => {
      const input = '\u001b[1mBold\u001b[22m \u001b[2mDim\u001b[22m \u001b[3mItalic\u001b[23m \u001b[4mUnderline\u001b[24m'
      const segments = parseAnsi(input)
      expect(segments[0].text).toBe('Bold')
      expect(segments[0].style?.fontWeight).toBe('bold')
      expect(segments[2].text).toBe('Dim')
      expect(segments[2].style?.opacity).toBe(0.6)
      expect(segments[4].text).toBe('Italic')
      expect(segments[4].style?.fontStyle).toBe('italic')
      expect(segments[6].text).toBe('Underline')
      expect(segments[6].style?.textDecoration).toBe('underline')
    })

    it('parses combined parameters in a single sequence', () => {
      const input = '\u001b[1;31;43mBold Red On Yellow\u001b[0m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(1)
      expect(segments[0].text).toBe('Bold Red On Yellow')
      expect(segments[0].style?.fontWeight).toBe('bold')
      expect(segments[0].style?.color).toBeDefined()
      expect(segments[0].style?.backgroundColor).toBeDefined()
    })

    it('parses 256 color sequences', () => {
      const input = '\u001b[38;5;196mRed 196\u001b[0m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(1)
      expect(segments[0].text).toBe('Red 196')
      expect(segments[0].style?.color).toBeDefined()
    })

    it('parses 24-bit TrueColor RGB sequences', () => {
      const input = '\u001b[38;2;123;200;80mRGB Green\u001b[0m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(1)
      expect(segments[0].text).toBe('RGB Green')
      expect(segments[0].style?.color).toBe('rgb(123, 200, 80)')
    })

    it('adapts colors in high-contrast mode', () => {
      const input = '\u001b[31mRed Alert\u001b[0m'
      const normalSegments = parseAnsi(input, { highContrast: false })
      const contrastSegments = parseAnsi(input, { highContrast: true })
      expect(normalSegments[0].style?.color).not.toBe(contrastSegments[0].style?.color)
      expect(contrastSegments[0].style?.color).toBe('#ff3333')
    })

    it('handles resets properly across segment transitions', () => {
      const input = '\u001b[32mOK\u001b[0m Normal \u001b[31mERR\u001b[m'
      const segments = parseAnsi(input)
      expect(segments).toHaveLength(3)
      expect(segments[0].text).toBe('OK')
      expect(segments[0].style?.color).toBeDefined()
      expect(segments[1].text).toBe(' Normal ')
      expect(segments[1].style?.color).toBeUndefined()
      expect(segments[2].text).toBe('ERR')
      expect(segments[2].style?.color).toBeDefined()
    })
  })
})
