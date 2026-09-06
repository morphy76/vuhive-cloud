import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { AlertTriangle, CheckCircle2, Info, XCircle } from "lucide-react"

import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 transition-colors",
  {
    variants: {
      variant: {
        default:
          "bg-slate-900 text-slate-50 hover:bg-slate-900/80 dark:bg-slate-50 dark:text-slate-900 dark:hover:bg-slate-50/80",
        success:
          "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300",
        warning:
          "bg-amber-50 text-amber-700 dark:bg-amber-950/60 dark:text-amber-300",
        error:
          "bg-red-50 text-red-700 dark:bg-red-950/60 dark:text-red-300",
        info:
          "bg-brand-50 text-brand-700 dark:bg-brand-950/60 dark:text-brand-300",
        outline: "text-slate-950 dark:text-slate-50 border border-slate-200 dark:border-slate-800",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

const Badge = React.forwardRef<HTMLDivElement, BadgeProps>(
  ({ className, variant, children, ...props }, ref) => {
    return (
      <div ref={ref} className={cn(badgeVariants({ variant }), className)} {...props}>
        {variant === "success" && <CheckCircle2 className="h-3.5 w-3.5" />}
        {variant === "warning" && <AlertTriangle className="h-3.5 w-3.5" />}
        {variant === "error" && <XCircle className="h-3.5 w-3.5" />}
        {variant === "info" && <Info className="h-3.5 w-3.5" />}
        {children}
      </div>
    )
  }
)
Badge.displayName = "Badge"

export { Badge, badgeVariants }
