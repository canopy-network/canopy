import { useState, ReactNode } from "react";

export function Tooltip({ label, children }: { label: string; children: ReactNode }) {
  const [show, setShow] = useState(false);
  return (
    <span className="relative inline-flex items-center gap-1">
      {children}
      <button
        type="button"
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onClick={() => setShow((s) => !s)}
        className="inline-flex h-3.5 w-3.5 items-center justify-center rounded-full bg-zinc-700 text-[9px] font-bold text-zinc-300 hover:bg-zinc-600"
        aria-label={label}
      >
        ?
      </button>
      {show && (
        <span className="absolute bottom-full left-1/2 z-50 mb-2 w-48 -translate-x-1/2 rounded-md bg-zinc-800 px-2 py-1 text-xs text-zinc-200 shadow-lg">
          {label}
        </span>
      )}
    </span>
  );
}
