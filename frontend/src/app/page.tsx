"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useMarkets, type MarketWithIndices } from "@/lib/hooks/useMarkets";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { StatusPill } from "@/components/widgets/StatusPill";
import { LoadingSkeleton } from "@/components/widgets/LoadingSkeleton";
import { formatAmount } from "@/lib/arbor/format";
import type { MarketStatus } from "@/lib/arbor/types";
import { Portfolio } from "@/components/sections/Portfolio";

type Filter = "all" | MarketStatus;

const TABS: { key: Filter; label: string }[] = [
  { key: "all", label: "All statuses" },
  { key: "ACTIVE", label: "Active" },
  { key: "PAUSED", label: "Paused" },
  { key: "INSOLVENT", label: "Insolvent" },
  { key: "DEPRECATED", label: "Deprecated" },
];

const TIER_LABEL: Record<number, string> = {
  0: "Tier 0 · CNPY",
  1: "Tier 1 · BTC / ETH",
  2: "Tier 2 · Majors",
  3: "Tier 3 · Restricted",
};

// Rate model — exact mirror of contract/interest_rate.go (ARCM §14).
const U_OPT = 8000;
const BASE = 200;
const S1 = 800;
const S2 = 10000;
const BPS = 10000;
function borrowAprBps(u: number): number {
  if (u <= U_OPT) return BASE + (u * S1) / U_OPT;
  return BASE + S1 + ((u - U_OPT) * S2) / (BPS - U_OPT);
}
function supplyAprBps(u: number, rf: number): number {
  const br = borrowAprBps(u);
  return (br * u * (BPS - rf)) / (BPS * BPS);
}

