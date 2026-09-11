import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { CookbookView } from '../views/CookbookView'

const sampleMarkdown = `
# Adoption Guide

Welcome to the adoption guide.

## 1. Core Domain Concepts
Explanation of domain concepts.

## 2. Authoring Load Tests
How to author scenarios.

### Recipe 1: Registering a Test Suite
Instructions for registering a suite.
`

describe('CookbookView', () => {
  it('renders cookbook header, offline indicator, and markdown content', () => {
    render(<CookbookView content={sampleMarkdown} />)

    expect(
      screen.getByRole('heading', { level: 1, name: /control plane adoption guide/i })
    ).toBeInTheDocument()
    expect(screen.getByText(/offline available/i)).toBeInTheDocument()
    expect(screen.getByText('Explanation of domain concepts.')).toBeInTheDocument()
  })

  it('renders Table of Contents with extracted headings', () => {
    render(<CookbookView content={sampleMarkdown} />)

    const toc = screen.getByRole('navigation', { name: /table of contents/i })
    expect(toc).toBeInTheDocument()
    expect(within(toc).getByText('1. Core Domain Concepts')).toBeInTheDocument()
    expect(within(toc).getByText('Recipe 1: Registering a Test Suite')).toBeInTheDocument()
  })

  it('filters Table of Contents entries using search input', () => {
    render(<CookbookView content={sampleMarkdown} />)

    const toc = screen.getByRole('navigation', { name: /table of contents/i })
    const searchInput = within(toc).getByRole('searchbox', { name: /filter recipes and sections/i })
    fireEvent.change(searchInput, { target: { value: 'Recipe 1' } })

    expect(within(toc).getByText('Recipe 1: Registering a Test Suite')).toBeInTheDocument()
    expect(within(toc).queryByText('1. Core Domain Concepts')).not.toBeInTheDocument()
  })

  it('scrolls to heading when TOC link is clicked', () => {
    const scrollIntoViewMock = vi.fn()
    window.HTMLElement.prototype.scrollIntoView = scrollIntoViewMock

    render(<CookbookView content={sampleMarkdown} />)

    const toc = screen.getByRole('navigation', { name: /table of contents/i })
    const link = within(toc).getByRole('link', { name: '1. Core Domain Concepts' })
    fireEvent.click(link)

    expect(scrollIntoViewMock).toHaveBeenCalled()
  })
})
