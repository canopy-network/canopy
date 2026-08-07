"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { queryWaterfallEvents, queryHeight, type WaterfallEvent } from "@/lib/canopy/rpc";
import { formatAmount } from "@/lib/arbor/format";

const LAYER_META: Record<string, { label: string; badge: string; edge: string }> = {
  layer2: { label: "Layer 2 · R_fund", badge: "bg-amber-500/10 text-amber-300 ring-amber-500/30", edge: "border-l-amber-400/70" },
  layer3: { label: "Layer 3 · T_fund", badge: "bg-orange-500/10 text-orange-300 ring-orange-500/30", edge: "border-l-orange-400/70" },
  layer4: { label: "Layer 4 · socialized", badge: "bg-rose-500/10 text-rose-300 ring-rose-500/30", edge: "border-l-rose-400/70" },
  nasm: { label: "NASM · R_nusd", badge: "bg-sky-500/10 text-sky-300 ring-sky-500/30", edge: "border-l-sky-400/70" },
};

const EVENT_LABEL: Record<string, string> = {
  EventReserveFundDrawDown: "Reserve fund draw-down",
  EventTreasuryDrawDown: "Treasury draw-down",
  EventBadDebtSocialization: "Bad-debt socialization",
  EventLossFactorExhausted: "Loss factor exhausted → insolvent",
  EventLossFactorAppliedToAlreadyInsolventMarket: "Haircut on already-insolvent market",
  EventLayer4PendingCountWarning: "Layer-4 backlog warning",
  EventLayer4PendingBadDebtTotalSaturated: "Layer-4 counter saturated",
  EventLayer4PendingCountUnderflow: "Layer-4 counter underflow",
  EventDepositWithdrawBlockedDuringPendingLoss: "Deposit/withdraw blocked (pending loss)",
  EventInsolventMarketValueRecovered: "Recovered value → R_fund",
  EventNasmVaultLiquidated: "NASM vault liquidated",
  EventIndexEncodingOverflowHalted: "Index encoding overflow halted",
};

// Per-event semantics: the plugin's normalized log carries a generic
// remainingBalance whose meaning (and scale) depends on the event type.
function metrics(e: WaterfallEvent): { label: string; value: string; tone?: string }[] {
  const out: { label: string; value: string; tone?: string }[] = [];
  if (e.badDebt != null) out.push({ label: "bad debt", value: formatAmount(BigInt(e.badDebt), 9) });
  switch (e.eventType) {
    case "EventBadDebtSocialization":
    case "EventLossFactorAppliedToAlreadyInsolventMarket":
      // remainingBalance = new loss factor, RAY (1e18) scale
      if (e.remainingBalance != null)
        out.push({ label: "new loss factor", value: (Number(BigInt(e.remainingBalance)) / 1e18).toFixed(6), tone: "text-rose-300" });
      break;
    case "EventLossFactorExhausted":
      if (e.remainingBalance != null) out.push({ label: "total supplied equiv", value: formatAmount(BigInt(e.remainingBalance), 9) });
      break;
    case "EventReserveFundDrawDown":
      if (e.remainingBalance != null) out.push({ label: "R_fund after", value: formatAmount(BigInt(e.remainingBalance), 9) });
      break;
    case "EventTreasuryDrawDown":
      if (e.remainingBalance != null) out.push({ label: `T_fund after${e.pool ? ` (${e.pool})` : ""}`, value: formatAmount(BigInt(e.remainingBalance), 9) });
      break;
    case "EventNasmVaultLiquidated":
      if (e.remainingBalance != null) out.push({ label: "seized collateral", value: formatAmount(BigInt(e.remainingBalance), 9) });
      break;
    default:
      if (e.remainingBalance != null) out.push({ label: "remaining", value: formatAmount(BigInt(e.remainingBalance), 9) });
  }
  return out;
}

function ageLabel(current: number | null, h: number): string {
  if (current == null || current <= h) return "";
  const secs = (current - h) * 5; // ~5s blocks
  if (secs < 90) return `≈${Math.max(1, Math.round(secs))}s ago`;
  if (secs < 3600) return `≈${Math.round(secs / 60)}m ago`;
  if (secs < 86400) return `≈${Math.floor(secs / 3600)}h ${Math.floor((secs % 3600) / 60)}m ago`;
  return `≈${Math.floor(secs / 86400)}d ago`;
}

