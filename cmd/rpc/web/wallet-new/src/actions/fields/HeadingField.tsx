import React from "react";
import { motion } from "framer-motion";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { cx } from "@/ui/cx";

type HeadingFieldProps = {
  field: any;
  templateContext: any;
  resolveTemplate: (s?: any) => any;
};

/**
 * Heading field type - Text headings/titles
 *
 * Schema:
 * {
 *   "type": "heading",
 *   "text": "Heading text",
 *   "level": 1 | 2 | 3 | 4, // h1, h2, h3, h4
 *   "icon": "Settings", // Optional Lucide icon
 *   "align": "left" | "center" | "right",
 *   "color": "primary" | "secondary" | "muted" | "accent",
 *   "span": { "base": 12 }
 * }
 */
export const HeadingField: React.FC<HeadingFieldProps> = ({
  field,
  resolveTemplate,
}) => {
  const text = resolveTemplate(field.text);
  const level = field.level || 2;
  const icon = field.icon;
  const align = field.align || "left";
  const color = field.color || "primary";
  const span = field.span?.base ?? 12;

  const HeadingTag = `h${level}` as keyof JSX.IntrinsicElements;

  const sizeStyles: Record<number, string> = {
    1: "text-2xl font-bold",
    2: "text-xl font-semibold",
    3: "text-lg font-semibold",
    4: "text-base font-medium",
  };

  const colorStyles: Record<string, string> = {
    primary: "text-foreground",
    secondary: "text-text-secondary",
    muted: "text-muted-foreground",
    accent: "text-primary",
  };

  const alignStyles: Record<string, string> = {
    left: "justify-start text-left",
    center: "justify-center text-center",
    right: "justify-end text-right",
  };

  return (
    <motion.div
      className={cx(`col-span-${span}`, "mb-3")}
      initial={{ opacity: 0, y: -5 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <div className={cx("flex items-center gap-2", alignStyles[align])}>
        {icon && (
          <LucideIcon
            name={icon}
            className={cx("w-5 h-5", colorStyles[color])}
          />
        )}
        <HeadingTag className={cx(sizeStyles[level], colorStyles[color])}>
          {text}
        </HeadingTag>
      </div>
    </motion.div>
  );
};
