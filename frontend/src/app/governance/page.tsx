"use client";

import Link from "next/link";
import { useMarkets } from "@/lib/hooks/useMarkets";
import { TIER_PARAMS } from "@/lib/arbor/constants";
import { getGovernanceParams } from "@/lib/canopy/pluginRpc";
import { useEffect, useState } from "react";

type Tone = "emerald" | "amber" | "zinc";
const toneCls: Record<Tone, string> = {
  emerald: "bg-emerald-500/15 text-emerald-300",
  amber: "bg-amber-500/15 text-amber-300",
  zinc: "bg-zinc-500/15 text-zinc-400",
};

function Pill({ tone, label }: { tone: Tone; label: string }) {
  return (
    <span
      className={`inline-flex whitespace-nowrap rounded-full px-2 py-0.5 text-[11px] font-medium ${toneCls[tone]}`}
    >
      {label}
    </span>
  );
}

interface ParamRow {
  name: string;
  value: string;
  bounds: string;
  tone: Tone;
  status: string;
  action: string;
  editable?: boolean;
}

export default function GovernancePage() {
  const { data: markets } = useMarkets();
  const list = markets ?? [];

  const [gov, setGov] = useState<{ treasuryCutBps: bigint } | null>(null);
  useEffect(() => {
    let alive = true;
    getGovernanceParams().then((g) => { if (alive) setGov(g); });
    return () => { alive = false; };
  }, []);

  const reserveFactors = list.map((e) => ({
    id: e.market.marketId,
    bps: Number(e.market.reserveFactorBps),
  }));
  const reserveSummary = reserveFactors.length
    ? reserveFactors.map((r) => `${r.id}: ${(r.bps / 100).toFixed(2)}%`).join(" · ")
    : "per-market";

  const tierAssets = (t: number): string =>
    Array.from(
      new Set(
        list
          .filter((e) => e.market.assetTier === t)
          .map((e) => e.market.collateralAssetId)
      )
    ).join(", ") || "—";

  const params: ParamRow[] = [
    {
      name: "RESERVE_FACTOR",
      value: `Per-market · ${reserveSummary}`,
      bounds: "2% – 30% (200–3000 bps)",
      tone: "emerald",
      status: "Enforced per-market",
      action: "update_market_params",
      editable: true,
    },
    {
      name: "LIQUIDATION_CLOSE_FACTOR",
      value: "Dynamic: 30% / 60% / 100% by HF",
      bounds: "HF-tiered (ARCM §7)",
      tone: "emerald",
      status: "Enforced (dynamic)",
      action: "hardcoded · close_factor.go",
    },
    {
      name: "LIQUIDATION_INCENTIVE_FACTOR",
      value: "Per-tier: 1.030 / 1.036 / 1.050 / 1.090",
      bounds: "per asset tier",
      tone: "emerald",
      status: "Enforced per-tier",
      action: "hardcoded · tier_params.go",
    },
    {
      name: "LIQUIDATION_THRESHOLD (HF)",
      value: "HF ≤ 1.0",
      bounds: "ARCM §5 boundary",
      tone: "emerald",
      status: "Enforced",
      action: "hardcoded · health_factor.go",
    },
    {
      name: "MAX_LTV (tier 0–3)",
      value: "80% / 75% / 65% / 40%",
      bounds: "per asset tier",
      tone: "emerald",
      status: "Enforced per-tier",
      action: "hardcoded · tier_params.go",
    },
    {
      name: "ORACLE_STALENESS_THRESHOLD",
      value: "30 blocks",
      bounds: "placeholder for per-tier 50/30/20/10",
      tone: "amber",
      status: "Enforced (default)",
      action: "hardcoded · price_resolve.go",
    },
    {
      name: "MIN_REPORTERS (oracle quorum)",
      value: "1 (devnet override)",
      bounds: "ARCM spec = 3",
      tone: "amber",
      status: "Enforced (devnet override)",
      action: "hardcoded · price_resolve.go",
    },
    {
      name: "MIN_DEPOSIT",
      value: "Not enforced",
      bounds: "AYIS §13: 1 CNPY (1–100), pending NUSD",
      tone: "amber",
      status: "Not enforced",
      action: "deferred · deposit.go",
    },
    {
      name: "BORROW_CAP",
      value: "None",
      bounds: "—",
      tone: "zinc",
      status: "Not implemented",
      action: "—",
    },
    {
      name: "SUPPLY_CAP",
      value: "None",
      bounds: "—",
      tone: "zinc",
      status: "Not implemented",
      action: "—",
    },
    {
      name: "TREASURY_CUT",
      value: gov ? Number(gov.treasuryCutBps) + " bps" : "…",
      bounds: "25–150 bps (checked at DeliverTx)",
      tone: "emerald",
      status: "Enforced (live {22})",
      action: "set_treasury_cut",
    },
    {
      name: "CIRCUIT_BREAKER_DEVIATION",
      value: "Not implemented",
      bounds: "ARCM Rule 4 deferred",
      tone: "zinc",
      status: "Not implemented",
      action: "— · {20} has no writer",
    },
    {
      name: "DUST_CLAMP_THRESHOLD",
      value: "Values rounding to zero",
      bounds: "no explicit constant",
      tone: "emerald",
      status: "Enforced (events on clamp)",
      action: "implicit",
    },
  ];

  const tiers = [0, 1, 2, 3].map((t) => {
    const p = TIER_PARAMS[t] ?? { ltvMaxBps: 0n, ltvLiqBps: 0n, lifBps: 0n };
    return {
      tier: t,
      maxLtv: Number(p.ltvMaxBps) / 100,
      liqLtv: Number(p.ltvLiqBps) / 100,
      lif: Number(p.lifBps) / 10000,
      assets: tierAssets(t),
    };
  });

  const closeFactorTiers = [
    { range: "HF > 0.95", factor: "30%", bps: "3000 bps", tone: "emerald" as Tone },
    { range: "0.85 < HF ≤ 0.95", factor: "60%", bps: "6000 bps", tone: "amber" as Tone },
    { range: "HF ≤ 0.85", factor: "100%", bps: "10000 bps", tone: "zinc" as Tone },
  ];

  return (
    <div className="space-y-8">
      <section className="space-y-2">
        <h1 className="display-title">
          Governance parameters
        </h1>
        <p className="max-w-2xl text-sm text-zinc-500">
          Protocol parameters and their on-chain status. The governance parameter
          store ({"{22}"}) now holds treasury_cut_bps on-chain (live row below); the remaining values are still the hardcoded; the values below are the hardcoded
          launch defaults enforced directly in the contract code, with bounds where
          the code enforces them. RESERVE_FACTOR (per-market) and TREASURY_CUT (global) have live on-chain governance
          action today.
        </p>
      </section>

      <section className="space-y-4">
        <h2 className="section-h">
          Parameters
        </h2>
        <div className="no-scrollbar overflow-x-auto rounded-2xl glass p-5 backdrop-blur">
          <table className="w-full min-w-[52rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Parameter</th>
                <th className="pb-2 pr-4 font-medium">Current value</th>
                <th className="pb-2 pr-4 font-medium">Bounds</th>
                <th className="pb-2 pr-4 font-medium">Status</th>
                <th className="pb-2 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              {params.map((p) => (
                <tr key={p.name} className="border-t border-white/5">
                  <td className="py-2.5 pr-4 font-mono text-xs text-zinc-200">
                    {p.name}
                  </td>
                  <td className="py-2.5 pr-4 text-xs text-zinc-300">{p.value}</td>
                  <td className="py-2.5 pr-4 text-xs text-zinc-500">{p.bounds}</td>
                  <td className="py-2.5 pr-4">
                    <Pill tone={p.tone} label={p.status} />
                  </td>
                  <td className="py-2.5 text-xs">
                    {p.editable ? (
                      <Link
                        href="/admin"
                        className="text-indigo-300 underline-offset-2 hover:underline"
                      >
                        Edit (Admin)
                      </Link>
                    ) : (
                      <span className="text-zinc-500">{p.action}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        <section className="space-y-4">
          <h2 className="section-h">
            Asset tiers
          </h2>
          <div className="no-scrollbar overflow-x-auto rounded-2xl glass p-5 backdrop-blur">
            <table className="w-full min-w-[26rem] text-sm">
              <thead>
                <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                  <th className="pb-2 pr-4 font-medium">Tier</th>
                  <th className="pb-2 pr-4 text-right font-medium">Max LTV</th>
                  <th className="pb-2 pr-4 text-right font-medium">Liq LTV</th>
                  <th className="pb-2 pr-4 text-right font-medium">LIF</th>
                  <th className="pb-2 font-medium">Assets (live)</th>
                </tr>
              </thead>
              <tbody>
                {tiers.map((t) => (
                  <tr key={t.tier} className="border-t border-white/5">
                    <td className="py-2.5 pr-4 text-zinc-200">Tier {t.tier}</td>
                    <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-300">
                      {t.maxLtv.toFixed(0)}%
                    </td>
                    <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-300">
                      {t.liqLtv.toFixed(0)}%
                    </td>
                    <td className="py-2.5 pr-4 text-right tabular-nums text-zinc-300">
                      {t.lif.toFixed(3)}
                    </td>
                    <td className="py-2.5 font-mono text-xs text-zinc-400">
                      {t.assets}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="mt-3 text-[11px] text-zinc-600">
              From tier_params.go. Tier 4 (Blacklisted) is never stored — an asset
              with no {"{29}"} registry entry is ineligible, not Tier 4. Asset
              column reflects collateral assets of live markets.
            </p>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="section-h">
            Liquidation close-factor curve
          </h2>
          <div className="rounded-2xl glass p-5 backdrop-blur">
            <div className="space-y-2">
              {closeFactorTiers.map((c) => (
                <div
                  key={c.range}
                  className="flex items-center justify-between rounded-xl border border-white/5 bg-white/[0.02] px-3 py-2"
                >
                  <span className="text-xs text-zinc-400">{c.range}</span>
                  <span className="flex items-center gap-3">
                    <span className="text-sm tabular-nums text-zinc-100">
                      {c.factor}
                    </span>
                    <Pill tone={c.tone} label={c.bps} />
                  </span>
                </div>
              ))}
            </div>
            <p className="mt-3 text-[11px] text-zinc-600">
              ARCM §7 dynamic close factor (close_factor.go): the fraction of a
              liquidatable position&apos;s debt a liquidator may repay, tiered by how
              far underwater the position is. Max repay = current debt × close
              factor.
            </p>
          </div>
        </section>
      </div>
    </div>
  );
}
