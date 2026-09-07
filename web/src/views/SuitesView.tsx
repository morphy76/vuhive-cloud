import React, { useState } from 'react'
import { Layers, Plus, BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { VisuallyHidden } from '@/components/ui/visually-hidden'
import { HelpTooltip } from '@/components/help/HelpTooltip'
import { CreateSuiteDialog } from '@/components/dialogs/CreateSuiteDialog'
import { OfflinePreviewBadge } from '@/components/ui/offline-preview-badge'
import { useRecipe } from '@/context/RecipeContext'

export const SuitesView: React.FC = () => {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const { openRecipe } = useRecipe()

  const sampleSuites = [
    {
      id: 'suite-e2e-checkout',
      name: 'Checkout & Payment Stress Test',
      status: 'ACTIVE',
      buildStatus: 'READY',
      platforms: ['linux/amd64', 'linux/arm64'],
      updatedAt: '10m ago',
    },
    {
      id: 'suite-search-catalog',
      name: 'Product Catalog High Throughput',
      status: 'ACTIVE',
      buildStatus: 'READY',
      platforms: ['linux/arm64'],
      updatedAt: '2h ago',
    },
    {
      id: 'suite-auth-flood',
      name: 'OAuth2 Token Grant Barrier Test',
      status: 'ACTIVE',
      buildStatus: 'READY',
      platforms: ['linux/amd64'],
      updatedAt: '1d ago',
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white">
            Test Suites
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Source Packages & Artifacts: manage test scenarios and ephemeral cross-compilations.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <OfflinePreviewBadge />
          <Button
            variant="outline"
            onClick={() => openRecipe('recipe-1')}
            className="min-h-[44px] gap-2 border-slate-200 dark:border-slate-800"
            aria-label="View Suites API Recipe"
          >
            <BookOpen className="w-4 h-4 text-brand-600 dark:text-brand-400" />
            <span className="hidden sm:inline">API Recipe</span>
          </Button>
          <Button
            onClick={() => setIsCreateOpen(true)}
            className="min-h-[44px] gap-2"
          >
            <Plus className="w-4 h-4" />
            <span>New Suite</span>
          </Button>
        </div>
      </div>

      <div className="rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xs overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-slate-600 dark:text-slate-400">
            <caption>
              <VisuallyHidden>Test suites with build artifacts and target architectures</VisuallyHidden>
            </caption>
            <thead className="bg-slate-50 dark:bg-slate-800/60 text-xs uppercase font-semibold text-slate-500 dark:text-slate-400 border-b border-slate-200 dark:border-slate-800">
              <tr>
                <th scope="col" className="px-6 py-4">Suite Name</th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Status</span>
                    <HelpTooltip
                      text="Lifecycle state of the test suite (ACTIVE or ARCHIVED)."
                      label="Help for status column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Build Artifact</span>
                    <HelpTooltip
                      text="Cross-compilation status of scenario binary (READY, BUILDING, FAILED, or PENDING)."
                      label="Help for build artifact column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">
                  <div className="flex items-center gap-1.5">
                    <span>Target Arch</span>
                    <HelpTooltip
                      text="Target CPU architectures compiled statically for runner container execution."
                      label="Help for target architecture column"
                    />
                  </div>
                </th>
                <th scope="col" className="px-6 py-4">Updated</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
              {sampleSuites.map((s) => (
                <tr key={s.id} className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors">
                  <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                    <div className="flex items-center gap-3">
                      <Layers className="w-5 h-5 text-brand-500 flex-shrink-0" />
                      <div>
                        <div>{s.name}</div>
                        <div className="text-xs text-slate-400 font-mono">{s.id}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="inline-flex items-center gap-1.5">
                      <Badge variant="success">{s.status}</Badge>
                      <HelpTooltip
                        text="Active suite ready for scheduling and execution."
                        label="Help for suite status badge"
                      />
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="inline-flex items-center gap-1.5">
                      <Badge variant="info">{s.buildStatus}</Badge>
                      <HelpTooltip
                        text="Scenario statically compiled and packaged in S3 object store."
                        label="Help for build status badge"
                      />
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex gap-1.5 flex-wrap">
                      {s.platforms.map((p) => (
                        <span key={p} className="px-2 py-0.5 rounded-md text-xs font-mono bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300">
                          {p}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-xs font-mono">{s.updatedAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <CreateSuiteDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />
    </div>
  )
}
