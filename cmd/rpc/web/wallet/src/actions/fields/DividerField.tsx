import React from "react";
import { cx } from "@/ui/cx";

type DividerFieldProps = {
  field: any;
  resolveTemplate: (s?: any) => any;
};

/**
 * Divider field type - Horizontal separator
 *
 * Schema:
 * {
 *   "type": "divider",
 *   "label": "Optional label text",
 *   "variant": "solid" | "dashed" | "dotted" | "gradient",
 *   "spacing": "sm" | "md" | "lg", // Vertical spacing
 *   "span": { "base": 12 }
 * }
 */
export const DividerField: React.FC<DividerFieldProps> = ({
  field,
  resolveTemplate,
}) => {
  const label = resolveTemplate(field.label);
  const variant = field.variant || "solid";
  const spacing = field.spacing || "md";
  const span = field.span?.base ?? 12;

  const spacingStyles: Record<string, string> = {
    sm: "my-2",
    md: "my-4",
    lg: "my-6",
  };

  const variantStyles: Record<string, string> = {
    solid: "border-t border-border",
    dashed: "border-t border-dashed border-border",
    dotted: "border-t border-dotted border-border",
    gradient:
      "h-px bg-gradient-to-r from-transparent via-bg-accent to-transparent",
  };

  return (
    <div className={cx(`col-span-${span}`, spacingStyles[spacing])}>
      {label ? (
        <div className="relative">
          <div className={variantStyles[variant]} />
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="bg-background px-3 text-xs text-muted-foreground uppercase tracking-wider">
              {label}
            </span>
          </div>
        </div>
      ) : (
        <div className={variantStyles[variant]} />
      )}
    </div>
  );
};
