import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { cx } from "@/ui/cx";

type CollapsibleGroupFieldProps = {
  field: any;
  value: any;
  templateContext: any;
  resolveTemplate: (s?: any) => any;
  onChange: (value: any) => void;
};

/**
 * CollapsibleGroup field type - Toggleable advanced options section
 *
 * This field stores its collapsed state in the form as a boolean.
 * Use with showIf on child fields to control their visibility.
 *
 * Schema:
 * {
 *   "type": "collapsibleGroup",
 *   "id": "showAdvancedCommittees",
 *   "name": "showAdvancedCommittees",
 *   "title": "Advanced Options",
 *   "description": "Click to show advanced configuration",
 *   "icon": "Settings",
 *   "variant": "default" | "primary",
 *   "defaultExpanded": false,
 *   "span": { "base": 12 }
 * }
 *
 * Then in child fields:
 * {
 *   "showIf": "{{ form.showAdvancedCommittees }}"
 * }
 */
export const CollapsibleGroupField: React.FC<CollapsibleGroupFieldProps> = ({
  field,
  value,
  resolveTemplate,
  onChange,
}) => {
  const title = resolveTemplate(field.title) || "Advanced Options";
  const description = resolveTemplate(field.description);
  const icon = field.icon || "Settings";
  const variant = field.variant || "default";

  // Use form value for expanded state, default to field.defaultExpanded
  const isExpanded = value === true || value === "true";

  // Initialize with defaultExpanded if value is undefined
  React.useEffect(() => {
    if (value === undefined && field.defaultExpanded) {
      onChange(true);
    }
  }, []);

  const handleToggle = () => {
    onChange(!isExpanded);
  };

  const span = field.span?.base ?? 12;

  // Variant styling
  const variantStyles: Record<string, { bg: string; border: string; text: string; icon: string; hover: string }> = {
    default: {
      bg: "bg-bg-secondary/30",
      border: "border-bg-accent/50",
      text: "text-text-muted",
      icon: "text-text-muted",
      hover: "hover:bg-bg-secondary/50 hover:border-bg-accent",
    },
    primary: {
      bg: "bg-primary/5",
      border: "border-primary/20",
      text: "text-primary/80",
      icon: "text-primary/60",
      hover: "hover:bg-primary/10 hover:border-primary/30",
    },
  };

  const styles = variantStyles[variant] || variantStyles.default;

  return (
    <motion.div
      className={cx(`col-span-${span}`)}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.2 }}
    >
      <button
        type="button"
        onClick={handleToggle}
        className={cx(
          "w-full rounded-lg border px-4 py-3 transition-all duration-200",
          "flex items-center gap-3 cursor-pointer",
          styles.bg,
          styles.border,
          styles.hover
        )}
      >
        <div className={cx("flex-shrink-0", styles.icon)}>
          <LucideIcon name={icon} className="w-4 h-4" />
        </div>
        <div className="flex-1 text-left">
          <span className={cx("text-sm font-medium", styles.text)}>
            {title}
          </span>
          {description && !isExpanded && (
            <span className="text-xs text-text-muted ml-2">
              {description}
            </span>
          )}
        </div>
        <div className={styles.icon}>
          <motion.div
            animate={{ rotate: isExpanded ? 180 : 0 }}
            transition={{ duration: 0.2 }}
          >
            <LucideIcon name="ChevronDown" className="w-4 h-4" />
          </motion.div>
        </div>
      </button>
    </motion.div>
  );
};
