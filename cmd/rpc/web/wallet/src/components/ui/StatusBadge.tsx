import React from "react";
import { type VariantProps, cva } from "class-variance-authority";
import { cx } from "@/ui/cx";
import { WALLET_BADGE_CLASS } from "@/components/ui/badgeStyles";

const statusBadgeVariants = cva(
  `${WALLET_BADGE_CLASS} transition-colors`,
  {
    variants: {
      status: {
        confirmed: "",
        pending: "",
        failed: "",
        open: "",
        staked: "",
        unstaking: "",
        paused: "",
        liquid: "",
        delegated: "",
        active: "",
        inactive: "",
        warning: "",
        error: "",
        info: "",
        live: "",
      },
      size: {
        sm: "",
        md: "",
        lg: "",
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
