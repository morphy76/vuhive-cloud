export interface TocHeading {
  id: string
  text: string
  level: number
}

/**
 * Converts heading text into a clean, URL-safe slug matching GitHub Flavored Markdown rules,
 * preserving double hyphens resulting from removed ampersands and punctuation, and trimming
 * leading/trailing hyphens.
 */
export function slugify(text: string): string {
  return text
    // Remove markdown inline formatting (e.g. `code`, **bold**, *italic*)
    .replace(/[`*_~]/g, '')
    // Replace non-alphanumeric characters except spaces and hyphens with empty
    .replace(/[^\w\s-]/g, '')
    .trim()
    // Convert each whitespace to a hyphen (preserving multi-spaces as double hyphens)
    .replace(/\s/g, '-')
    // Trim leading and trailing hyphens
    .replace(/^-+|-+$/g, '')
    .toLowerCase()
}

/**
 * Parses markdown string and extracts h1 (#), h2 (##), and h3 (###) headings,
 * ignoring lines inside code fences. Deduplicates IDs when multiple headings share identical text.
 */
export function extractHeadings(markdown: string): TocHeading[] {
  const headings: TocHeading[] = []
  const idCounts = new Map<string, number>()
  const lines = markdown.split('\n')

  let inCodeFence = false

  for (const line of lines) {
    const trimmed = line.trim()

    // Detect fenced code block start/end
    if (trimmed.startsWith('```') || trimmed.startsWith('~~~')) {
      inCodeFence = !inCodeFence
      continue
    }

    if (inCodeFence) {
      continue
    }

    // Match #, ##, or ###
    const match = trimmed.match(/^(#{1,3})\s+(.+)$/)
    if (match) {
      const level = match[1].length
      const rawText = match[2].trim()
      // Display text: remove enclosing code backticks or bold markers for cleaner TOC presentation
      const cleanText = rawText.replace(/[`*]/g, '')
      const baseSlug = slugify(rawText)

      let id = baseSlug
      const count = idCounts.get(baseSlug) || 0
      if (count > 0) {
        id = `${baseSlug}-${count}`
      }
      idCounts.set(baseSlug, count + 1)

      headings.push({
        id,
        text: cleanText,
        level,
      })
    }
  }

  return headings
}

/**
 * Filters a list of TOC headings based on case-insensitive keyword search query.
 */
export function filterHeadings(headings: TocHeading[], query: string): TocHeading[] {
  const normalizedQuery = query.trim().toLowerCase()
  if (!normalizedQuery) {
    return headings
  }

  return headings.filter((heading) =>
    heading.text.toLowerCase().includes(normalizedQuery) ||
    heading.id.toLowerCase().includes(normalizedQuery)
  )
}
