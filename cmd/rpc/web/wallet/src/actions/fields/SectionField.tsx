import React from "react";
import { motion } from "framer-motion";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { cx } from "@/ui/cx";

type SectionFieldProps = {
  field: any;
  templateContext: any;
  resolveTemplate: (s?: any) => any;
};

/**
 * Section field type - Visual grouping component
 *
 * Schema:
 * {
 *   "type": "section",
 *   "title": "Section Title",
 *   "description": "Optional description text",
 *   "icon": "Settings", // Optional Lucide icon name
 *   "variant": "default" | "info" | "warning" | "success" | "error" | "primary",
 *   "collapsible": false,
 *   "defaultCollapsed": false,
 *   "span": { "base": 12 }
 * }
 */
export const SectionField: React.FC<SectionFieldProps> = ({
  field,
  resolveTemplate,
}) => {
  const title = resolveTemplate(field.title);
  const description = resolveTemplate(field.description);
  const icon = field.icon;
  const variant = field.variant || "default";
  const collapsible = field.collapsible || false;
  const [collapsed, setCollapsed] = React.useState(field.defaultCollapsed || false);

  const span = field.span?.base ?? 12;

  // Variant styling
  const variantStyles: Record<string, { bg: string; border: string; text: string; icon: string }> = {
    default: {
      bg: "bg-card/50",
      border: "border-border",
      text: "text-foreground",
      icon: "text-muted-foreground",
    },
    info: {
      bg: "bg-blue-950/30",
      border: "border-blue-800/30",
      text: "text-blue-100",
      icon: "text-blue-400",
    },
    warning: {
      bg: "bg-yellow-950/30",
      border: "border-yellow-800/30",
      text: "text-yellow-100",
      icon: "text-yellow-400",
    },
    success: {
      bg: "bg-emerald-950/30",
      border: "border-emerald-800/30",
      text: "text-emerald-100",
      icon: "text-emerald-400",
    },
    error: {
      bg: "bg-red-950/30",
      border: "border-red-800/30",
      text: "text-red-100",
      icon: "text-red-400",
    },
    primary: {
      bg: "bg-primary/10",
      border: "border-primary/30",
      text: "text-primary-foreground",
      icon: "text-primary",
    },
  };

  const styles = variantStyles[variant] || variantStyles.default;

  return (
    <motion.div
      className={cx(`col-span-${span}`, "mb-2")}
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
    >
      <div className={cx("rounded-lg border p-4", styles.bg, styles.border)}>
        <div
          className={cx(
            "flex items-center gap-3",
            collapsible && "cursor-pointer",
          )}
          onClick={() => collapsible && setCollapsed(!collapsed)}
        >
          {icon && (
            <div className={cx("flex-shrink-0", styles.icon)}>
              <LucideIcon name={icon} className="w-5 h-5" />
            </div>
          )}
          <div className="flex-1">
            {title && (
              <h4 className={cx("text-sm font-semibold", styles.text)}>
                {title}
              </h4>
            )}
            {description && !collapsed && (
              <p className="text-xs text-muted-foreground mt-1">{description}</p>
            )}
          </div>
          {collapsible && (
            <div className={styles.icon}>
              <LucideIcon
                name={collapsed ? "ChevronDown" : "ChevronUp"}
                className="w-4 h-4"
              />
            </div>
          )}
        </div>
      </div>
    </motion.div>
  );
};
