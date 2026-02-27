import React from "react";
import { type VariantProps, cva } from "class-variance-authority";
import { cx } from "@/ui/cx";

const statusBadgeVariants = cva(
  "inline-flex items-center rounded-md font-mono font-medium tracking-tight transition-colors",
  {
    variants: {
      status: {
        // Transaction statuses
        confirmed: "bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20",
        pending:   "bg-amber-500/10 text-amber-400 ring-1 ring-amber-500/20",
        failed:    "bg-red-500/10 text-red-400 ring-1 ring-red-500/20",
        open:      "bg-red-500/10 text-red-400 ring-1 ring-red-500/20",

        // Validator statuses
        staked:    "bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20",
        unstaking: "bg-amber-500/10 text-amber-400 ring-1 ring-amber-500/20",
        paused:    "bg-red-500/10 text-red-400 ring-1 ring-red-500/20",

        // Account statuses
        liquid:    "bg-muted/50 text-muted-foreground ring-1 ring-border/60",
        delegated: "bg-primary/10 text-primary ring-1 ring-primary/20",

        // Generic
        active:    "bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20",
        inactive:  "bg-muted/50 text-muted-foreground ring-1 ring-border/60",
        warning:   "bg-amber-500/10 text-amber-400 ring-1 ring-amber-500/20",
        error:     "bg-red-500/10 text-red-400 ring-1 ring-red-500/20",
        info:      "bg-blue-500/10 text-blue-400 ring-1 ring-blue-500/20",

        // Live indicator
        live:      "bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20",
      },
      size: {
        sm: "px-1.5 py-0.5 text-[10px]",
        md: "px-2 py-0.5 text-xs",
        lg: "px-2.5 py-1 text-xs",
      },
    },
    defaultVariants: {
      status: "inactive",
      size: "md",
    },
  }
);

// Map string status to variant
const statusMap: Record<string, VariantProps<typeof statusBadgeVariants>["status"]> = {
  // Case-insensitive mapping
  confirmed: "confirmed",
  pending: "pending",
  failed: "failed",
  open: "open",
  staked: "staked",
  unstaking: "unstaking",
  paused: "paused",
  liquid: "liquid",
  delegated: "delegated",
  active: "active",
  inactive: "inactive",
  live: "live",
};

export interface StatusBadgeProps
  extends Omit<React.HTMLAttributes<HTMLSpanElement>, "children">,
    VariantProps<typeof statusBadgeVariants> {
  /** Status text to display */
  label?: string;
  /** Show pulsing dot indicator */
  pulse?: boolean;
}

export const StatusBadge = React.memo<StatusBadgeProps>(({
  className,
  status,
  size,
  label,
  pulse = false,
  ...props
}) => {
  // Auto-detect status from label if not provided
  const resolvedStatus = status || statusMap[label?.toLowerCase() ?? ""] || "inactive";
  const displayLabel = label || (status ? status.toString() : "Unknown");

  return (
    <span
      className={cx(statusBadgeVariants({ status: resolvedStatus, size }), className)}
      {...props}
    >
      {pulse && (
        <span className="relative mr-1.5 flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-current opacity-75" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-current" />
        </span>
      )}
      {displayLabel.charAt(0).toUpperCase() + displayLabel.slice(1)}
    </span>
  );
});

StatusBadge.displayName = "StatusBadge";

export { statusBadgeVariants };
