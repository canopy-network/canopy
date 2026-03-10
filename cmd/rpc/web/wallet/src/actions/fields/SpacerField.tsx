import React from "react";
import { cx } from "@/ui/cx";

type SpacerFieldProps = {
  field: any;
};

/**
 * Spacer field type - Empty space for layout
 *
 * Schema:
 * {
 *   "type": "spacer",
 *   "height": "sm" | "md" | "lg" | "xl" | "2xl",
 *   "span": { "base": 12 }
 * }
 */
export const SpacerField: React.FC<SpacerFieldProps> = ({ field }) => {
  const height = field.height || "md";
  const span = field.span?.base ?? 12;

  const heightStyles: Record<string, string> = {
    sm: "h-2",
    md: "h-4",
    lg: "h-6",
    xl: "h-8",
    "2xl": "h-12",
  };

  return <div className={cx(`col-span-${span}`, heightStyles[height])} />;
};
