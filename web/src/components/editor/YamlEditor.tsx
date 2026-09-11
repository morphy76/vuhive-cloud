import React, { useState, useEffect, useMemo, useCallback } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { oneDark } from '@codemirror/theme-one-dark'
import {
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  FileCode,
  ChevronDown,
  Sparkles,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  validateYaml,
  validateVuhiveSchema,
  formatYamlError,
  VUHIVE_YAML_TEMPLATES,
  type YamlValidationResult,
  type SchemaValidationResult,
  type VuhiveYamlTemplate,
} from '@/lib/yaml-validator'

export interface YamlEditorProps {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  height?: string
  minHeight?: string
  maxHeight?: string
  placeholder?: string
  ariaLabel?: string
  id?: string
  showTemplates?: boolean
  onSelectTemplate?: (template: VuhiveYamlTemplate) => void
  onValidationChange?: (result: YamlValidationResult, schema: SchemaValidationResult) => void
  className?: string
}

export const YamlEditor: React.FC<YamlEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  height = '280px',
  minHeight = '180px',
  maxHeight = '500px',
  placeholder = '# Enter scenario configuration YAML here...',
  ariaLabel = 'YAML Configuration Editor',
  id = 'yaml-editor',
  showTemplates = false,
  onSelectTemplate,
  onValidationChange,
  className = '',
}) => {
  const [isDarkMode, setIsDarkMode] = useState(() => {
    if (typeof document !== 'undefined') {
      return document.documentElement.classList.contains('dark')
    }
    return false
  })
  const [isTemplateMenuOpen, setIsTemplateMenuOpen] = useState(false)

  // Observe theme change on root element
  useEffect(() => {
    if (typeof document === 'undefined') return
    const observer = new MutationObserver(() => {
      setIsDarkMode(document.documentElement.classList.contains('dark'))
    })
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
    return () => observer.disconnect()
  }, [])

  // Validate on value change
  const { validation, schema } = useMemo(() => {
    const valResult = validateYaml(value)
    const schemaResult = valResult.isValid
      ? validateVuhiveSchema(valResult.parsed)
      : { isValid: false, errors: [], warnings: [] }
    return { validation: valResult, schema: schemaResult }
  }, [value])

  // Propagate validation results
  useEffect(() => {
    onValidationChange?.(validation, schema)
  }, [validation, schema, onValidationChange])

  const handleTemplateClick = useCallback(
    (template: VuhiveYamlTemplate) => {
      setIsTemplateMenuOpen(false)
      if (onSelectTemplate) {
        onSelectTemplate(template)
      } else if (onChange) {
        onChange(template.yaml)
      }
    },
    [onChange, onSelectTemplate]
  )

  const extensions = useMemo(() => [yaml()], [])

  return (
    <div
      className={`rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 overflow-hidden shadow-xs flex flex-col ${className}`}
    >
      {/* Editor Toolbar */}
      <div className="flex flex-wrap items-center justify-between gap-2 px-3.5 py-2 bg-slate-50 dark:bg-slate-800/60 border-b border-slate-200 dark:border-slate-800">
        <div className="flex items-center gap-2">
          <FileCode className="w-4 h-4 text-brand-600 dark:text-brand-400" />
          <span className="text-xs font-semibold text-slate-700 dark:text-slate-300">
            vuhive.yaml
          </span>
          {readOnly && (
            <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-slate-200 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
              Read Only
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* Preset Templates Dropdown */}
          {showTemplates && !readOnly && (
            <div className="relative">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setIsTemplateMenuOpen((prev) => !prev)}
                className="h-7 text-xs gap-1.5 px-2.5 border-slate-200 dark:border-slate-700"
                aria-haspopup="true"
                aria-expanded={isTemplateMenuOpen}
              >
                <Sparkles className="w-3.5 h-3.5 text-brand-500" />
                <span>Load Template</span>
                <ChevronDown className="w-3 h-3 text-slate-400" />
              </Button>

              {isTemplateMenuOpen && (
                <div
                  role="menu"
                  className="absolute right-0 mt-1 w-64 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl z-20 py-1"
                >
                  <div className="px-3 py-1 text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                    Preset Scenarios
                  </div>
                  {VUHIVE_YAML_TEMPLATES.map((tmpl) => (
                    <button
                      key={tmpl.id}
                      type="button"
                      role="menuitem"
                      onClick={() => handleTemplateClick(tmpl)}
                      className="w-full text-left px-3 py-2 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
                    >
                      <div className="text-xs font-semibold text-slate-900 dark:text-white">
                        {tmpl.name}
                      </div>
                      <div className="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-1">
                        {tmpl.description}
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Validation Status Badge in Toolbar */}
          {validation.isValid ? (
            <span className="flex items-center gap-1 text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
              <CheckCircle2 className="w-3.5 h-3.5" />
              <span>Valid YAML</span>
            </span>
          ) : (
            <span className="flex items-center gap-1 text-[11px] font-medium text-red-600 dark:text-red-400">
              <AlertCircle className="w-3.5 h-3.5" />
              <span>Syntax Error</span>
            </span>
          )}
        </div>
      </div>

      {/* CodeMirror Surface */}
      <div
        id={id}
        role="textbox"
        aria-label={ariaLabel}
        aria-multiline="true"
        aria-invalid={!validation.isValid}
        className="font-mono text-xs overflow-auto"
      >
        <CodeMirror
          value={value}
          height={height}
          minHeight={minHeight}
          maxHeight={maxHeight}
          theme={isDarkMode ? oneDark : undefined}
          extensions={extensions}
          onChange={onChange}
          readOnly={readOnly}
          placeholder={placeholder}
          basicSetup={{
            lineNumbers: true,
            foldGutter: true,
            bracketMatching: true,
            closeBrackets: true,
            autocompletion: true,
            highlightActiveLine: !readOnly,
          }}
        />
      </div>

      {/* Validation Diagnostic Footer Bar */}
      {!validation.isValid && validation.errors.length > 0 && (
        <div
          role="alert"
          className="p-3 bg-red-50 dark:bg-red-950/50 border-t border-red-200 dark:border-red-900/60 text-xs text-red-700 dark:text-red-300 flex items-start gap-2"
        >
          <AlertCircle className="w-4 h-4 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
          <div className="space-y-0.5">
            <span className="font-semibold">YAML Syntax Error:</span>
            <ul className="list-disc list-inside space-y-0.5">
              {validation.errors.map((err, i) => (
                <li key={i} className="font-mono text-[11px]">
                  {formatYamlError(err)}
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}

      {/* Schema Warnings Footer (if YAML syntax is valid but schema has suggestions) */}
      {validation.isValid && schema.warnings.length > 0 && (
        <div className="px-3.5 py-2 bg-amber-50 dark:bg-amber-950/40 border-t border-amber-200 dark:border-amber-900/40 text-[11px] text-amber-800 dark:text-amber-200 flex items-center gap-2">
          <AlertTriangle className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400 shrink-0" />
          <span className="truncate">
            {schema.warnings[0]}
            {schema.warnings.length > 1 && ` (+${schema.warnings.length - 1} more recommendations)`}
          </span>
        </div>
      )}
    </div>
  )
}
