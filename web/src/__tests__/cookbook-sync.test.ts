import { describe, it, expect } from 'vitest'
import fs from 'fs'
import path from 'path'
import cookbookRaw from '../docs/cookbook.md?raw'

describe('Cookbook Offline Asset Sync & Delivery', () => {
  it('bundles cookbook raw markdown with non-empty content', () => {
    expect(cookbookRaw).toBeDefined()
    expect(cookbookRaw.length).toBeGreaterThan(1000)
    expect(cookbookRaw).toContain('vuhive-cloud Adoption Guide & API Recipes (Cookbook)')
  })

  it('keeps web/src/docs/cookbook.md synchronized with root docs/cookbook.md', () => {
    const rootCookbookPath = path.resolve(__dirname, '../../../../docs/cookbook.md')
    if (fs.existsSync(rootCookbookPath)) {
      const rootContent = fs.readFileSync(rootCookbookPath, 'utf-8')
      const localCookbookPath = path.resolve(__dirname, '../docs/cookbook.md')
      const localContent = fs.readFileSync(localCookbookPath, 'utf-8')

      expect(localContent).toBe(rootContent)
    }
  })
})
