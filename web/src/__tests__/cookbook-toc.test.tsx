import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CookbookToc } from '../components/markdown/CookbookToc'
import { TocHeading } from '../lib/markdown-utils'

const mockHeadings: TocHeading[] = [
  { id: '1-core-domain-concepts', text: '1. Core Domain Concepts', level: 2 },
  { id: '2-authoring-load-tests', text: '2. Authoring & Packaging Load Tests', level: 2 },
  { id: 'recipe-1-suites', text: 'Recipe 1: Registering a Test Suite', level: 3 },
  { id: 'recipe-2-builds', text: 'Recipe 2: Monitoring Build Status', level: 3 },
  { id: 'recipe-16-cli', text: 'Recipe 16: Adopting Developer CLI & Keycloak OIDC', level: 3 },
]

describe('CookbookToc', () => {
  it('renders Table of Contents header and all headings by default', () => {
    render(<CookbookToc headings={mockHeadings} />)

    expect(screen.getByText(/table of contents/i)).toBeInTheDocument()
    expect(screen.getByText('1. Core Domain Concepts')).toBeInTheDocument()
    expect(screen.getByText('Recipe 1: Registering a Test Suite')).toBeInTheDocument()
    expect(screen.getByText('Recipe 16: Adopting Developer CLI & Keycloak OIDC')).toBeInTheDocument()
  })

  it('filters headings based on search query', () => {
    render(<CookbookToc headings={mockHeadings} />)

    const searchInput = screen.getByRole('searchbox', { name: /filter recipes and sections/i })
    fireEvent.change(searchInput, { target: { value: 'keycloak' } })

    expect(screen.getByText('Recipe 16: Adopting Developer CLI & Keycloak OIDC')).toBeInTheDocument()
    expect(screen.queryByText('1. Core Domain Concepts')).not.toBeInTheDocument()
    expect(screen.queryByText('Recipe 2: Monitoring Build Status')).not.toBeInTheDocument()
  })

  it('displays empty state message when no headings match search', () => {
    render(<CookbookToc headings={mockHeadings} />)

    const searchInput = screen.getByRole('searchbox', { name: /filter recipes and sections/i })
    fireEvent.change(searchInput, { target: { value: 'nonexistent-query-12345' } })

    expect(screen.getByText(/no sections matching/i)).toBeInTheDocument()
  })

  it('calls onSelectHeading when an item is clicked', () => {
    const handleSelect = vi.fn()
    render(<CookbookToc headings={mockHeadings} onSelectHeading={handleSelect} />)

    const item = screen.getByRole('link', { name: 'Recipe 1: Registering a Test Suite' })
    fireEvent.click(item)

    expect(handleSelect).toHaveBeenCalledWith('recipe-1-suites')
  })

  it('marks active heading with aria-current="true"', () => {
    render(<CookbookToc headings={mockHeadings} activeId="recipe-1-suites" />)

    const activeItem = screen.getByRole('link', { name: 'Recipe 1: Registering a Test Suite' })
    expect(activeItem).toHaveAttribute('aria-current', 'true')

    const inactiveItem = screen.getByRole('link', { name: 'Recipe 2: Monitoring Build Status' })
    expect(inactiveItem).not.toHaveAttribute('aria-current')
  })
})
