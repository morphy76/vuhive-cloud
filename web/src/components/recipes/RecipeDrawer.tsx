import React, { useState, useEffect, useRef } from 'react'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import {
  X,
  Terminal,
  Copy,
  Check,
  BookOpen,
  SlidersHorizontal,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { RECIPES, RecipeId, getRecipeForRoute, getRecipeById } from '@/data/recipes'
import { RouteId } from '@/types/navigation'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'

export interface RecipeDrawerProps {
  isOpen: boolean
  onClose: () => void
  currentRoute: RouteId
  initialRecipeId?: RecipeId
}

export const RecipeDrawer: React.FC<RecipeDrawerProps> = ({
  isOpen,
  onClose,
  currentRoute,
  initialRecipeId,
}) => {
  const { toast } = useToast()
  const [selectedRecipeId, setSelectedRecipeId] = useState<RecipeId>(() => {
    return initialRecipeId || getRecipeForRoute(currentRoute).id
  })

  // When route or initialRecipeId changes and drawer is reopened, sync recipe
  useEffect(() => {
    if (initialRecipeId) {
      setSelectedRecipeId(initialRecipeId)
    } else {
      setSelectedRecipeId(getRecipeForRoute(currentRoute).id)
    }
  }, [currentRoute, initialRecipeId, isOpen])

  const activeRecipe = getRecipeById(selectedRecipeId) || RECIPES[0]

  // Form parameters state initialized from recipe defaults
  const [params, setParams] = useState<Record<string, string>>({
    ...activeRecipe.defaultParams,
  })

  // Re-initialize parameters when recipe changes
  useEffect(() => {
    setParams({ ...activeRecipe.defaultParams })
  }, [selectedRecipeId, activeRecipe])

  const [copiedStepId, setCopiedStepId] = useState<string | null>(null)
  const [showConfig, setShowConfig] = useState<boolean>(true)

  // Touch swipe handling to close on right swipe
  const touchStartX = useRef<number | null>(null)
  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
  }
  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartX.current === null) return
    const touchEndX = e.changedTouches[0].clientX
    const deltaX = touchEndX - touchStartX.current
    if (deltaX > 50) {
      onClose()
    }
    touchStartX.current = null
  }

  const handleCopy = async (curlText: string, stepId: string) => {
    try {
      await navigator.clipboard.writeText(curlText)
      setCopiedStepId(stepId)
      setTimeout(() => setCopiedStepId(null), 2000)
      toast({
        title: 'cURL command copied!',
        description: 'Executable API request copied to clipboard.',
      })
    } catch {
      toast({
        title: 'Copy failed',
        description: 'Please copy the command manually.',
        variant: 'destructive',
      })
    }
  }

  const handleParamChange = (key: string, value: string) => {
    setParams((prev) => ({
      ...prev,
      [key]: value,
    }))
  }

  const getMethodBadgeVariant = (
    method: string
  ): 'default' | 'success' | 'warning' | 'error' | 'info' | 'outline' => {
    switch (method) {
      case 'POST':
        return 'info'
      case 'GET':
        return 'success'
      case 'PUT':
        return 'warning'
      case 'DELETE':
        return 'error'
      default:
        return 'outline'
    }
  }

  return (
    <DialogPrimitive.Root open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogPrimitive.Portal>
        {/* Backdrop Overlay */}
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-slate-950/60 backdrop-blur-xs duration-300 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        {/* Slide-over Content Drawer */}
        <DialogPrimitive.Content
          className={cn(
            'fixed inset-y-0 right-0 z-50 flex w-full max-w-2xl flex-col bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800 shadow-2xl duration-300 outline-none',
            'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right'
          )}
          onTouchStart={handleTouchStart}
          onTouchEnd={handleTouchEnd}
          aria-label="Recipe Guidance"
        >
          {/* Top Header */}
          <div className="flex h-16 items-center justify-between px-5 sm:px-6 border-b border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-900/50 backdrop-blur-xs flex-shrink-0">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-brand-600/10 dark:bg-brand-500/20 text-brand-600 dark:text-brand-400 flex items-center justify-center">
                <BookOpen className="w-4 h-4" />
              </div>
              <div className="flex flex-col">
                <span className="text-xs font-semibold uppercase tracking-wider text-brand-600 dark:text-brand-400">
                  Cookbook Guidance
                </span>
                <span className="text-sm font-bold text-slate-900 dark:text-white">
                  API Recipes & cURL Generator
                </span>
              </div>
            </div>

            <DialogPrimitive.Close
              onClick={onClose}
              aria-label="Close recipe guidance"
              className="rounded-lg p-2 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors min-h-[44px] min-w-[44px] inline-flex items-center justify-center focus:outline-none focus:ring-2 focus:ring-brand-500"
            >
              <X className="w-5 h-5" />
            </DialogPrimitive.Close>
          </div>

          {/* Body Content - Scrollable */}
          <div className="flex-1 overflow-y-auto p-5 sm:p-6 space-y-6">
            {/* Recipe Selector Dropdown */}
            <div className="space-y-2">
              <label
                htmlFor="recipe-select"
                className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400"
              >
                Select API Recipe
              </label>
              <div className="relative">
                <select
                  id="recipe-select"
                  aria-label="Select API Recipe"
                  value={selectedRecipeId}
                  onChange={(e) => setSelectedRecipeId(e.target.value as RecipeId)}
                  className="w-full appearance-none rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/80 px-4 py-2.5 text-sm font-medium text-slate-900 dark:text-slate-100 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20 min-h-[44px]"
                >
                  {RECIPES.map((recipe) => (
                    <option key={recipe.id} value={recipe.id}>
                      {recipe.title}
                    </option>
                  ))}
                </select>
                <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-slate-400">
                  <ChevronDown className="w-4 h-4" />
                </div>
              </div>
            </div>

            {/* Recipe Heading & Overview */}
            <div className="space-y-2 bg-brand-50/50 dark:bg-brand-950/30 p-4 rounded-2xl border border-brand-100 dark:border-brand-900/40">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <DialogPrimitive.Title className="text-base sm:text-lg font-bold text-slate-900 dark:text-white leading-tight">
                  {activeRecipe.title}
                </DialogPrimitive.Title>
                <Badge variant="outline" className="text-xs font-mono">
                  {activeRecipe.cookbookRef}
                </Badge>
              </div>
              <DialogPrimitive.Description className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                {activeRecipe.description}
              </DialogPrimitive.Description>
            </div>

            {/* Parameter Configuration Panel */}
            <div className="rounded-2xl border border-slate-200 dark:border-slate-800 overflow-hidden bg-white dark:bg-slate-900 shadow-xs">
              <button
                type="button"
                onClick={() => setShowConfig((prev) => !prev)}
                className="w-full flex items-center justify-between p-4 text-left font-medium text-sm text-slate-800 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                aria-expanded={showConfig}
              >
                <div className="flex items-center gap-2">
                  <SlidersHorizontal className="w-4 h-4 text-brand-600 dark:text-brand-400" />
                  <span>Interactive Dynamic Parameters</span>
                </div>
                {showConfig ? (
                  <ChevronUp className="w-4 h-4 text-slate-400" />
                ) : (
                  <ChevronDown className="w-4 h-4 text-slate-400" />
                )}
              </button>

              {showConfig && (
                <div className="p-4 pt-0 border-t border-slate-100 dark:border-slate-800 grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                  {activeRecipe.paramFields.map((field) => (
                    <div key={field.key} className="space-y-1">
                      <label
                        htmlFor={`param-${field.key}`}
                        className="text-xs font-medium text-slate-600 dark:text-slate-400"
                      >
                        {field.label}
                      </label>
                      <input
                        id={`param-${field.key}`}
                        type="text"
                        value={params[field.key] ?? field.defaultValue}
                        onChange={(e) => handleParamChange(field.key, e.target.value)}
                        placeholder={field.placeholder}
                        className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 px-3 py-1.5 text-xs text-slate-900 dark:text-slate-100 font-mono focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Recipe Steps Walkthrough & Generated cURL */}
            <div className="space-y-4">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                Recipe Steps & Executable cURL
              </h2>

              {activeRecipe.steps.map((step, idx) => {
                const generatedCurl = step.generateCurl(params)
                const isCopied = copiedStepId === step.id

                return (
                  <div
                    key={step.id}
                    className="rounded-2xl border border-slate-200 dark:border-slate-800 p-4 sm:p-5 bg-white dark:bg-slate-900 shadow-xs space-y-3"
                  >
                    {/* Step Title & Endpoint Info */}
                    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                      <div>
                        <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                          {step.title}
                        </h3>
                        <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                          {step.description}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <Badge variant={getMethodBadgeVariant(step.method)} className="text-[11px] font-mono">
                          {step.method}
                        </Badge>
                        <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
                          {step.endpoint}
                        </span>
                      </div>
                    </div>

                    {/* Generated cURL Code Box */}
                    <div className="relative rounded-xl bg-slate-950 dark:bg-black/90 p-4 font-mono text-xs text-emerald-400 overflow-x-auto shadow-inner border border-slate-800">
                      <pre data-testid={idx === 0 ? 'curl-preview' : `curl-step-${step.id}`}>
                        <code>{generatedCurl}</code>
                      </pre>
                      <button
                        type="button"
                        onClick={() => handleCopy(generatedCurl, step.id)}
                        aria-label={`Copy step ${idx + 1} cURL`}
                        className="absolute top-3 right-3 p-1.5 rounded-lg bg-slate-800/80 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
                      >
                        {isCopied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                      </button>
                    </div>

                    {/* Copy as cURL primary CTA button */}
                    <div className="flex justify-end pt-1">
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => handleCopy(generatedCurl, step.id)}
                        aria-label="Copy as cURL"
                        className="gap-1.5 text-xs min-h-[36px]"
                      >
                        {isCopied ? (
                          <>
                            <Check className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                            <span>Copied!</span>
                          </>
                        ) : (
                          <>
                            <Terminal className="w-3.5 h-3.5" />
                            <span>Copy as cURL</span>
                          </>
                        )}
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