export function WaterfallPanel() {
  const [data, setData] = useState<{ events: WaterfallEvent[]; available: boolean } | null>(null);
  const [chainHeight, setChainHeight] = useState<number | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    setRefreshing(true);
    const [d, h] = await Promise.all([queryWaterfallEvents(50), queryHeight()]);
    setData(d);
    setChainHeight(h);
    setRefreshing(false);
  }, []);

  useEffect(() => {
    let alive = true;
    (async () => {
      const [d, h] = await Promise.all([queryWaterfallEvents(50), queryHeight()]);
      if (alive) { setData(d); setChainHeight(h); }
    })();
    return () => { alive = false; };
  }, []);

  const events = data?.events ?? [];
  const markets = new Set(events.map((e) => e.marketId).filter(Boolean));
  const latest = events.length ? events[0].height ?? 0 : 0;

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center gap-3">
        <div>
          <h2 className="section-h">Waterfall activity</h2>
          <p className="text-xs text-zinc-500">
            Bad-debt absorption across R_fund → T_fund → lenders (ARCM §9.2), read live from the {`{42}`} durable log.
          </p>
        </div>
        <button
          type="button"
          onClick={load}
          disabled={refreshing}
          className="ml-auto inline-flex items-center gap-1.5 rounded-lg border border-white/10 px-2.5 py-1.5 text-xs text-zinc-400 transition hover:bg-white/5 hover:text-zinc-200 disabled:opacity-50"
        >
          <svg viewBox="0 0 20 20" fill="currentColor" className={`h-3.5 w-3.5 ${refreshing ? "animate-spin" : ""}`}>
            <path fillRule="evenodd" d="M15.312 11.424a5.5 5.5 0 0 1-9.201 2.466l-.312-.311h2.433a.75.75 0 0 0 0-1.5H3.989a.75.75 0 0 0-.75.75v4.242a.75.75 0 0 0 1.5 0v-2.43l.31.31a7 7 0 0 0 11.712-3.138.75.75 0 0 0-1.449-.39Zm1.23-3.723a.75.75 0 0 0 .219-.53V2.929a.75.75 0 0 0-1.5 0V5.36l-.31-.31A7 7 0 0 0 3.239 8.188a.75.75 0 1 0 1.448.389 5.5 5.5 0 0 1 9.201-2.466l.312.311H11.767a.75.75 0 0 0 0 1.5h4.243a.75.75 0 0 0 .531-.221Z" clipRule="evenodd" />
          </svg>
          {refreshing ? "Refreshing…" : "Refresh"}
        </button>
      </div>

      {!data ? (
        <div className="space-y-2">
          {[0, 1].map((i) => (
            <div key={i} className="h-16 animate-pulse rounded-xl border border-white/5 bg-white/[0.03]" />
          ))}
        </div>
      ) : !data.available ? (
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
          Discrete bad-debt waterfall events are emitted on-chain but this node
          exposes no query route for them yet. Awaiting
          <code className="mx-1 text-zinc-200">/v1/query/waterfall-events</code>
          (plugin-persisted rolling log, range-scanned like all-markets). This
          panel populates automatically once it lands.
        </div>
      ) : events.length === 0 ? (
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
          No waterfall events recorded — no Layer 2/3/4 draw-downs have fired yet.
        </div>
      ) : (
        <>
          <div className="flex flex-wrap gap-2 text-[11px] tabular-nums">
            <span className="rounded-full border border-white/10 bg-white/[0.03] px-2.5 py-1 text-zinc-400">
              {events.length} event{events.length === 1 ? "" : "s"}
            </span>
            <span className="rounded-full border border-white/10 bg-white/[0.03] px-2.5 py-1 text-zinc-400">
              {markets.size} market{markets.size === 1 ? "" : "s"} affected
            </span>
            <span className="rounded-full border border-white/10 bg-white/[0.03] px-2.5 py-1 text-zinc-400">
              latest #{latest.toLocaleString()} {ageLabel(chainHeight, latest)}
            </span>
          </div>

          <div className="space-y-2">
            {events.map((e, i) => {
              const layer = LAYER_META[e.layer ?? ""] ?? LAYER_META.layer4;
              return (
                <div
                  key={`${e.height}-${e.marketId}-${i}`}
                  className={`rounded-xl border border-white/5 border-l-2 ${layer.edge} bg-white/[0.02] px-4 py-3 transition hover:bg-white/[0.04]`}
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={`rounded-md px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide ring-1 ${layer.badge}`}>
                      {layer.label}
                    </span>
                    <span className="text-sm font-medium text-zinc-100">
                      {EVENT_LABEL[e.eventType] ?? e.eventType}
                    </span>
                    {e.marketId && (
                      <Link
                        href={`/markets/${e.marketId}`}
                        className="font-mono text-xs text-zinc-500 transition hover:text-indigo-300"
                      >
                        {e.marketId}
                      </Link>
                    )}
                    <span className="ml-auto text-xs tabular-nums text-zinc-600">
                      #{(e.height ?? 0).toLocaleString()}
                      {ageLabel(chainHeight, e.height ?? 0) && (
                        <span className="ml-1.5 text-zinc-700">{ageLabel(chainHeight, e.height ?? 0)}</span>
                      )}
                    </span>
                  </div>
                  <div className="mt-1.5 flex flex-wrap gap-x-6 gap-y-1 text-xs tabular-nums">
                    {metrics(e).map((m) => (
                      <span key={m.label} className="text-zinc-500">
                        {m.label}{" "}
                        <span className={m.tone ?? "text-zinc-300"}>{m.value}</span>
                      </span>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>

          <p className="text-[11px] text-zinc-600">
            Source: /v1/query/waterfall-events · most-recent-first · capped at 50 ·
            amounts in 9-decimal native units, loss factors in RAY.
          </p>
        </>
      )}
    </section>
  );
}
