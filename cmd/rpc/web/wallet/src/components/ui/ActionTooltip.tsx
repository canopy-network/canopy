import React, { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

type ActionTooltipProps = {
  label: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
};

export const ActionTooltip: React.FC<ActionTooltipProps> = ({
  label,
  description,
  children,
  className = "",
}) => {
  const triggerRef = useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState<{ left: number; top: number; placement: "top" | "bottom" }>({
    left: 0,
    top: 0,
    placement: "top",
  });

  const updatePosition = useCallback(() => {
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;

    const estimatedTooltipWidth = 224;
    const estimatedTooltipHeight = description ? 64 : 42;
    const viewportPadding = 12;
    const topSpace = rect.top;
    const bottomSpace = window.innerHeight - rect.bottom;
    const placement =
      topSpace >= estimatedTooltipHeight + viewportPadding || topSpace >= bottomSpace
        ? "top"
        : "bottom";

    const centeredLeft = rect.left + rect.width / 2;
    const clampedLeft = Math.min(
      Math.max(centeredLeft, viewportPadding + estimatedTooltipWidth / 2),
      window.innerWidth - viewportPadding - estimatedTooltipWidth / 2,
    );

    setPosition({
      left: clampedLeft,
      top: placement === "top" ? rect.top - 8 : rect.bottom + 8,
      placement,
    });
  }, [description]);

  useEffect(() => {
    if (!isOpen) return;

    updatePosition();

    const handleViewportChange = () => updatePosition();

    window.addEventListener("resize", handleViewportChange);
    window.addEventListener("scroll", handleViewportChange, true);

    return () => {
      window.removeEventListener("resize", handleViewportChange);
      window.removeEventListener("scroll", handleViewportChange, true);
    };
  }, [isOpen, updatePosition]);

  return (
    <div
      ref={triggerRef}
      className={`inline-flex ${className}`}
      onMouseEnter={() => {
        updatePosition();
        setIsOpen(true);
      }}
      onMouseLeave={() => setIsOpen(false)}
      onFocusCapture={() => {
        updatePosition();
        setIsOpen(true);
      }}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setIsOpen(false);
        }
      }}
    >
      {children}
      {typeof document !== "undefined" && isOpen
        ? createPortal(
            <div
              className="pointer-events-none fixed z-[12000] w-max max-w-56"
              style={{
                left: position.left,
                top: position.top,
                transform:
                  position.placement === "top"
                    ? "translate(-50%, -100%)"
                    : "translate(-50%, 0)",
              }}
            >
              <div className="rounded-xl border border-[#272729] bg-[#171717] px-3 py-2 text-left shadow-[0_18px_40px_rgba(0,0,0,0.45)]">
                <div className="flex flex-col gap-0.5">
                  <span className="text-[11px] font-semibold leading-tight text-white">
                    {label}
                  </span>
                  {description ? (
                    <span className="text-[10px] leading-relaxed text-white/60">
                      {description}
                    </span>
                  ) : null}
                </div>
              </div>
            </div>,
            document.body,
          )
        : null}
    </div>
  );
};
