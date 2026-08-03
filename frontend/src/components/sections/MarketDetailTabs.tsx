"use client";

import { useEffect, useState } from "react";
import { StatusPill } from "@/components/widgets/StatusPill";
import { TIER_PARAMS } from "@/lib/arbor/constants";
import { getInterestRemainder } from "@/lib/canopy/pluginRpc";
import type { Market } from "@/lib/arbor/types";

// Rate model constants — exact mirror of contract/interest_rate.go (ARCM §14).
const U_OPTIMAL_BPS = 8000; // 80%
const BASE_RATE_BPS = 200; // 2%
const SLOPE1_BPS = 800; // 8%
const SLOPE2_BPS = 10000; // 100%
const BPS_SCALE = 10000;
const BLOCKS_PER_YEAR = 1_576_800; // 20s block time (interest_rate.go corrected figure)

function borrowRateBps(utilBps: number): number {
  if (utilBps <= U_OPTIMAL_BPS) {
    return BASE_RATE_BPS + (utilBps * SLOPE1_BPS) / U_OPTIMAL_BPS;
  }
  const excess = utilBps - U_OPTIMAL_BPS;
  const remaining = BPS_SCALE - U_OPTIMAL_BPS;
  return BASE_RATE_BPS + SLOPE1_BPS + (excess * SLOPE2_BPS) / remaining;
}

function supplyRateBps(utilBps: number, reserveFactorBps: number): number {
  const br = borrowRateBps(utilBps);
  return (br * utilBps * (BPS_SCALE - reserveFactorBps)) / (BPS_SCALE * BPS_SCALE);
}

function Flag({ bad, label }: { bad: boolean; label: string }) {
  return (
    <div className="flex items-center justify-between rounded-xl border border-white/5 bg-white/[0.02] px-3 py-2">
      <span className="text-xs text-zinc-400">{label}</span>
      <span
        className={`text-xs font-medium ${bad ? "text-rose-300" : "text-emerald-300"}`}
      >
        {bad ? "Yes" : "No"}
      </span>
    </div>
  );
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="rounded-xl border border-white/5 bg-white/[0.02] px-3 py-2">
      <p className="text-xs text-zinc-400">{label}</p>
      <p className={`mt-1 text-sm tabular-nums ${tone ?? "text-zinc-200"}`}>{value}</p>
    </div>
  );
}

function RateCurve({ utilBps }: { utilBps: number }) {
  const W = 340;
  const H = 150;
  const PAD = 30;
  const maxY = 1.15; // 115% headroom (rate reaches 110% at 100% util)
  const px = (u: number) => PAD + (u / 100) * (W - 2 * PAD);
  const py = (r: number) => H - PAD - (r / 100 / maxY) * (H - 2 * PAD);
  const pts: string[] = [];
  for (let u = 0; u <= 100; u += 2) {
    const r = borrowRateBps(u * 100) / 100;
    pts.push(`${px(u).toFixed(1)},${py(r).toFixed(1)}`);
  }
  const curU = Math.min(100, utilBps / 100);
  const curR = borrowRateBps(utilBps) / 100;
  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="w-full max-w-md">
      <line x1={px(80)} y1={PAD} x2={px(80)} y2={H - PAD} stroke="rgba(255,255,255,0.08)" strokeDasharray="3 3" />
      <line x1={PAD} y1={H - PAD} x2={W - PAD} y2={H - PAD} stroke="rgba(255,255,255,0.15)" />
      <line x1={PAD} y1={PAD} x2={PAD} y2={H - PAD} stroke="rgba(255,255,255,0.15)" />
      <polyline points={pts.join(" ")} fill="none" stroke="#818cf8" strokeWidth={2} />
      <line x1={px(curU)} y1={H - PAD} x2={px(curU)} y2={py(curR)} stroke="#34d399" strokeWidth={1.5} strokeDasharray="2 2" />
      <circle cx={px(curU)} cy={py(curR)} r={3.5} fill="#34d399" />
      <text x={PAD - 2} y={H - 10} fill="#71717a" fontSize={9}>0%</text>
      <text x={px(80) - 10} y={H - 10} fill="#71717a" fontSize={9}>80%</text>
      <text x={W - PAD - 18} y={H - 10} fill="#71717a" fontSize={9}>100%</text>
      <text x={6} y={py(0) + 3} fill="#71717a" fontSize={9}>0</text>
      <text x={2} y={py(100) + 3} fill="#71717a" fontSize={9}>110%</text>
    </svg>
  );
}

