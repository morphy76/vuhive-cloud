import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * Merges class names using clsx for conditional logic and tailwind-merge
 * for resolving Tailwind CSS class conflicts.
 *
 * This is the standard shadcn/ui `cn()` utility used by all UI primitives.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
