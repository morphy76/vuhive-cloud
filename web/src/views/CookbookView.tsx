import React, { useState, useEffect, useMemo } from 'react'
import { BookOpen, CheckCircle2, ChevronUp } from 'lucide-react'
import defaultCookbookMarkdown from '@/docs/cookbook.md?raw'
import { MarkdownViewer } from '@/components/markdown/MarkdownViewer'
import { CookbookToc } from '@/components/markdown/CookbookToc'
import { extractHeadings } from '@/lib/markdown-utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

interface CookbookViewProps {
  content?: string
}

export const CookbookView: React.FC<CookbookViewProps> = ({ content }) => {
  const markdown = content ?? defaultCookbookMarkdown
  const headings = useMemo(() => extractHeadings(markdown), [markdown])
  const [activeId, setActiveId] = useState<string | undefined>(headings[0]?.id)
  const [showScrollTop, setShowScrollTop] = useState(false)

  // Scroll spy to highlight active TOC heading
  useEffect(() => {
    if (headings.length === 0) return

    const handleScroll = () => {
      setShowScrollTop(window.scrollY > 400)

      // Find heading closest to top of viewport
      const headingElements = headings
        .map((h) => ({ id: h.id, el: document.getElementById(h.id) }))
        .filter((item): item is { id: string; el: HTMLElement } => item.el !== null)

      const scrollPos = window.scrollY + 120

      for (let i = headingElements.length - 1; i >= 0; i--) {
        if (headingElements[i].el.offsetTop <= scrollPos) {
          setActiveId(headingElements[i].id)
          return
        }
      }
    }

    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [headings])

  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return (
    <div className="space-y-6">
      {/* Header section */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-slate-200 dark:border-slate-800 pb-5">
        <div className="flex items-start gap-3">
          <div className="p-2.5 rounded-xl bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400 border border-brand-200 dark:border-brand-800/60 mt-1">
            <BookOpen className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
                Control Plane Adoption Guide & Recipes
              </h1>
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
              End-to-end guidance for Kubernetes load testing, artifact compilation, distributed synchronization, and API recipes.
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          <Badge variant="outline" className="gap-1.5 py-1 px-2.5 text-xs text-emerald-600 dark:text-emerald-400 border-emerald-300 dark:border-emerald-800 bg-emerald-50/50 dark:bg-emerald-950/30">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" />
            <span>Offline Available</span>
          </Badge>
        </div>
      </div>

      {/* Main Content Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Main Markdown Article */}
        <main className="lg:col-span-8 xl:col-span-9 min-w-0 bg-white dark:bg-slate-900 rounded-2xl border border-slate-200 dark:border-slate-800 p-6 sm:p-8 shadow-xs">
          <MarkdownViewer content={markdown} />
        </main>

        {/* Sticky Table of Contents Sidebar */}
        <aside className="hidden lg:block lg:col-span-4 xl:col-span-3 sticky top-6">
          <CookbookToc
            headings={headings}
            activeId={activeId}
            onSelectHeading={setActiveId}
          />
        </aside>
      </div>

      {/* Scroll to top floating button */}
      {showScrollTop && (
        <Button
          variant="outline"
          size="icon"
          onClick={scrollToTop}
          aria-label="Scroll back to top"
          className="fixed bottom-20 right-6 z-40 rounded-full shadow-lg bg-white dark:bg-slate-800 border-slate-300 dark:border-slate-700 min-h-[44px] min-w-[44px]"
        >
          <ChevronUp className="w-5 h-5" />
        </Button>
      )}
    </div>
  )
}

export default CookbookView
