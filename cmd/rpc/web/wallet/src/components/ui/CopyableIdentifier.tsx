import React from "react";
import { Copy } from "lucide-react";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

interface CopyableIdentifierProps {
  value: string;
  label?: string;
  children: React.ReactNode;
  className?: string;
  iconClassName?: string;
}

export const CopyableIdentifier: React.FC<CopyableIdentifierProps> = ({
  value,
  label = "Identifier",
  children,
  className = "",
  iconClassName = "",
}) => {
  const { copyToClipboard } = useCopyToClipboard();

  return (
    <button
      type="button"
      data-row-click-ignore="true"
      className={`group/copy inline-flex min-w-0 items-center gap-1.5 text-left transition-colors hover:text-primary ${className}`}
      title={`Copy ${label.toLowerCase()}: ${value}`}
      aria-label={`Copy ${label.toLowerCase()}`}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void copyToClipboard(value, label);
      }}
    >
      <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
        {children}
      </span>
      <Copy
        className={`h-3 w-3 shrink-0 text-white/35 opacity-0 transition-opacity group-hover/copy:opacity-100 group-focus-visible/copy:opacity-100 ${iconClassName}`}
        aria-hidden="true"
      />
    </button>
  );
};
