"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useMarkets } from "@/lib/hooks/useMarkets";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import {
  getAllBorrowerPositions,
  type BorrowerPositionSummary,
} from "@/lib/canopy/pluginRpc";
import { computeHealthFactorScaled } from "@/lib/arbor/math";
import { formatAmount, formatHealthFactor } from "@/lib/arbor/format";
import {
  TIER_PARAMS,
  HF_LIQUIDATABLE_SCALED,
  STATE_REFRESH_INTERVAL_MS,
} from "@/lib/arbor/constants";
import { bytesToHex } from "@/lib/canopy/decode";
import type { Market } from "@/lib/arbor/types";

type PosMetric = {
  debtUsd: number;
  collUsd: number;
  liquidatable: boolean;
  priced: boolean;
};

function fmtUsd(n: number | null): string {
  if (n === null || !Number.isFinite(n)) return "—";
  if (n >= 1_000_000) return `$${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `$${(n / 1_000).toFixed(2)}K`;
  if (n >= 1) return `$${n.toFixed(2)}`;
  if (n > 0) return `$${n.toFixed(6)}`;
  return "$0.00";
}

function shortAddr(hex: string): string {
  if (hex.length <= 12) return hex;
  return `0x${hex.slice(0, 6)}…${hex.slice(-4)}`;
}

function PositionRow({
  p,
  market,
  report,
}: {
  p: BorrowerPositionSummary;
  market: Market | undefined;
  report: (key: string, m: PosMetric) => void;
}) {
  const collAsset = market?.collateralAssetId ?? null;
  const debtAsset = market?.debtAssetId ?? null;
  const { data: collPrice } = useAssetPrice(collAsset);
  const { data: debtPrice } = useAssetPrice(debtAsset);
  const tier = market ? TIER_PARAMS[market.assetTier] : undefined;

  const pricesOk =
    !!market &&
    !!tier &&
    !!collPrice?.available &&
    !!debtPrice?.available &&
    collPrice.price != null &&
    debtPrice.price != null;

  const hf = pricesOk
    ? computeHealthFactorScaled(
        p.collateralQuantity,
        collPrice!.price as bigint,
        tier!.ltvLiqBps,
        p.currentDebt,
        debtPrice!.price as bigint
      )
    : null;

  const noDebt = hf === 0n;
  const liquidatable = hf != null && !noDebt && hf <= HF_LIQUIDATABLE_SCALED;
  const debtUsd = pricesOk
    ? Number(p.currentDebt) * (Number(debtPrice!.price) / 1e8)
    : null;
  const collUsd = pricesOk
    ? Number(p.collateralQuantity) * (Number(collPrice!.price) / 1e8)
    : null;
  const hfActual = hf != null ? Number(hf) / 1e6 : null;
  const closeFactorBps =
    hfActual == null || noDebt ? null : hfActual > 0.95 ? 3000 : hfActual > 0.85 ? 6000 : 10000;
  const lif = tier ? Number(tier.lifBps) / 10000 : null;
  const distance =
    hfActual == null
      ? "No oracle"
      : noDebt
        ? "Safe"
        : hfActual <= 1.0
        ? "Liquidatable"
        : hfActual <= 1.2
          ? "At risk"
          : "Safe";
  const distCls =
    distance === "Liquidatable"
      ? "bg-rose-500/15 text-rose-300"
      : distance === "At risk"
        ? "bg-amber-500/15 text-amber-300"
        : distance === "Safe"
          ? "bg-emerald-500/15 text-emerald-300"
          : "bg-zinc-500/15 text-zinc-400";

  const addrHex = bytesToHex(p.address);
  const key = `${p.marketId}:${addrHex}`;

  useEffect(() => {
    report(key, {
      debtUsd: debtUsd ?? 0,
      collUsd: collUsd ?? 0,
      liquidatable,
      priced: pricesOk,
    });
  }, [report, key, debtUsd, collUsd, liquidatable, pricesOk]);

  return (
    <tr className="border-t border-white/5">
      <td className="py-2.5 pr-4 font-mono text-xs text-zinc-400">
        {shortAddr(addrHex)}
      </td>
      <td className="py-2.5 pr-4">
        <p className="text-xs text-zinc-200">{p.marketId}</p>
        <p className="text-[10px] uppercase text-zinc-600">
          {market ? `${market.collateralAssetId}/${market.debtAssetId}` : "—"}
        </p>
      </td>
      <td className="py-2.5 pr-4 text-right">
        <p className="tabular-nums text-zinc-200">{fmtUsd(debtUsd)}</p>
        <p className="text-[10px] tabular-nums text-zinc-600">
          {formatAmount(p.currentDebt, 0)} {debtAsset ?? ""}
        </p>
      </td>
      <td className="py-2.5 pr-4 text-right">
        <p className="tabular-nums text-zinc-200">{fmtUsd(collUsd)}</p>
        <p className="text-[10px] tabular-nums text-zinc-600">
          {formatAmount(p.collateralQuantity, 0)} {collAsset ?? ""}
        </p>
      </td>
      <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-200">
        {hf == null ? "—" : formatHealthFactor(hf)}
      </td>
      <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-300">
        {liquidatable && closeFactorBps != null ? `${closeFactorBps / 100}%` : "—"}
      </td>
      <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-300">
        {lif != null ? lif.toFixed(3) : "—"}
      </td>
      <td className="py-2.5 pr-4">
        <span
          className={`inline-flex whitespace-nowrap rounded-full px-2 py-0.5 text-[11px] font-medium ${distCls}`}
        >
          {distance}
        </span>
      </td>
      <td className="py-2.5 text-right">
        {liquidatable && market ? (
          <Link
            href={`/markets/${encodeURIComponent(market.marketId)}`}
            className="inline-flex rounded-full bg-rose-500/20 px-3 py-1 text-[11px] font-semibold text-rose-200 transition hover:bg-rose-500/30"
          >
            Liquidate
          </Link>
        ) : (
          <span className="text-[11px] text-zinc-600">—</span>
        )}
      </td>
    </tr>
  );
}

export default function LiquidationPage() {
  const { data: positions, isLoading } = useQuery({
    queryKey: ["all-borrower-positions"],
    queryFn: getAllBorrowerPositions,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    refetchIntervalInBackground: true,
    staleTime: 5_000,
  });
  const { data: marketsData } = useMarkets();

  const marketMap: Record<string, Market> = {};
  (marketsData ?? []).forEach((e) => {
    marketMap[e.market.marketId] = e.market;
  });

  const [metrics, setMetrics] = useState<Record<string, PosMetric>>({});
  const report = (key: string, m: PosMetric) => {
    setMetrics((prev) => {
      const old = prev[key];
      if (
        old &&
        old.debtUsd === m.debtUsd &&
        old.collUsd === m.collUsd &&
        old.liquidatable === m.liquidatable &&
        old.priced === m.priced
      )
        return prev;
      return { ...prev, [key]: m };
    });
  };

  const list = positions ?? [];
  const vals = Object.values(metrics);
  const liquidatableCount = vals.filter((v) => v.liquidatable).length;
  const pricedCount = vals.filter((v) => v.priced).length;
  const unpricedCount = vals.filter((v) => !v.priced).length;
  const totalDebtUsd = vals.reduce((a, v) => a + (v.priced ? v.debtUsd : 0), 0);
  const totalCollUsd = vals.reduce((a, v) => a + (v.priced ? v.collUsd : 0), 0);

  return (
    <div className="space-y-8">
      <section className="space-y-2">
        <h1 className="display-title">
          Liquidation dashboard
        </h1>
        <p className="max-w-2xl text-sm text-zinc-500">
          Every borrower position on chain, joined to live oracle prices and the
          market&apos;s tier parameters. Health factor uses the on-chain formula
          (health_factor.go); close factor is dynamic by HF (close_factor.go); LIF
          is per-tier. Executing a liquidation uses the Liquidate form on the
          market page.
        </p>
      </section>

      <section className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">Liquidatable positions</p>
          <p className="mt-2 text-3xl font-semibold tabular-nums text-white">
            {liquidatableCount}
          </p>
          <p className="mt-1 text-xs text-zinc-500">
            {unpricedCount > 0 ? `${unpricedCount} unpriced (HF n/a)` : "all assessable"}
          </p>
        </div>
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">Monitored positions</p>
          <p className="mt-2 text-3xl font-semibold tabular-nums text-white">
            {list.length}
          </p>
          <p className="mt-1 text-xs text-zinc-500">{pricedCount} with oracle price</p>
        </div>
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">Total debt value</p>
          <p className="mt-2 text-3xl font-semibold tabular-nums text-white">
            {fmtUsd(totalDebtUsd)}
          </p>
          <p className="mt-1 text-xs text-zinc-500">priced positions only</p>
        </div>
        <div className="rounded-2xl glass p-5 backdrop-blur">
          <p className="text-xs text-zinc-500">Total collateral value</p>
          <p className="mt-2 text-3xl font-semibold tabular-nums text-white">
            {fmtUsd(totalCollUsd)}
          </p>
          <p className="mt-1 text-xs text-zinc-500">priced positions only</p>
        </div>
      </section>

            {liquidatableCount === 0 && pricedCount > 0 && (
        <p className="all-healthy-note rounded-xl border border-emerald-500/25 bg-emerald-500/10 px-3 py-2 text-[11px] text-emerald-200/90">
          All {pricedCount} priced positions are healthy - none are currently below the liquidation threshold (HF &lt;= 1.0).
        </p>
      )}

      <section className="space-y-4">
        <h2 className="section-h">
          Borrower positions
        </h2>
        {isLoading ? (
          <p className="text-sm text-zinc-500">Loading positions…</p>
        ) : list.length === 0 ? (
          <p className="rounded-2xl border border-white/10 bg-white/[0.02] px-4 py-8 text-center text-sm text-zinc-500">
            No borrower positions on chain.
          </p>
        ) : (
          <div className="no-scrollbar overflow-x-auto rounded-2xl glass p-5 backdrop-blur">
            <table className="w-full min-w-[58rem] text-sm">
              <thead>
                <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                  <th className="pb-2 pr-4 font-medium">Position</th>
                  <th className="pb-2 pr-4 font-medium">Market</th>
                  <th className="pb-2 pr-4 text-right font-medium">Debt value</th>
                  <th className="pb-2 pr-4 text-right font-medium">Collateral value</th>
                  <th className="pb-2 pr-4 text-right font-medium">HF</th>
                  <th className="pb-2 pr-4 text-right font-medium">Close factor</th>
                  <th className="pb-2 pr-4 text-right font-medium">LIF</th>
                  <th className="pb-2 pr-4 font-medium">Distance</th>
                  <th className="pb-2 text-right font-medium">Action</th>
                </tr>
              </thead>
              <tbody>
                {list.map((p) => (
                  <PositionRow
                    key={`${p.marketId}:${bytesToHex(p.address)}`}
                    p={p}
                    market={marketMap[p.marketId]}
                    report={report}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
        {unpricedCount > 0 && (
          <p className="text-[11px] text-zinc-600">
            {unpricedCount} position(s) use assets without an oracle price, so their
            health factor cannot be computed and they are excluded from the dollar
            totals. Post a price for those assets (Oracle page) to assess them.
          </p>
        )}
        <p className="text-[11px] text-zinc-600">
          Bad-debt coverage (the 4-layer waterfall: borrower collateral → R_fund →
          backstop → loss factor) is on the{" "}
          <Link href="/monitor" className="text-indigo-300 underline-offset-2 hover:underline">
            Monitor
          </Link>{" "}
          page.
        </p>
      </section>
    </div>
  );
}
