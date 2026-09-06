import React from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme } from '@/context/ThemeContext'

export const ThemeToggle: React.FC<{ className?: string }> = ({ className = '' }) => {
  const { resolvedTheme, toggleTheme } = useTheme()

  return (
    <button
      type="button"
      onClick={toggleTheme}
      aria-label="Toggle theme"
      className={`inline-flex items-center justify-center p-2 rounded-lg text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors focus:outline-none focus:ring-2 focus:ring-brand-500 min-h-[44px] min-w-[44px] ${className}`}
    >
      {resolvedTheme === 'dark' ? (
        <Sun className="w-5 h-5 text-amber-400" />
      ) : (
        <Moon className="w-5 h-5 text-slate-600" />
      )}
    </button>
  )
}