export function MarketDetailTabs({
  market,
  reserveFund,
  lossFactor,
}: {
  market: Market;
  reserveFund: bigint;
  lossFactor: bigint;
}) {
  const [tab, setTab] = useState<"admission" | "rate" | "params">("rate");
  const tier = TIER_PARAMS[market.assetTier];

  const utilBps =
    market.totalSupplied > 0n
      ? Number((market.totalBorrowed * 10000n) / market.totalSupplied)
      : 0;
  const reserveFactor = Number(market.reserveFactorBps);
  const borrowApy = borrowRateBps(utilBps) / 100;
  const supplyApy = supplyRateBps(utilBps, reserveFactor) / 100;

  const [interestRemainder, setInterestRemainder] = useState<bigint>(0n);
  useEffect(() => {
    let alive = true;
    getInterestRemainder(market.marketId).then((r) => {
      if (alive) setInterestRemainder(r.interestRemainderRay);
    });
    return () => { alive = false; };
  }, [market.marketId]);

  const TABS = [
    { key: "admission" as const, label: "Admission gates" },
    { key: "rate" as const, label: "Rate model" },
    { key: "params" as const, label: "Market parameters" },
  ];

  return (
    <div className="rounded-2xl glass p-5 backdrop-blur">
      <div className="no-scrollbar -mx-1 flex items-center gap-2 overflow-x-auto px-1">
        {TABS.map((t) => (
          <button
            key={t.key}
            type="button"
            onClick={() => setTab(t.key)}
            className={`whitespace-nowrap rounded-full border px-3 py-1.5 text-xs transition ${
              tab === t.key
                ? "border-white/15 bg-white/10 text-white"
                : "border-white/10 bg-white/[0.02] text-zinc-400 hover:bg-white/5 hover:text-zinc-200"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="mt-4">
        {tab === "admission" && (
          <div className="space-y-3">
            <div className="flex items-center justify-between rounded-xl border border-white/5 bg-white/[0.02] px-3 py-2">
              <span className="text-xs text-zinc-400">Market status</span>
              <StatusPill status={market.status} />
            </div>
            <Flag bad={market.indexOverflowHalted} label="Index-overflow halted" />
            <Flag
              bad={market.layer4PendingCount > 0}
              label="Layer-4 loss pending (blocks deposit / withdraw / borrow)"
            />
            <div className="grid grid-cols-2 gap-3">
              <Stat label="Layer-4 pending count" value={String(market.layer4PendingCount)} />
              <Stat
                label="Authorized oracle submitters"
                value={String(market.authorizedSubmitters?.length ?? 0)}
              />
            </div>
            <p className="text-[11px] text-zinc-600">
              Gates read live from the Market record ({"{16}"}). A non-zero Layer-4
              pending count blocks deposit/withdraw/borrow until the loss factor is
              applied (EventDepositWithdrawBlockedDuringPendingLoss).
            </p>
          </div>
        )}

        {tab === "rate" && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <Stat label="Utilization" value={`${(utilBps / 100).toFixed(2)}%`} />
              <Stat label="Borrow APR" value={`${borrowApy.toFixed(2)}%`} tone="text-rose-300" />
              <Stat label="Supply APR" value={`${supplyApy.toFixed(4)}%`} tone="text-emerald-300" />
              <Stat
                label="Interest remainder"
                value={(Number(interestRemainder) / 1e18).toFixed(9)}
                tone="text-zinc-400"
              />
            </div>
            <RateCurve utilBps={utilBps} />
            <p className="text-[11px] text-zinc-600">
              Kinked two-slope model (ARCM §14, interest_rate.go): base{" "}
              {BASE_RATE_BPS / 100}%, slope1 {SLOPE1_BPS / 100}% up to{" "}
              {U_OPTIMAL_BPS / 100}% utilization, slope2 {SLOPE2_BPS / 100}% beyond.
              Supply APR = borrow APR × utilization × (1 − reserve factor{" "}
              {reserveFactor / 100}%). Per-block accrual uses{" "}
              {BLOCKS_PER_YEAR.toLocaleString()} blocks/yr (20s blocks).
            </p>
          </div>
        )}

        {tab === "params" && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <Stat label="Collateral asset" value={market.collateralAssetId.toUpperCase()} />
              <Stat label="Debt asset" value={market.debtAssetId.toUpperCase()} />
              <Stat label="Asset tier" value={String(market.assetTier)} />
              <Stat label="Reserve factor (live)" value={`${reserveFactor / 100}%`} />
              {tier && (
                <>
                  <Stat label={`Max LTV (tier ${market.assetTier})`} value={`${Number(tier.ltvMaxBps) / 100}%`} />
                  <Stat label="Liquidation LTV" value={`${Number(tier.ltvLiqBps) / 100}%`} />
                  <Stat label="Liquidation incentive (LIF)" value={(Number(tier.lifBps) / 10000).toFixed(4)} />
                </>
              )}
              <Stat label="Loss factor (live)" value={(Number(lossFactor) / 1e18).toFixed(6)} />
              <Stat label="Reserve fund (live)" value={(Number(reserveFund) / 1e9).toFixed(9)} />
            </div>
            <p className="text-[11px] text-zinc-600">
              Governance parameter store ({"{22}"}) is not yet on-chain; rate-model and
              tier parameters are the hardcoded launch defaults (interest_rate.go,
              tier_params.go). Reserve factor, loss factor, and reserve fund are read
              live from the market.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
