import type React from 'react'

export interface AnsiSegment {
  text: string
  style?: React.CSSProperties
  className?: string
}

export interface ParseAnsiOptions {
  highContrast?: boolean
}

// Regex to match ANSI escape sequences (CSI sequences like \x1b[...m and control sequences)
const ANSI_REGEX = /(?:\u001b|\\u001b|\x1b)\[[0-9;?]*[a-zA-Z]/g
const SGR_REGEX = /(?:\u001b|\\u001b|\x1b)\[([0-9;]*)m/g
// Non-SGR CSI sequences (letters except 'm')
const NON_SGR_REGEX = /(?:\u001b|\\u001b|\x1b)\[[0-9;?]*[A-LN-Za-ln-z]/g

// Standard ANSI dark palette
const STANDARD_FG: Record<number, string> = {
  30: '#64748b', // Black / Dark Slate
  31: '#f87171', // Red
  32: '#4ade80', // Green
  33: '#facc15', // Yellow
  34: '#60a5fa', // Blue
  35: '#c084fc', // Magenta
  36: '#38bdf8', // Cyan
  37: '#f1f5f9', // White
}

const HIGH_CONTRAST_FG: Record<number, string> = {
  30: '#94a3b8',
  31: '#ff3333',
  32: '#00ff66',
  33: '#ffff00',
  34: '#3399ff',
  35: '#ff33ff',
  36: '#00ffff',
  37: '#ffffff',
}

const BRIGHT_FG: Record<number, string> = {
  90: '#94a3b8', // Bright Black / Gray
  91: '#ef4444', // Bright Red
  92: '#22c55e', // Bright Green
  93: '#eab308', // Bright Yellow
  94: '#3b82f6', // Bright Blue
  95: '#a855f7', // Bright Magenta
  96: '#06b6d4', // Bright Cyan
  97: '#ffffff', // Bright White
}

const HIGH_CONTRAST_BRIGHT_FG: Record<number, string> = {
  90: '#cbd5e1',
  91: '#ff4d4d',
  92: '#33ff77',
  93: '#ffff33',
  94: '#4da6ff',
  95: '#ff4dff',
  96: '#33ffff',
  97: '#ffffff',
}

const STANDARD_BG: Record<number, string> = {
  40: '#0f172a',
  41: '#991b1b',
  42: '#166534',
  43: '#854d0e',
  44: '#1e40af',
  45: '#6b21a8',
  46: '#155e75',
  47: '#e2e8f0',
}

const BRIGHT_BG: Record<number, string> = {
  100: '#334155',
  101: '#dc2626',
  102: '#16a34a',
  103: '#ca8a04',
  104: '#2563eb',
  105: '#9333ea',
  106: '#0891b2',
  107: '#f8fafc',
}

/**
 * Strips all ANSI escape sequences and terminal control codes from a string.
 */
export function stripAnsi(str: string): string {
  if (!str) return ''
  return str
    .replace(/(?:\u001b|\\u001b|\x1b)\][^\u0007\u001b]*(?:\u0007|\u001b\\)/g, '') // OSC sequences
    .replace(ANSI_REGEX, '') // CSI & other sequences
    .replace(/[\u0000-\u0008\u000b-\u001a\u001c-\u001f]/g, '') // non-printable control chars
}

/**
 * Resolves an 8-bit ANSI 256-color index (0-255) to an RGB string.
 */
function get256Color(index: number, highContrast = false): string {
  if (index < 8) {
    const code = 30 + index
    return (highContrast ? HIGH_CONTRAST_FG : STANDARD_FG)[code] || '#ffffff'
  }
  if (index < 16) {
    const code = 90 + (index - 8)
    return (highContrast ? HIGH_CONTRAST_BRIGHT_FG : BRIGHT_FG)[code] || '#ffffff'
  }
  if (index >= 16 && index <= 231) {
    const adjusted = index - 16
    const r = Math.floor(adjusted / 36) * 51
    const g = Math.floor((adjusted % 36) / 6) * 51
    const b = (adjusted % 6) * 51
    return `rgb(${r}, ${g}, ${b})`
  }
  if (index >= 232 && index <= 255) {
    const c = 8 + (index - 232) * 10
    return `rgb(${c}, ${c}, ${c})`
  }
  return '#ffffff'
}

interface CurrentStyle {
  fg?: string
  bg?: string
  bold?: boolean
  dim?: boolean
  italic?: boolean
  underline?: boolean
}

function buildCssProperties(style: CurrentStyle): React.CSSProperties | undefined {
  const props: React.CSSProperties = {}
  let hasProps = false

  if (style.fg) {
    props.color = style.fg
    hasProps = true
  }
  if (style.bg) {
    props.backgroundColor = style.bg
    hasProps = true
  }
  if (style.bold) {
    props.fontWeight = 'bold'
    hasProps = true
  }
  if (style.dim) {
    props.opacity = 0.6
    hasProps = true
  }
  if (style.italic) {
    props.fontStyle = 'italic'
    hasProps = true
  }
  if (style.underline) {
    props.textDecoration = 'underline'
    hasProps = true
  }

  return hasProps ? props : undefined
}

/**
 * Parses an ANSI string into styled segments ready for rendering as HTML spans.
 */
export function parseAnsi(
  input: string,
  options: ParseAnsiOptions = {}
): AnsiSegment[] {
  if (!input) return []

  const highContrast = Boolean(options.highContrast)
  const fgMap = highContrast ? HIGH_CONTRAST_FG : STANDARD_FG
  const brightFgMap = highContrast ? HIGH_CONTRAST_BRIGHT_FG : BRIGHT_FG

  const segments: AnsiSegment[] = []
  let currentStyle: CurrentStyle = {}

  // Clean OSC and non-SGR sequences first, keeping SGR codes intact
  const cleaned = input
    .replace(/(?:\u001b|\\u001b|\x1b)\][^\u0007\u001b]*(?:\u0007|\u001b\\)/g, '')
    .replace(NON_SGR_REGEX, '')

  let lastIndex = 0
  const regex = new RegExp(SGR_REGEX.source, 'g')
  let match: RegExpExecArray | null

  while ((match = regex.exec(cleaned)) !== null) {
    const textBefore = cleaned.slice(lastIndex, match.index)
    if (textBefore) {
      const style = buildCssProperties(currentStyle)
      segments.push(style ? { text: textBefore, style } : { text: textBefore })
    }

    lastIndex = regex.lastIndex

    // Process SGR codes
    const paramsStr = match[1]
    const params = paramsStr ? paramsStr.split(';').map(Number) : [0]

    for (let i = 0; i < params.length; i++) {
      const code = params[i]

      if (code === 0 || isNaN(code)) {
        // Reset all
        currentStyle = {}
      } else if (code === 1) {
        currentStyle.bold = true
      } else if (code === 2) {
        currentStyle.dim = true
      } else if (code === 3) {
        currentStyle.italic = true
      } else if (code === 4) {
        currentStyle.underline = true
      } else if (code === 22) {
        currentStyle.bold = false
        currentStyle.dim = false
      } else if (code === 23) {
        currentStyle.italic = false
      } else if (code === 24) {
        currentStyle.underline = false
      } else if (code >= 30 && code <= 37) {
        currentStyle.fg = fgMap[code]
      } else if (code === 39) {
        currentStyle.fg = undefined
      } else if (code >= 40 && code <= 47) {
        currentStyle.bg = STANDARD_BG[code]
      } else if (code === 49) {
        currentStyle.bg = undefined
      } else if (code >= 90 && code <= 97) {
        currentStyle.fg = brightFgMap[code]
      } else if (code >= 100 && code <= 107) {
        currentStyle.bg = BRIGHT_BG[code]
      } else if (code === 38 || code === 48) {
        // Extended 256 or TrueColor
        const isFg = code === 38
        const mode = params[i + 1]

        if (mode === 5 && i + 2 < params.length) {
          // 256 colors: 38;5;n
          const colorIndex = params[i + 2]
          const colorVal = get256Color(colorIndex, highContrast)
          if (isFg) currentStyle.fg = colorVal
          else currentStyle.bg = colorVal
          i += 2
        } else if (mode === 2 && i + 4 < params.length) {
          // TrueColor: 38;2;r;g;b
          const r = params[i + 2]
          const g = params[i + 3]
          const b = params[i + 4]
          const colorVal = `rgb(${r}, ${g}, ${b})`
          if (isFg) currentStyle.fg = colorVal
          else currentStyle.bg = colorVal
          i += 4
        }
      }
    }
  }

  // Trailing text
  const trailingText = cleaned.slice(lastIndex)
  if (trailingText) {
    const style = buildCssProperties(currentStyle)
    segments.push(style ? { text: trailingText, style } : { text: trailingText })
  }

  return segments
}
