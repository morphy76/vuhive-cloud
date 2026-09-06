import * as React from "react"
import { cn } from "@/lib/utils"

export interface SkipLinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {}

const SkipLink = React.forwardRef<HTMLAnchorElement, SkipLinkProps>(
  ({ className, href = "#main-content", children = "Skip to main content", ...props }, ref) => {
    return (
      <a
        ref={ref}
        href={href}
        className={cn(
          "sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:rounded-xl focus:bg-brand-600 focus:px-4 focus:py-2 focus:text-white focus:font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500 focus-visible:ring-offset-2",
          className
        )}
        {...props}
      >
        {children}
      </a>
    )
  }
)
SkipLink.displayName = "SkipLink"

export { SkipLink }
