"use client";

import { useCallback, useEffect, useState } from "react";
import { useMarkets, type MarketWithIndices } from "@/lib/hooks/useMarkets";
import { useMarketPools } from "@/lib/hooks/useMarketPools";
import { formatAmount } from "@/lib/arbor/format";
import { getTreasury } from "@/lib/canopy/pluginRpc";
import { RAY } from "@/lib/arbor/constants";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";

function Flag({ bad }: { bad: boolean }) {
  return bad ? (
    <span className="inline-flex items-center gap-1 text-rose-300">
      <span className="h-1.5 w-1.5 rounded-full bg-rose-400" />
      Yes
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 text-zinc-500">
      <span className="h-1.5 w-1.5 rounded-full bg-zinc-600" />
      No
    </span>
  );
}

function LayerCard({
  layer,
  title,
  desc,
  value,
  sub,
  tone,
}: {
  layer: string;
  title: string;
  desc: string;
  value: string;
  sub?: string;
  tone: "emerald" | "indigo" | "zinc" | "rose";
}) {
  const ring =
    tone === "emerald"
      ? "border-emerald-500/30 bg-emerald-500/[0.06]"
      : tone === "indigo"
        ? "border-indigo-500/30 bg-indigo-500/[0.06]"
        : tone === "rose"
          ? "border-rose-500/30 bg-rose-500/[0.06]"
          : "border-white/10 bg-white/[0.03]";
  return (
    <div className={`rounded-2xl border p-5 backdrop-blur ${ring}`}>
      <p className="text-[11px] font-semibold uppercase tracking-wide text-zinc-500">
        {layer}
      </p>
      <p className="mt-1 text-sm font-semibold text-zinc-100">{title}</p>
      <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
        {value}
      </p>
      {sub && <p className="mt-1 text-[11px] text-zinc-500">{sub}</p>}
      <p className="mt-1 text-[11px] text-zinc-500">{desc}</p>
    </div>
  );
}

function MarketRow({
  entry,
  onCollateral,
  onUsd,
}: {
  entry: MarketWithIndices;
  onCollateral: (id: string, v: bigint) => void;
  onUsd: (id: string, suppliedUsd: number, rfUsd: number, collateralUsd: number) => void;
}) {
  const m = entry.market;
  const { data: pools } = useMarketPools(m.marketId);
  const collateral = pools?.collateral ?? 0n;
  const supply = pools?.supply ?? 0n;

  useEffect(() => {
    onCollateral(m.marketId, collateral);
    return () => onCollateral(m.marketId, 0n);
  }, [onCollateral, m.marketId, collateral]);

  const { data: priceData } = useAssetPrice(m.debtAssetId);
  const unitPrice =
    priceData?.available && priceData.price != null
      ? Number(priceData.price) / 1e8
      : null;

  const { data: collPriceData } = useAssetPrice(m.collateralAssetId);
  const collUnitPrice =
    collPriceData?.available && collPriceData.price != null
      ? Number(collPriceData.price) / 1e8
      : null;

  useEffect(() => {
    onUsd(
      m.marketId,
      unitPrice !== null ? Number(m.totalSupplied) * unitPrice : 0,
      unitPrice !== null ? Number(entry.reserveFund) * unitPrice : 0,
      collUnitPrice !== null ? Number(collateral) * collUnitPrice : 0
    );
    return () => onUsd(m.marketId, 0, 0, 0);
  }, [onUsd, m.marketId, m.totalSupplied, entry.reserveFund, unitPrice, collateral, collUnitPrice]);

  
function renderLossFactor(entry: { lossFactor: bigint; lossFactorAbsent: boolean }, status: string) {
  const RAY = 1000000000000000000n;
  if (entry.lossFactorAbsent) {
    return <span className="text-zinc-600" title="loss_factor not yet initialized (market exists but never hair-cut)">—</span>;
  }
  if (entry.lossFactor === 0n) {
    if (status === "INSOLVENT") {
      return <span className="font-medium text-rose-300" title="Fully exhausted (Layer 4 ran to completion; I11 satisfied)">exhausted</span>;
    }
    return <span className="font-medium text-rose-300" title="Fully exhausted but status != INSOLVENT — possible I11 violation">exhausted <span aria-hidden="true">⚠</span></span>;
  }
  if (entry.lossFactor === RAY) {
    return <span className="tabular-nums text-zinc-300">1.000000</span>;
  }
  const v = Number(entry.lossFactor) / 1e18;
  return <span className="tabular-nums text-zinc-300" title="Partial haircut applied">{v.toFixed(6)}</span>;
}

  const tvl = m.totalSupplied;
  const rfRatio = tvl > 0n ? Number((entry.reserveFund * 10000n) / tvl) / 100 : 0;

  return (
    <tr className="border-t border-white/5">
      <td className="py-3 pr-4 text-xs font-medium text-zinc-200">{m.marketId}</td>
      <td className="py-3 text-center"><Flag bad={m.status === "INSOLVENT"} /></td>
      <td className="py-3 text-center"><Flag bad={m.indexOverflowHalted} /></td>
      <td className="py-3 text-center"><Flag bad={m.layer4PendingCount > 0} /></td>
      <td className="py-3 text-center"><Flag bad={m.status === "PAUSED"} /></td>
      <td className="py-3 text-center"><Flag bad={m.status === "DEPRECATED"} /></td>
      <td className="py-3 pr-4 text-right text-xs tabular-nums text-zinc-300">
        {formatAmount(supply, 0)}
      </td>
      <td className="py-3 pr-4 text-right text-xs tabular-nums text-zinc-300">
        {formatAmount(collateral, 0)}
      </td>
      <td className="py-3 pr-4 text-right text-xs tabular-nums text-zinc-300">
        {formatAmount(entry.reserveFund, 0)}
      </td>
      <td className="py-3 pr-4 text-right">{renderLossFactor(entry, m.status)}</td>
      <td className="py-3 text-right text-xs tabular-nums text-zinc-300">
        {rfRatio.toFixed(1)}%
      </td>
    </tr>
  );
}

