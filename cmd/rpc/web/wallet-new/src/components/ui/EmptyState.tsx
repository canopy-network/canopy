import React from "react";
import { motion } from "framer-motion";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { cx } from "@/ui/cx";

export interface EmptyStateProps {
  /** Icon name from Lucide icons */
  icon?: string;
  /** Main title text */
  title: string;
  /** Optional description text */
  description?: string;
  /** Optional action button */
  action?: {
    label: string;
    onClick: () => void;
  };
  /** Additional CSS classes */
  className?: string;
  /** Size variant */
  size?: "sm" | "md" | "lg";
}

const sizeConfig = {
  sm: {
    icon: "w-8 h-8",
    iconBg: "w-12 h-12",
    title: "text-sm",
    description: "text-xs",
    padding: "py-6",
    gap: "gap-2",
  },
  md: {
    icon: "w-10 h-10",
    iconBg: "w-16 h-16",
    title: "text-base",
    description: "text-sm",
    padding: "py-8",
    gap: "gap-3",
  },
  lg: {
    icon: "w-12 h-12",
    iconBg: "w-20 h-20",
    title: "text-lg",
    description: "text-base",
    padding: "py-12",
    gap: "gap-4",
  },
};

export const EmptyState = React.memo<EmptyStateProps>(({
  icon = "Inbox",
  title,
  description,
  action,
  className,
  size = "md",
}) => {
  const config = sizeConfig[size];

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className={cx(
        "flex flex-col items-center justify-center text-center",
        config.padding,
        config.gap,
        className
      )}
    >
      <div
        className={cx(
          "flex items-center justify-center rounded-full bg-bg-accent/50",
          config.iconBg
        )}
      >
        <LucideIcon
          name={icon}
          className={cx("text-text-muted", config.icon)}
        />
      </div>

      <div className="space-y-1">
        <h4 className={cx("font-medium text-text-primary", config.title)}>
          {title}
        </h4>
        {description && (
          <p className={cx("text-text-muted max-w-xs", config.description)}>
            {description}
          </p>
        )}
      </div>

      {action && (
        <button
          onClick={action.onClick}
          className="mt-2 px-4 py-2 text-sm font-medium text-primary hover:text-primary/80
                     bg-primary/10 hover:bg-primary/20 rounded-lg transition-colors"
        >
          {action.label}
        </button>
      )}
    </motion.div>
  );
});

EmptyState.displayName = "EmptyState";
