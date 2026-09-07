import React, { createContext, useContext, useState, useCallback } from 'react'
import { RecipeId } from '@/data/recipes'

interface RecipeContextType {
  isOpen: boolean
  activeRecipeId?: RecipeId
  openRecipe: (recipeId?: RecipeId) => void
  closeRecipe: () => void
}

const RecipeContext = createContext<RecipeContextType | null>(null)

export const RecipeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isOpen, setIsOpen] = useState<boolean>(false)
  const [activeRecipeId, setActiveRecipeId] = useState<RecipeId | undefined>(undefined)

  const openRecipe = useCallback((recipeId?: RecipeId) => {
    setActiveRecipeId(recipeId)
    setIsOpen(true)
  }, [])

  const closeRecipe = useCallback(() => {
    setIsOpen(false)
  }, [])

  return (
    <RecipeContext.Provider value={{ isOpen, activeRecipeId, openRecipe, closeRecipe }}>
      {children}
    </RecipeContext.Provider>
  )
}

const defaultRecipeContext: RecipeContextType = {
  isOpen: false,
  activeRecipeId: undefined,
  openRecipe: () => {},
  closeRecipe: () => {},
}

export const useRecipe = (): RecipeContextType => {
  const context = useContext(RecipeContext)
  return context || defaultRecipeContext
}