export default function MonitorPage() {
  const { data: markets, isLoading } = useMarkets();
  const list = markets ?? [];

  const [collateralByMarket, setCollateralByMarket] = useState<
    Record<string, bigint>
  >({});
  const onCollateral = useCallback(
    (id: string, v: bigint) => setCollateralByMarket((p) => ({ ...p, [id]: v })),
    []
  );

  const [usdByMarket, setUsdByMarket] = useState<
    Record<string, { s: number; rf: number; c: number }>
  >({});
  const onUsd = useCallback(
    (id: string, s: number, rf: number, coll: number) =>
      setUsdByMarket((p) => ({ ...p, [id]: { s, rf, c: coll } })),
    []
  );
  const totalTvlUsd = Object.values(usdByMarket).reduce((a, v) => a + v.s, 0);
  const totalRfUsd = Object.values(usdByMarket).reduce((a, v) => a + v.rf, 0);
  const totalCollateralUsd = Object.values(usdByMarket).reduce((a, v) => a + v.c, 0);
  const collateralBreakdown = list
    .map((e) => ({
      v: collateralByMarket[e.market.marketId] ?? 0n,
      a: e.market.collateralAssetId,
    }))
    .filter((x) => x.v > 0n)
    .map((x) => `${formatAmount(x.v, 0)} ${x.a}`)
    .join(" \u00b7 ");

  const totalRf = list.reduce((a, e) => a + e.reserveFund, 0n);
  const [treasuryArbor, setTreasuryArbor] = useState<bigint>(0n);
  useEffect(() => {
    let alive = true;
    getTreasury("arbor").then((t) => { if (alive) setTreasuryArbor(t.amount); });
    return () => { alive = false; };
  }, []);
  const totalTvl = list.reduce((a, e) => a + e.market.totalSupplied, 0n);
  const totalCollateral = Object.values(collateralByMarket).reduce(
    (a, b) => a + b,
    0n
  );
  const l4PendingBacklog = list.reduce(
    (a, e) => a + e.market.layer4PendingCount,
    0
  );
  const l4Haircuted = list.filter((e) => e.lossFactor < RAY).length;
  const coverageRatio =
    totalTvl > 0n ? Number((totalRf * 10000n) / totalTvl) / 100 : 0;

  return (
    <div className="space-y-8">
      <section className="space-y-2">
        <h1 className="display-title">
          Reserve & risk monitor
        </h1>
        <p className="text-sm text-zinc-500">
          Bad‑debt waterfall, reserve adequacy, and per‑market risk flags — read
          live from the ARBOR plugin&apos;s custom queries.
        </p>
      </section>

      <section className="space-y-4">
        <h2 className="section-h">
          4‑layer bad‑debt waterfall
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <LayerCard
            layer="Layer 1"
            title="Borrower collateral"
            desc="First line of defense — seized in liquidation"
            value={`$${totalCollateralUsd.toLocaleString(undefined, { maximumFractionDigits: 2 })}`}
            sub={collateralBreakdown || `${formatAmount(totalCollateral, 0)} native`}
            tone="emerald"
          />
          <LayerCard
            layer="Layer 2"
            title="Reserve fund (R_fund)"
            desc="Protocol reserves cover shortfalls"
            value={formatAmount(totalRf, 0)}
            tone="indigo"
          />
          <LayerCard
            layer="Layer 3"
            title="Protocol treasury (T_fund)"
            desc="Arbor treasury draw-down — isolated from NASM"
            value={formatAmount(treasuryArbor, 0)}
            tone={treasuryArbor > 0n ? "indigo" : "zinc"}
          />
          <LayerCard
            layer="Layer 4"
            title="Loss factor (socialized)"
            desc="Pro‑rata haircut to lenders"
            value={`${l4PendingBacklog > 0 ? `${l4PendingBacklog} pending` : l4Haircuted > 0 ? `${l4Haircuted} applied` : "0"}`}
            tone={l4PendingBacklog > 0 || l4Haircuted > 0 ? "rose" : "zinc"}
          />
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">Total R_fund</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            ${totalRfUsd.toLocaleString(undefined, { maximumFractionDigits: 2 })}
          </p>
          <p className="mt-1 text-[11px] text-zinc-500">
            {formatAmount(totalRf, 0)} native
          </p>
        </div>
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">Total TVL (supplied)</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            ${totalTvlUsd.toLocaleString(undefined, { maximumFractionDigits: 2 })}
          </p>
          <p className="mt-1 text-[11px] text-zinc-500">
            {formatAmount(totalTvl, 0)} native
          </p>
        </div>
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">R_fund / TVL coverage</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            {coverageRatio.toFixed(1)}%
          </p>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="section-h">
          Per‑market risk flags & reserves
        </h2>
        {isLoading ? (
          <p className="text-sm text-zinc-500">Loading markets…</p>
        ) : list.length === 0 ? (
          <p className="rounded-2xl border border-white/10 bg-white/[0.02] px-4 py-8 text-center text-sm text-zinc-500">
            No markets on chain yet.
          </p>
        ) : (
          <div className="no-scrollbar overflow-x-auto rounded-2xl glass p-5 backdrop-blur">
            <table className="w-full min-w-[56rem] text-sm">
              <thead>
                <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                  <th className="pb-2 pr-4 font-medium">Market</th>
                  <th className="pb-2 pr-4 text-center font-medium">Insolvent</th>
                  <th className="pb-2 pr-4 text-center font-medium">Idx overflow</th>
                  <th className="pb-2 pr-4 text-center font-medium">L4</th>
                  <th className="pb-2 pr-4 text-center font-medium">Paused</th>
                  <th className="pb-2 pr-4 text-center font-medium">Deprecated</th>
                  <th className="pb-2 pr-4 text-right font-medium">Supply pool</th>
                  <th className="pb-2 pr-4 text-right font-medium">Collat. pool</th>
                  <th className="pb-2 pr-4 text-right font-medium">R_fund</th>
                  <th className="pb-2 pr-4 text-right font-medium">Loss factor</th>
                  <th className="pb-2 text-right font-medium">R_fund/TVL</th>
                </tr>
              </thead>
              <tbody>
                {list.map((entry) => (
                  <MarketRow
                    key={entry.market.marketId}
                    entry={entry}
                    onCollateral={onCollateral}
                    onUsd={onUsd}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="text-[11px] text-zinc-600">
          Amounts in whole units. Supply/collateral pools from
          /v1/query/pool; R_fund from /v1/query/reservefund; loss factor from
          /v1/query/lossfactor (RAY = 1.0, no haircut).
        </p>
      </section>
    </div>
  );
}