function fmtUsd(n: number | null): string {
  if (n === null || !Number.isFinite(n)) return "—";
  if (n >= 1_000_000) return `$${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `$${(n / 1_000).toFixed(2)}K`;
  if (n >= 1) return `$${n.toFixed(2)}`;
  if (n > 0) return `$${n.toFixed(6)}`;
  return "$0.00";
}

function fmtUnitPrice(price: bigint): string {
  const u = Number(price) / 1e8;
  return `$${u.toLocaleString(undefined, { maximumFractionDigits: 2 })}`;
}

function fmtPct(n: number | null): string {
  if (n === null || !Number.isFinite(n)) return "—";
  return `${n.toFixed(1)}%`;
}

function utilBand(u: number | null): { bar: string; text: string } {
  if (u === null) return { bar: "bg-zinc-600", text: "text-zinc-500" };
  if (u < 70) return { bar: "bg-emerald-400", text: "text-emerald-300" };
  if (u < 85) return { bar: "bg-amber-400", text: "text-amber-300" };
  return { bar: "bg-rose-400", text: "text-rose-300" };
}

function Monogram({ symbol }: { symbol: string }) {
  return (
    <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-indigo-500/80 to-emerald-400/80 text-xs font-bold text-[#05070d] shadow-lg shadow-indigo-500/10">
      {symbol.slice(0, 4).toUpperCase()}
    </div>
  );
}

function StatCard({
  label,
  value,
  sub,
  subTone = "muted",
}: {
  label: string;
  value: string;
  sub?: string;
  subTone?: "muted" | "up" | "down";
}) {
  const tone =
    subTone === "up"
      ? "text-emerald-400"
      : subTone === "down"
        ? "text-rose-400"
        : "text-zinc-500";
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 backdrop-blur">
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-tight tabular-nums text-white">
        {value}
      </p>
      {sub && <p className={`mt-1 text-xs tabular-nums ${tone}`}>{sub}</p>}
    </div>
  );
}

function MarketCard({
  entry,
  report,
}: {
  entry: MarketWithIndices;
  report: (id: string, s: number, b: number, priced: boolean) => void;
}) {
  const m = entry.market;
  const { data: priceData } = useAssetPrice(m.debtAssetId);
  const price =
    priceData?.available && priceData.price != null ? priceData.price : null;

  const suppliedUsd =
    price !== null ? (Number(m.totalSupplied) / 1e9) * (Number(price) / 1e8) : null;
  const borrowedUsd =
    price !== null ? (Number(m.totalBorrowed) / 1e9) * (Number(price) / 1e8) : null;

  useEffect(() => {
    report(m.marketId, suppliedUsd ?? 0, borrowedUsd ?? 0, suppliedUsd !== null);
  }, [report, m.marketId, suppliedUsd, borrowedUsd]);

  const util =
    m.totalSupplied > 0n
      ? (Number(m.totalBorrowed) / Number(m.totalSupplied)) * 100
      : null;
  const utilBps =
    m.totalSupplied > 0n
      ? Number((m.totalBorrowed * 10000n) / m.totalSupplied)
      : 0;
  const band = utilBand(util);
  const lossFactorRatio = Number(entry.lossFactor) / 1e18;
  const borrowApr = borrowAprBps(utilBps) / 100;
  const supplyApr = supplyAprBps(utilBps, Number(m.reserveFactorBps)) / 100;

  return (
    <Link
      href={`/markets/${encodeURIComponent(m.marketId)}`}
      className="group flex flex-col rounded-2xl border border-white/10 bg-white/[0.03] p-5 backdrop-blur transition hover:border-white/20 hover:bg-white/[0.05]"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <Monogram symbol={m.collateralAssetId} />
          <div className="leading-tight">
            <p className="text-sm font-semibold text-white">{m.marketId}</p>
            <p className="text-[11px] text-zinc-500">
              {TIER_LABEL[m.assetTier] ?? `Tier ${m.assetTier}`}
            </p>
          </div>
        </div>
        <StatusPill status={m.status} />
      </div>

      <div className="mt-5 grid grid-cols-2 gap-3 text-xs">
        <div>
          <p className="text-zinc-500">Supplied</p>
          {suppliedUsd !== null ? (
            <>
              <p className="mt-1 tabular-nums text-zinc-100">
                {fmtUsd(suppliedUsd)}
              </p>
              <p className="text-[10px] tabular-nums text-zinc-600">
                {formatAmount(m.totalSupplied, 9)} {m.debtAssetId}
              </p>
            </>
          ) : (
            <>
              <p className="mt-1 tabular-nums text-zinc-100">
                {formatAmount(m.totalSupplied, 9)}
              </p>
              <p className="text-[10px] uppercase text-zinc-600">
                {m.debtAssetId}
              </p>
            </>
          )}
        </div>
        <div>
          <p className="text-zinc-500">Borrowed</p>
          {borrowedUsd !== null ? (
            <>
              <p className="mt-1 tabular-nums text-zinc-100">
                {fmtUsd(borrowedUsd)}
              </p>
              <p className="text-[10px] tabular-nums text-zinc-600">
                {formatAmount(m.totalBorrowed, 9)} {m.debtAssetId}
              </p>
            </>
          ) : (
            <>
              <p className="mt-1 tabular-nums text-zinc-100">
                {formatAmount(m.totalBorrowed, 9)}
              </p>
              <p className="text-[10px] uppercase text-zinc-600">
                {m.debtAssetId}
              </p>
            </>
          )}
        </div>
      </div>

      <div className="mt-4">
        <div className="flex items-center justify-between text-xs">
          <span className="text-zinc-500">Utilization</span>
          <span className={`tabular-nums ${band.text}`}>{fmtPct(util)}</span>
        </div>
        <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-white/5">
          <div
            className={`h-full rounded-full transition-all ${band.bar}`}
            style={{ width: `${util !== null ? Math.min(100, util) : 0}%` }}
          />
        </div>
      </div>

      <div className="mt-3 grid grid-cols-2 gap-2 border-t border-white/5 pt-3 text-[10px] tabular-nums text-zinc-500">
        <div>
          Borrow APR
          <p className="text-rose-300">{borrowApr.toFixed(2)}%</p>
        </div>
        <div>
          Supply APR
          <p className="text-emerald-300">{supplyApr.toFixed(2)}%</p>
        </div>
      </div>

      <div className="mt-3 grid grid-cols-3 gap-2 border-t border-white/5 pt-3 text-[10px] tabular-nums text-zinc-500">
        <div>
          R_fund
          <p className="text-zinc-300">{formatAmount(entry.reserveFund, 9)}</p>
        </div>
        <div>
          Loss factor
          <p className="text-zinc-300">{lossFactorRatio.toFixed(6)}</p>
        </div>
        <div>
          L4 pending
          <p className="text-zinc-300">{m.layer4PendingCount}</p>
        </div>
      </div>

      {price !== null ? (
        <p className="mt-3 text-[10px] text-zinc-600">
          {m.debtAssetId} @ {fmtUnitPrice(price)} · live oracle
        </p>
      ) : (
        <p className="mt-3 text-[10px] text-zinc-600">
          USD valuation pending oracle quorum ({m.collateralAssetId}/
          {m.debtAssetId})
        </p>
      )}
    </Link>
  );
}

export default function HomePage() {
  const { data: markets, isLoading, error } = useMarkets();
  const [usd, setUsd] = useState<
    Record<string, { s: number; b: number; priced: boolean }>
  >({});
  const [filter, setFilter] = useState<Filter>("all");

  const report = useCallback(
    (id: string, s: number, b: number, priced: boolean) => {
      setUsd((prev) => {
        if (prev[id]?.s === s && prev[id]?.b === b && prev[id]?.priced === priced)
          return prev;
        return { ...prev, [id]: { s, b, priced } };
      });
    },
    []
  );

  const list = markets ?? [];
  const count = list.length;

  const totalSupplied = list.reduce((a, e) => a + e.market.totalSupplied, 0n);
  const totalBorrowed = list.reduce((a, e) => a + e.market.totalBorrowed, 0n);
  const util =
    totalSupplied > 0n
      ? (Number(totalBorrowed) / Number(totalSupplied)) * 100
      : null;

  const counts = list.reduce(
    (acc, e) => {
      acc[e.market.status] = (acc[e.market.status] ?? 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const shown =
    filter === "all" ? list : list.filter((e) => e.market.status === filter);

  const empty = !isLoading && count === 0;
  const activeLabel = TABS.find((t) => t.key === filter)?.label ?? "";

  const activeCount = counts["ACTIVE"] ?? 0;

  const pricedSupplied = Object.values(usd).reduce(
    (a, v) => (v.priced ? a + v.s : a),
    0
  );
  const pricedBorrowed = Object.values(usd).reduce(
    (a, v) => (v.priced ? a + v.b : a),
    0
  );
  const unpricedActive = list.filter(
    (e) =>
      !usd[e.market.marketId]?.priced &&
      (e.market.totalSupplied > 0n || e.market.totalBorrowed > 0n)
  ).length;
  const unitLabel =
    Array.from(new Set(list.map((e) => e.market.debtAssetId))).length === 1
      ? ` ${list[0]?.market.debtAssetId ?? ""}`
      : "";

  return (
    <div className="space-y-8">
      <section className="space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight text-white">
              Protocol overview
            </h1>
            <p className="mt-1 text-sm text-zinc-500">
              Isolated ARBOR lending markets, read live from the plugin RPC.
            </p>
          </div>
          <Link
            href="/admin"
            className="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-indigo-500 to-violet-500 px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-indigo-500/20 transition hover:from-indigo-400 hover:to-violet-400"
          >
            + Create market
          </Link>
        </div>

        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard
            label="Markets"
            value={count.toString()}
            sub={empty ? "none yet" : `${activeCount} active`}
            subTone={empty ? "muted" : "up"}
          />
          <StatCard
            label="Total value locked"
            value={
              pricedSupplied > 0
                ? fmtUsd(pricedSupplied)
                : formatAmount(totalSupplied, 9)
            }
            sub={
              pricedSupplied > 0
                ? `${formatAmount(totalSupplied, 9)} native${
                    unpricedActive > 0 ? ` · ${unpricedActive} unpriced` : ""
                  }`
                : `native units${unitLabel}`
            }
          />
          <StatCard
            label="Total borrowed"
            value={
              pricedBorrowed > 0
                ? fmtUsd(pricedBorrowed)
                : formatAmount(totalBorrowed, 9)
            }
            sub={
              pricedBorrowed > 0
                ? `${formatAmount(totalBorrowed, 9)} native${
                    unpricedActive > 0 ? ` · ${unpricedActive} unpriced` : ""
                  }`
                : `native units${unitLabel}`
            }
            subTone="down"
          />
          <StatCard
            label="Utilization"
            value={fmtPct(util)}
            sub="borrowed / supplied"
          />
        </div>

        <p className="text-[11px] text-zinc-600">
          Dollar values use live oracle medians from /v1/query/prices and only
          sum markets whose debt asset is priced (the standard TVL convention);
          native 9‑decimal totals are shown alongside. Health factor and per‑card
          APR derive from the on‑chain rate model.
        </p>
      </section>

      <Portfolio />

      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold tracking-tight text-white">
            All markets
          </h2>
          <span className="text-xs tabular-nums text-zinc-600">
            {count} total
          </span>
        </div>

        <div className="no-scrollbar -mx-1 flex items-center gap-2 overflow-x-auto px-1">
          {TABS.map((t) => {
            const n = t.key === "all" ? count : counts[t.key] ?? 0;
            const active = filter === t.key;
            return (
              <button
                key={t.key}
                type="button"
                onClick={() => setFilter(t.key)}
                className={`inline-flex items-center gap-2 whitespace-nowrap rounded-full border px-3 py-1.5 text-xs transition ${
                  active
                    ? "border-white/15 bg-white/10 text-white"
                    : "border-white/10 bg-white/[0.02] text-zinc-400 hover:bg-white/5 hover:text-zinc-200"
                }`}
              >
                {t.label}
                <span
                  className={`tabular-nums ${active ? "text-zinc-300" : "text-zinc-600"}`}
                >
                  {n}
                </span>
              </button>
            );
          })}
        </div>

        {error && (
          <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-xs text-rose-300">
            Failed to load markets. Check the ARBOR plugin RPC connection.
          </div>
        )}

        {isLoading && <LoadingSkeleton rows={4} />}

        {!isLoading && empty && (
          <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-10 text-center backdrop-blur">
            <div className="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-gradient-to-br from-indigo-500 to-emerald-400 text-lg font-extrabold text-[#05070d]">
              A
            </div>
            <p className="mt-4 text-base font-semibold text-white">
              No markets yet
            </p>
            <p className="mx-auto mt-1 max-w-sm text-sm text-zinc-500">
              ARBOR markets are created on chain by an authority. Launch your
              first isolated market to populate the overview and portfolio.
            </p>
            <Link
              href="/admin"
              className="mt-5 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-xs font-semibold text-zinc-200 transition hover:bg-white/10"
            >
              Go to Admin
            </Link>
          </div>
        )}

        {!isLoading && !empty && shown.length === 0 && (
          <p className="rounded-2xl border border-white/10 bg-white/[0.02] px-4 py-8 text-center text-sm text-zinc-500">
            No {activeLabel.toLowerCase()} markets.
          </p>
        )}

        <div className="grid gap-4 md:grid-cols-2">
          {shown.map((entry) => (
            <MarketCard key={entry.market.marketId} entry={entry} report={report} />
          ))}
        </div>
      </section>
    </div>
  );
}
