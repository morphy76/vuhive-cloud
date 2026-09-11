import React from 'react'
import { AlertTriangle, ShieldAlert, Code2, CheckCircle } from 'lucide-react'

export interface AstErrorCalloutProps {
  error: string
  onAllowInsecureToggle?: () => void
  showInsecureOption?: boolean
}

export const AstErrorCallout: React.FC<AstErrorCalloutProps> = ({
  error,
  onAllowInsecureToggle,
  showInsecureOption = false,
}) => {
  const isForbiddenImport =
    error.toLowerCase().includes('forbidden') ||
    error.toLowerCase().includes('prohibited') ||
    error.toLowerCase().includes('os/exec') ||
    error.toLowerCase().includes('syscall') ||
    error.toLowerCase().includes('unsafe')

  const isMissingVuhive =
    error.toLowerCase().includes('vuhive') &&
    (error.toLowerCase().includes('missing') || error.toLowerCase().includes('indirect'))

  const isPackageMain =
    error.toLowerCase().includes('package main') ||
    error.toLowerCase().includes('func main')

  const isMissingScenario =
    error.toLowerCase().includes('scenario') && error.toLowerCase().includes('entrypoint')

  const getRemediation = () => {
    if (isMissingVuhive) {
      return {
        title: 'Missing Direct vuhive Dependency',
        rule: 'Direct Framework Dependency',
        tips: [
          'Add github.com/morphy76/vuhive as a direct dependency in go.mod.',
          'Run go get github.com/morphy76/vuhive@latest followed by go mod tidy.',
        ],
        codeSnippet: 'go get github.com/morphy76/vuhive@latest\ngo mod tidy',
      }
    }

    if (isPackageMain) {
      return {
        title: 'Package Main Prohibited (Inverted Control)',
        rule: 'Inverted Driver Architecture',
        tips: [
          'Do not declare "package main" or "func main()". The cloud builder automatically injects a hardened execution driver.',
          'Declare "package scenario" in your Go file and export a constructor such as func NewScenario() *vuhive.Scenario.',
        ],
        codeSnippet: 'package scenario\n\nimport "github.com/morphy76/vuhive/pkg/vuhive"\n\nfunc NewScenario() *vuhive.Scenario {\n    return vuhive.NewScenario("User Flow")\n}',
      }
    }

    if (isForbiddenImport) {
      return {
        title: 'Forbidden Package Import Detected',
        rule: 'AST Security Sandbox Enforcement',
        tips: [
          'Packages providing direct OS execution or low-level memory access (os/exec, syscall, unsafe, plugin) are blocked for safety.',
          'Replace system calls with standard Go net/http libraries or custom scenario hooks.',
          'If this package is genuinely required, you may toggle "Allow Insecure Imports" if permitted by policy.',
        ],
        codeSnippet: '// Remove forbidden imports:\n// import "os/exec" -> use http.Client or vuhive hooks instead',
      }
    }

    if (isMissingScenario) {
      return {
        title: 'Missing Scenario Entrypoint Contract',
        rule: 'Scenario Contract Requirement',
        tips: [
          'Ensure your package implements func NewScenario() *vuhive.Scenario, func Scenario(), or func Register(*vuhive.Engine).',
        ],
        codeSnippet: 'func NewScenario() *vuhive.Scenario {\n    return vuhive.NewScenario("Load Test")\n}',
      }
    }

    return {
      title: 'Package Verification Rejected',
      rule: 'Go Package Static Analysis',
      tips: [
        'Ensure the source package contains a valid go.mod and scenario.go implementing package scenario.',
      ],
      codeSnippet: null,
    }
  }

  const info = getRemediation()

  return (
    <div
      className="rounded-xl border border-red-200 dark:border-red-900/60 bg-red-50/70 dark:bg-red-950/40 p-4 space-y-3"
      role="alert"
      aria-label="AST Static Analysis Failure Alert"
    >
      <div className="flex items-start gap-3">
        <div className="p-2 rounded-lg bg-red-100 dark:bg-red-900/60 text-red-600 dark:text-red-400 mt-0.5">
          {isForbiddenImport ? (
            <ShieldAlert className="w-5 h-5" />
          ) : (
            <AlertTriangle className="w-5 h-5" />
          )}
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h4 className="text-sm font-semibold text-red-900 dark:text-red-200">
              AST Static Analysis Failed
            </h4>
            <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-red-100 dark:bg-red-900/70 text-red-700 dark:text-red-300 font-medium">
              {info.rule}
            </span>
          </div>
          <p className="text-xs text-red-800 dark:text-red-300/90 font-mono break-all">
            {error}
          </p>
        </div>
      </div>

      {/* Actionable Remediation Guidance */}
      <div className="bg-white/80 dark:bg-slate-900/80 rounded-lg p-3 border border-red-100 dark:border-red-900/40 space-y-2">
        <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-800 dark:text-slate-200">
          <CheckCircle className="w-3.5 h-3.5 text-emerald-500" />
          <span>How to fix this issue:</span>
        </div>
        <ul className="text-xs text-slate-600 dark:text-slate-400 space-y-1 list-disc pl-4">
          {info.tips.map((tip, i) => (
            <li key={i}>{tip}</li>
          ))}
        </ul>

        {info.codeSnippet && (
          <div className="mt-2">
            <div className="flex items-center gap-1 text-[11px] font-mono text-slate-400 mb-1">
              <Code2 className="w-3 h-3" />
              <span>Recommended Scenario Definition:</span>
            </div>
            <pre className="p-2 rounded bg-slate-900 text-slate-100 text-[11px] font-mono overflow-x-auto whitespace-pre">
              {info.codeSnippet}
            </pre>
          </div>
        )}

        {isForbiddenImport && showInsecureOption && onAllowInsecureToggle && (
          <button
            type="button"
            onClick={onAllowInsecureToggle}
            className="mt-2 text-xs font-semibold text-brand-600 dark:text-brand-400 hover:underline inline-flex items-center gap-1"
          >
            <span>Enable &quot;Allow Insecure Imports&quot; override and retry</span>
          </button>
        )}
      </div>
    </div>
  )
}
