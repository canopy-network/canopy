import React from "react";
import { motion } from "framer-motion";
import { Loader2 } from "lucide-react";
import { cx } from "@/ui/cx";

export interface LoadingStateProps {
  /** Loading message */
  message?: string;
  /** Size variant */
  size?: "sm" | "md" | "lg";
  /** Additional CSS classes */
  className?: string;
  /** Show as overlay on existing content */
  overlay?: boolean;
}

const sizeConfig = {
  sm: {
    spinner: "w-5 h-5",
    text: "text-xs",
    padding: "py-4",
    gap: "gap-2",
  },
  md: {
    spinner: "w-8 h-8",
    text: "text-sm",
    padding: "py-8",
    gap: "gap-3",
  },
  lg: {
    spinner: "w-10 h-10",
    text: "text-base",
    padding: "py-12",
    gap: "gap-4",
  },
};

export const LoadingState = React.memo<LoadingStateProps>(({
  message = "Loading...",
  size = "md",
  className,
  overlay = false,
}) => {
  const config = sizeConfig[size];

  const content = (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.2 }}
      className={cx(
        "flex flex-col items-center justify-center",
        config.padding,
        config.gap,
        className
      )}
    >
      <Loader2 className={cx("text-primary animate-spin", config.spinner)} />
      {message && (
        <span className={cx("text-text-muted", config.text)}>{message}</span>
      )}
    </motion.div>
  );

  if (overlay) {
    return (
      <div className="absolute inset-0 flex items-center justify-center bg-bg-secondary/80 backdrop-blur-sm rounded-xl z-10">
        {content}
      </div>
    );
  }

  return content;
});

LoadingState.displayName = "LoadingState";

// Skeleton components for more sophisticated loading states
export const Skeleton = React.memo<{
  className?: string;
  animate?: boolean;
}>(({ className, animate = true }) => (
  <div
    className={cx(
      "bg-bg-accent rounded",
      animate && "animate-pulse",
      className
    )}
  />
));

Skeleton.displayName = "Skeleton";

export const SkeletonText = React.memo<{
  lines?: number;
  className?: string;
}>(({ lines = 1, className }) => (
  <div className={cx("space-y-2", className)}>
    {Array.from({ length: lines }).map((_, i) => (
      <Skeleton
        key={i}
        className={cx(
          "h-4",
          i === lines - 1 && lines > 1 ? "w-3/4" : "w-full"
        )}
      />
    ))}
  </div>
));

SkeletonText.displayName = "SkeletonText";

export const SkeletonCard = React.memo<{ className?: string }>(({ className }) => (
  <div
    className={cx(
      "bg-bg-secondary rounded-xl border border-bg-accent p-6 space-y-4",
      className
    )}
  >
    <div className="flex items-center gap-3">
      <Skeleton className="w-10 h-10 rounded-full" />
      <div className="flex-1 space-y-2">
        <Skeleton className="h-4 w-1/3" />
        <Skeleton className="h-3 w-1/2" />
      </div>
    </div>
    <SkeletonText lines={3} />
  </div>
));

SkeletonCard.displayName = "SkeletonCard";
