"use client";

import type { MarketStatus } from "@/lib/arbor/types";

const STYLES: Record<MarketStatus, string> = {
  ACTIVE:
    "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  PAUSED:
    "border-amber-500/30 bg-amber-500/10 text-amber-300",
  DEPRECATED:
    "border-rose-500/30 bg-rose-500/10 text-rose-300",
  INSOLVENT:
    "border-rose-500/30 bg-rose-500/10 text-rose-300",
};

export function StatusPill({ status }: { status: MarketStatus }) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${STYLES[status]}`}
    >
      {status}
    </span>
  );
}
