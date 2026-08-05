"use client";

import { useEffect, useState } from "react";
import { queryWaterfallEvents, type WaterfallEvent } from "@/lib/canopy/rpc";
import { formatAmount } from "@/lib/arbor/format";

const WATERFALL_META: Record<string, { label: string; layer: string; tone: string }> = {
  EventReserveFundDrawDown: { label: "Reserve fund draw-down", layer: "Layer 2", tone: "text-amber-300" },
  EventTreasuryDrawDown: { label: "Treasury draw-down", layer: "Layer 3", tone: "text-amber-300" },
  EventBadDebtSocialization: { label: "Bad-debt socialization", layer: "Layer 4", tone: "text-rose-300" },
  EventLossFactorExhausted: { label: "Loss factor exhausted", layer: "Layer 4", tone: "text-rose-300" },
  EventLossFactorAppliedToAlreadyInsolventMarket: { label: "Loss applied (already insolvent)", layer: "Layer 4", tone: "text-rose-300" },
  EventLayer4PendingCountWarning: { label: "Layer-4 backlog warning", layer: "Layer 4", tone: "text-rose-300" },
  EventInsolventMarketValueRecovered: { label: "Insolvent value recovered", layer: "R_fund", tone: "text-emerald-300" },
};

export function WaterfallPanel() {
  const [data, setData] = useState<{ events: WaterfallEvent[]; available: boolean } | null>(null);
  useEffect(() => {
    let alive = true;
    queryWaterfallEvents(1, 50).then((d) => { if (alive) setData(d); });
    return () => { alive = false; };
  }, []);

  return (
    <section className="space-y-3">
      <h2 className="section-h">Waterfall activity (Layer 2 / 3 / 4)</h2>
      {!data ? (
        <p className="text-sm text-zinc-500">Loading…</p>
      ) : !data.available ? (
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
          Discrete bad-debt waterfall events are emitted on-chain but this node
          exposes no query route for them yet. Awaiting
          <code className="mx-1 text-zinc-200">/v1/query/waterfall-events</code>
          (plugin-persisted rolling log, range-scanned like all-markets). This
          panel populates automatically once it lands.
        </div>
      ) : data.events.length === 0 ? (
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
          No waterfall events recorded — no Layer 2/3/4 draw-downs have fired yet.
        </div>
      ) : (
        <div className="space-y-2">
          {data.events.map((e, i) => {
            const m = WATERFALL_META[e.eventType] ?? { label: e.eventType, layer: "—", tone: "text-zinc-300" };
            return (
              <div key={i} className="rounded-xl border border-white/5 bg-white/[0.02] px-4 py-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className={`text-xs font-medium ${m.tone}`}>{m.layer}</span>
                  <span className="text-sm text-zinc-100">{m.label}</span>
                  {e.marketId && <span className="font-mono text-xs text-zinc-500">{e.marketId}</span>}
                  {e.height ? <span className="ml-auto text-xs tabular-nums text-zinc-600">#{Number(e.height).toLocaleString()}</span> : null}
                </div>
                <div className="mt-1 flex flex-wrap gap-x-5 gap-y-1 text-xs tabular-nums text-zinc-400">
                  {e.badDebt != null && <span>bad debt {formatAmount(BigInt(e.badDebt), 9)}</span>}
                  {e.remainingReserveFund != null && <span>R_fund after {formatAmount(BigInt(e.remainingReserveFund), 9)}</span>}
                  {e.remainingTreasury != null && <span>T_fund after {formatAmount(BigInt(e.remainingTreasury), 9)}{e.pool ? ` (${e.pool})` : ""}</span>}
                  {e.newLossFactor != null && <span>new loss factor {(Number(BigInt(e.newLossFactor)) / 1e18).toFixed(6)}</span>}
                  {e.recoveredAmount != null && <span>recovered {formatAmount(BigInt(e.recoveredAmount), 9)}{e.source ? ` (${e.source})` : ""}</span>}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
