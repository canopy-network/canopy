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
  const variantStyles: Record<string, { bg: string; border: string; text: string; icon: string; description: string }> = {
    default: {
      bg: "bg-card/50",
      border: "border-border",
      text: "text-foreground",
      icon: "text-muted-foreground",
      description: "text-muted-foreground",
    },
    info: {
      bg: "bg-[#216cd0]/10",
      border: "border-[#216cd0]/25",
      text: "text-[#216cd0]",
      icon: "text-[#216cd0]",
      description: "text-[#216cd0]",
    },
    warning: {
      bg: "bg-[#ddb228]/10",
      border: "border-[#ddb228]/25",
      text: "text-[#ddb228]",
      icon: "text-[#ddb228]",
      description: "text-[#ddb228]",
    },
    success: {
      bg: "bg-[#35cd48]/10",
      border: "border-[#35cd48]/25",
      text: "text-[#35cd48]",
      icon: "text-[#35cd48]",
      description: "text-[#35cd48]",
    },
    error: {
      bg: "bg-[#ff1845]/10",
      border: "border-[#ff1845]/25",
      text: "text-[#ff1845]",
      icon: "text-[#ff1845]",
      description: "text-[#ff1845]",
    },
    primary: {
      bg: "bg-white/[0.06]",
      border: "border-white/12",
      text: "text-foreground",
      icon: "text-foreground/70",
      description: "text-muted-foreground",
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
              <p className={cx("mt-1 text-xs", styles.description)}>{description}</p>
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
