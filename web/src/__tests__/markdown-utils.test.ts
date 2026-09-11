import { describe, it, expect } from 'vitest'
import { slugify, extractHeadings, filterHeadings } from '../lib/markdown-utils'

describe('markdown-utils', () => {
  describe('slugify', () => {
    it('converts plain title to lowercase hyphenated slug', () => {
      expect(slugify('Core Domain Concepts')).toBe('core-domain-concepts')
    })

    it('strips special characters and normalizes punctuation', () => {
      expect(slugify('Recipe 1: Registering a Test Suite & Uploading Source Packages')).toBe(
        'recipe-1-registering-a-test-suite--uploading-source-packages'
      )
    })

    it('handles code formatting inside heading', () => {
      expect(slugify('Scenario Configuration (`vuhive.yaml`)')).toBe('scenario-configuration-vuhiveyaml')
    })

    it('trims leading and trailing hyphens', () => {
      expect(slugify('--- Header ---')).toBe('header')
    })
  })

  describe('extractHeadings', () => {
    it('extracts h1, h2, and h3 headings with correct levels and slugs', () => {
      const markdown = `
# Main Title

Introduction text.

## 1. Core Domain Concepts
Some overview.

### A. Test Scenario Structure
Scenario details.

#### Skipped H4
Should ignore level 4.

## 2. Authoring & Packaging Load Tests
Packaging details.
`
      const headings = extractHeadings(markdown)
      expect(headings).toEqual([
        {
          id: 'main-title',
          text: 'Main Title',
          level: 1,
        },
        {
          id: '1-core-domain-concepts',
          text: '1. Core Domain Concepts',
          level: 2,
        },
        {
          id: 'a-test-scenario-structure',
          text: 'A. Test Scenario Structure',
          level: 3,
        },
        {
          id: '2-authoring--packaging-load-tests',
          text: '2. Authoring & Packaging Load Tests',
          level: 2,
        },
      ])
    })

    it('ignores headings inside fenced code blocks', () => {
      const markdown = `
# Real Heading

\`\`\`bash
# This is a comment inside a code fence, not a heading
echo "hello"
## Another comment
\`\`\`

## Another Real Heading
`
      const headings = extractHeadings(markdown)
      expect(headings).toHaveLength(2)
      expect(headings[0].text).toBe('Real Heading')
      expect(headings[1].text).toBe('Another Real Heading')
    })

    it('handles duplicate heading text by deduplicating IDs', () => {
      const markdown = `
## Overview
First overview.

## Overview
Second overview.
`
      const headings = extractHeadings(markdown)
      expect(headings).toHaveLength(2)
      expect(headings[0].id).toBe('overview')
      expect(headings[1].id).toBe('overview-1')
    })
  })

  describe('filterHeadings', () => {
    const sampleHeadings = [
      { id: '1-core-domain-concepts', text: '1. Core Domain Concepts', level: 2 },
      { id: 'recipe-1-suites', text: 'Recipe 1: Registering a Test Suite', level: 2 },
      { id: 'recipe-2-builds', text: 'Recipe 2: Monitoring Build Status', level: 2 },
      { id: 'recipe-16-cli', text: 'Recipe 16: Adopting Developer CLI & Keycloak OIDC', level: 2 },
    ]

    it('returns all headings if query is empty or whitespace', () => {
      expect(filterHeadings(sampleHeadings, '')).toEqual(sampleHeadings)
      expect(filterHeadings(sampleHeadings, '   ')).toEqual(sampleHeadings)
    })

    it('filters headings case-insensitively', () => {
      const filtered = filterHeadings(sampleHeadings, 'keycloak')
      expect(filtered).toHaveLength(1)
      expect(filtered[0].id).toBe('recipe-16-cli')
    })

    it('matches partial tokens across text', () => {
      const filtered = filterHeadings(sampleHeadings, 'recipe')
      expect(filtered).toHaveLength(3)
    })
  })
})
