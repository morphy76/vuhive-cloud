import React from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Hash } from 'lucide-react'
import { slugify } from '@/lib/markdown-utils'
import { CodeBlock } from './CodeBlock'
import { CalloutAlert } from './CalloutAlert'
import { cn } from '@/lib/utils'

interface MarkdownViewerProps {
  content: string
  className?: string
}

function getNodeText(node: any): string {
  if (!node) return ''
  if (typeof node === 'string') return node
  if (Array.isArray(node)) return node.map(getNodeText).join('')
  if (node.props?.children) return getNodeText(node.props.children)
  return ''
}

export const MarkdownViewer: React.FC<MarkdownViewerProps> = ({
  content,
  className,
}) => {
  return (
    <div
      className={cn(
        'markdown-viewer prose prose-slate dark:prose-invert max-w-none text-slate-800 dark:text-slate-200',
        className
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children, ...props }) => {
            const text = getNodeText(children)
            const id = slugify(text)
            return (
              <h1
                id={id}
                className="group flex items-center gap-2 text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 dark:text-white mt-10 mb-4 pb-2 border-b border-slate-200 dark:border-slate-800 scroll-mt-20"
                {...props}
              >
                <span>{children}</span>
                <a
                  href={`#${id}`}
                  className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity text-slate-400 hover:text-brand-600 dark:hover:text-brand-400 p-1"
                  aria-label={`Link to section: ${text}`}
                >
                  <Hash className="w-5 h-5" />
                </a>
              </h1>
            )
          },
          h2: ({ children, ...props }) => {
            const text = getNodeText(children)
            const id = slugify(text)
            return (
              <h2
                id={id}
                className="group flex items-center gap-2 text-xl sm:text-2xl font-semibold tracking-tight text-slate-900 dark:text-white mt-8 mb-3 pb-1.5 border-b border-slate-200/60 dark:border-slate-800/60 scroll-mt-20"
                {...props}
              >
                <span>{children}</span>
                <a
                  href={`#${id}`}
                  className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity text-slate-400 hover:text-brand-600 dark:hover:text-brand-400 p-1"
                  aria-label={`Link to section: ${text}`}
                >
                  <Hash className="w-4 h-4" />
                </a>
              </h2>
            )
          },
          h3: ({ children, ...props }) => {
            const text = getNodeText(children)
            const id = slugify(text)
            return (
              <h3
                id={id}
                className="group flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white mt-6 mb-2 scroll-mt-20"
                {...props}
              >
                <span>{children}</span>
                <a
                  href={`#${id}`}
                  className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity text-slate-400 hover:text-brand-600 dark:hover:text-brand-400 p-1"
                  aria-label={`Link to section: ${text}`}
                >
                  <Hash className="w-3.5 h-3.5" />
                </a>
              </h3>
            )
          },
          h4: ({ children, ...props }) => {
            const text = getNodeText(children)
            const id = slugify(text)
            return (
              <h4
                id={id}
                className="text-base font-semibold text-slate-900 dark:text-white mt-4 mb-2 scroll-mt-20"
                {...props}
              >
                {children}
              </h4>
            )
          },
          p: ({ children, ...props }) => (
            <p className="leading-7 text-slate-700 dark:text-slate-300 my-3 text-sm sm:text-base" {...props}>
              {children}
            </p>
          ),
          blockquote: ({ children }) => <CalloutAlert>{children}</CalloutAlert>,
          code: ({ className, children, ...props }) => {
            const match = /language-(\w+)/.exec(className || '')
            const isInline = !match && typeof children === 'string' && !children.includes('\n')

            if (isInline) {
              return (
                <code
                  className="rounded-md bg-slate-100 dark:bg-slate-800/80 px-1.5 py-0.5 font-mono text-xs font-semibold text-slate-800 dark:text-slate-200 border border-slate-200 dark:border-slate-700/60"
                  {...props}
                >
                  {children}
                </code>
              )
            }

            const codeString = String(children).replace(/\n$/, '')
            const language = match ? match[1] : undefined

            return (
              <CodeBlock language={language} className={className}>
                {codeString}
              </CodeBlock>
            )
          },
          pre: ({ children }) => <>{children}</>,
          table: ({ children, ...props }) => (
            <div className="my-5 overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-800 shadow-2xs">
              <table
                className="w-full text-left text-sm text-slate-700 dark:text-slate-300 divide-y divide-slate-200 dark:divide-slate-800"
                {...props}
              >
                {children}
              </table>
            </div>
          ),
          thead: ({ children, ...props }) => (
            <thead className="bg-slate-50 dark:bg-slate-800/60" {...props}>
              {children}
            </thead>
          ),
          th: ({ children, ...props }) => (
            <th
              className="px-4 py-3 text-xs font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300"
              {...props}
            >
              {children}
            </th>
          ),
          td: ({ children, ...props }) => (
            <td
              className="px-4 py-3 text-sm border-t border-slate-100 dark:border-slate-800/60"
              {...props}
            >
              {children}
            </td>
          ),
          ul: ({ children, ...props }) => (
            <ul className="list-disc list-inside space-y-1.5 my-3 text-sm sm:text-base text-slate-700 dark:text-slate-300" {...props}>
              {children}
            </ul>
          ),
          ol: ({ children, ...props }) => (
            <ol className="list-decimal list-inside space-y-1.5 my-3 text-sm sm:text-base text-slate-700 dark:text-slate-300" {...props}>
              {children}
            </ol>
          ),
          li: ({ children, ...props }) => (
            <li className="leading-6" {...props}>
              {children}
            </li>
          ),
          hr: ({ ...props }) => (
            <hr className="my-8 border-slate-200 dark:border-slate-800" {...props} />
          ),
          a: ({ href, children, ...props }) => {
            const isExternal = href?.startsWith('http://') || href?.startsWith('https://')
            const isAnchor = href?.startsWith('#')

            const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
              if (isAnchor && href) {
                e.preventDefault()
                const targetId = href.replace(/^#/, '')
                const element = document.getElementById(targetId)
                if (element) {
                  element.scrollIntoView({ behavior: 'smooth' })
                  window.history.pushState(null, '', href)
                }
              }
            }

            return (
              <a
                href={href}
                onClick={isAnchor ? handleClick : undefined}
                target={isExternal ? '_blank' : undefined}
                rel={isExternal ? 'noopener noreferrer' : undefined}
                className="font-medium text-brand-600 hover:text-brand-700 dark:text-brand-400 dark:hover:text-brand-300 underline underline-offset-2 transition-colors"
                {...props}
              >
                {children}
              </a>
            )
          },
          input: ({ type, checked, disabled, ...props }) => {
            if (type === 'checkbox') {
              return (
                <input
                  type="checkbox"
                  checked={checked}
                  disabled={true}
                  className="rounded border-slate-300 text-brand-600 focus:ring-brand-500 mr-2 h-4 w-4 align-middle"
                  aria-label={checked ? 'Completed task' : 'Pending task'}
                  {...props}
                />
              )
            }
            return <input type={type} {...props} />
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
