"use client";

import { useEffect, useState } from "react";
import { useMarkets } from "@/lib/hooks/useMarkets";
import { useWalletStore } from "@/lib/wallet";
import {
  getNusdSupply,
  getStabilityFeeIndex,
  getNasmTier,
  getNusdBalance,
  getTreasury,
  getNasmVaultPool,
  getAllNasmVaults,
} from "@/lib/canopy/pluginRpc";
import { formatAmount } from "@/lib/arbor/format";
import { MintNusdForm } from "@/components/forms/MintNusdForm";
import { BurnNusdForm } from "@/components/forms/BurnNusdForm";
import { LiquidateNasmVaultForm } from "@/components/forms/LiquidateNasmVaultForm";
import { AssetIcon } from "@/components/AssetIcon";
import { useNasmTierBacking } from "@/lib/hooks/useNasmVault";

const RAY = 1000000000000000000n;

function fmtBps(bps: bigint): string {
  return (Number(bps) / 100).toFixed(2) + "%";
}


function NasmVaults() {
  const wallet = useWalletStore();
  const [rows, setRows] = useState<Array<{ v: Record<string, unknown>; pool: bigint }>>([]);
  const [loading, setLoading] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const refetch = () => setRefreshKey((k) => k + 1);

  useEffect(() => {
    if (!wallet.address) return;
    let alive = true;
    getAllNasmVaults(wallet.address).then(async (list) => {
      const withPool = await Promise.all(
        list.map(async (v) => {
          const p = await getNasmVaultPool(String(v.vaultId ?? ""));
          return { v, pool: p?.amount ?? 0n };
        })
      );
      if (alive) { setRows(withPool); setLoading(false); }
    });
    return () => { alive = false; };
  }, [wallet.address, refreshKey]);

  return (
    <section className="space-y-4">
      <h2 className="section-h">Mint &amp; manage NUSD</h2>
      <div className="grid gap-4 lg:grid-cols-3">
        <MintNusdForm onMinted={refetch} />
        <BurnNusdForm onBurned={refetch} />
        <LiquidateNasmVaultForm onLiquidated={refetch} />
      </div>

      <h2 className="section-h">Your Vaults</h2>
      <p className="text-xs text-zinc-500">
        Vaults owned by the connected wallet, enumerated live via
        /v1/query/all-nasm-vaults?owner=… Collateral and minted NUSD principal come
        from each NasmVault record; escrowed collateral from nasmvaultpool.
      </p>
      {!wallet.address ? (
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
          Connect a wallet to view your vaults.
        </div>
      ) : loading ? (
        <p className="text-sm text-zinc-400">Loading vaults…</p>
      ) : rows.length === 0 ? (
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm text-zinc-400">
          No NUSD vaults found for this address.
        </div>
      ) : (
        <div className="glass rounded-2xl p-5 backdrop-blur no-scrollbar overflow-x-auto">
          <table className="w-full min-w-[36rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Vault</th>
                <th className="pb-2 pr-4 font-medium">Collateral</th>
                <th className="pb-2 pr-4 font-medium">Escrowed</th>
                <th className="pb-2 font-medium">NUSD principal</th>
              </tr>
            </thead>
            <tbody>
              {rows.map(({ v, pool }) => (
                <tr key={String(v.vaultId)} className="border-t border-white/5">
                  <td className="py-2.5 pr-4 font-mono text-xs text-zinc-200">{String(v.vaultId)}</td>
                  <td className="py-2.5 pr-4 tabular-nums text-zinc-300">
                    <span className="inline-flex items-center gap-2">
                      <AssetIcon symbol={String(v.collateralAssetId ?? "")} size={22} className="rounded-full" />
                      {formatAmount(BigInt(String(v.collateralQuantity ?? "0")), 0)} {String(v.collateralAssetId ?? "")}
                    </span>
                  </td>
                  <td className="py-2.5 pr-4 tabular-nums text-zinc-300">{formatAmount(pool, 0)}</td>
                  <td className="py-2.5 tabular-nums text-amber-200">{formatAmount(BigInt(String(v.nusdPrincipal ?? "0")), 6)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

export default function NusdPage() {
  const wallet = useWalletStore();
  const { data: markets } = useMarkets();
  const [supply, setSupply] = useState<bigint | null>(null);
  const [sfi, setSfi] = useState<{ sfIndexDecimal: bigint; lastAccrualBlock: number } | null>(null);
  const [treasury, setTreasury] = useState<bigint | null>(null);
  const [balance, setBalance] = useState<bigint | null>(null);
  const [tiers, setTiers] = useState<
    Record<string, { eligible: boolean; nasmTier: string; ltvMaxBps: bigint; ltvLiqBps: bigint }>
  >({});

  useEffect(() => {
    let alive = true;
    getNusdSupply().then((s) => { if (alive) setSupply(s.totalSupply); });
    getStabilityFeeIndex().then((v) => { if (alive) setSfi(v); });
    getTreasury("nasm").then((t) => { if (alive) setTreasury(t.amount); });
    return () => { alive = false; };
  }, []);

  useEffect(() => {
    if (!wallet.address) return;
    let alive = true;
    getNusdBalance(wallet.address).then((b) => { if (alive) setBalance(b.amount); });
    return () => { alive = false; };
  }, [wallet.address]);

  const assetIds = Array.from(
    new Set([
      ...(markets ?? []).flatMap((e) => [e.market.collateralAssetId, e.market.debtAssetId]),
      "eth", "CNPY", "usdc",
    ])
  );

  useEffect(() => {
    let alive = true;
    Promise.all(assetIds.map((a) => getNasmTier(a))).then((rs) => {
      if (!alive) return;
      const map: Record<string, { eligible: boolean; nasmTier: string; ltvMaxBps: bigint; ltvLiqBps: bigint }> = {};
      for (const r of rs) map[r.assetId] = r;
      setTiers(map);
    });
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(assetIds)]);

  const sfMult = sfi ? Number((sfi.sfIndexDecimal * 1000000n) / RAY) / 1000000 : null;
  const feePct = sfMult != null ? (sfMult - 1) * 100 : null;

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-amber-300/80">
          Arbor stablecoin · NASM
        </p>
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="display-title">NUSD</h1>
            <p className="mt-2 max-w-xl text-sm text-zinc-500">
              The ARBOR-native stablecoin, minted against NASM-tier collateral via vaults. A
              stability fee accrues to every minted position each block; the NASM treasury is
              fully isolated from Arbor lending&apos;s.
            </p>
          </div>
          <div className="text-right">
            <p className="text-xs text-zinc-500">Total supply</p>
            <p className="mt-1 text-3xl font-semibold tabular-nums text-amber-200">
              {supply != null ? formatAmount(supply, 6) : "—"}
              <span className="ml-1 text-base text-amber-300/60">NUSD</span>
            </p>
          </div>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="glass rounded-2xl p-5 backdrop-blur transition hover:border-amber-400/30">
          <p className="text-xs text-zinc-500">NUSD supply</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            {supply != null ? formatAmount(supply, 6) : "—"}
          </p>
          <p className="mt-1 text-[11px] text-zinc-500">1e6 precision</p>
        </div>
        <div className="glass rounded-2xl p-5 backdrop-blur transition hover:border-amber-400/30">
          <p className="flex items-center gap-1.5 text-xs text-zinc-500">
            Stability fee index
            <span className="relative flex h-1.5 w-1.5">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-60"></span>
              <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-amber-400"></span>
            </span>
          </p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            {sfMult != null ? sfMult.toFixed(6) + "×" : "—"}
          </p>
          <p className="mt-1 text-[11px] text-zinc-500">
            {feePct != null ? "+" + feePct.toFixed(6) + "% accrued · " : ""}
            {sfi ? "block " + sfi.lastAccrualBlock : "accruing per block"}
          </p>
        </div>
        <div className="glass rounded-2xl p-5 backdrop-blur transition hover:border-amber-400/30">
          <p className="text-xs text-zinc-500">NASM treasury {"{41}"}</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            {treasury != null ? formatAmount(treasury, 6) : "—"}
          </p>
          <p className="mt-1 text-[11px] text-zinc-500">isolated from Arbor {"{40}"}</p>
        </div>
        <div className="glass rounded-2xl p-5 backdrop-blur transition hover:border-amber-400/30">
          <p className="text-xs text-zinc-500">Your NUSD balance</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums text-white">
            {wallet.address ? (balance != null ? formatAmount(balance, 6) : "—") : "connect wallet"}
          </p>
          <p className="mt-1 text-[11px] text-zinc-500">
            {wallet.address ? "held by this address" : "to view your balance"}
          </p>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="section-h">NASM tier eligibility</h2>
        <p className="text-xs text-zinc-500">
          Which assets may back NUSD minting, at NASM&apos;s tighter LTV table (bridged from
          ARCM&apos;s {"{29}"} registry). Tier 2/3 assets are not eligible.
        </p>
        <div className="glass rounded-2xl p-5 backdrop-blur no-scrollbar overflow-x-auto">
          <table className="w-full min-w-[36rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Asset</th>
                <th className="pb-2 pr-4 font-medium">NASM tier</th>
                <th className="pb-2 pr-4 font-medium">Max LTV</th>
                <th className="pb-2 pr-4 font-medium">Liq LTV</th>
                <th className="pb-2 font-medium">Eligible</th>
              </tr>
            </thead>
            <tbody>
              {assetIds.map((a) => {
                const t = tiers[a];
                return (
                  <tr key={a} className="border-t border-white/5">
                    <td className="py-2.5 pr-4 font-medium text-zinc-200">
                      <span className="inline-flex items-center gap-2">
                        <AssetIcon symbol={a} size={22} className="rounded-full" />
                        {a}
                      </span>
                    </td>
                    <td className="py-2.5 pr-4 text-zinc-300">{t ? (t.eligible ? t.nasmTier : "—") : "…"}</td>
                    <td className="py-2.5 pr-4 tabular-nums text-zinc-300">{t && t.eligible ? fmtBps(t.ltvMaxBps) : "—"}</td>
                    <td className="py-2.5 pr-4 tabular-nums text-zinc-300">{t && t.eligible ? fmtBps(t.ltvLiqBps) : "—"}</td>
                    <td className="py-2.5">
                      {t ? (
                        t.eligible ? (
                          <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-300">eligible</span>
                        ) : (
                          <span className="rounded-full bg-zinc-500/15 px-2 py-0.5 text-[11px] font-medium text-zinc-400">not eligible</span>
                        )
                      ) : (
                        <span className="text-zinc-600">…</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <p className="text-[11px] text-zinc-600">
          Vault-level detail (per-vault collateral and minted NUSD) needs a vault id; the plugin
          exposes /v1/query/nasmvault and /v1/query/nasmvaultpool per vault, but there is no route
          to enumerate all vaults yet. Total supply and per-address balances above are live.
        </p>
      </section>

      <NasmTierConcentrationCap />
      <NasmVaults />
    </div>
  );
}

// NASM Spec Section 3.3: no single NASM tier (N-0 or N-1) may back more
// than maxTierShareBps of total NUSD supply. Shown proactively here so a
// mint that would breach the cap can be avoided before submitting, rather
// than only discovered via the on-chain rejection (error 258).
function NasmTierConcentrationCap() {
  const { data, isLoading } = useNasmTierBacking();

  const maxBps = data?.maxTierShareBps ?? 7000n;

  const rows = [
    { label: "Tier N-0", backing: data?.tierN0Backing ?? 0n, shareBps: data?.tierN0ShareBps ?? 0n },
    { label: "Tier N-1", backing: data?.tierN1Backing ?? 0n, shareBps: data?.tierN1ShareBps ?? 0n },
  ];

  return (
    <section className="space-y-4">
      <h2 className="section-h">NASM tier concentration cap</h2>
      <p className="text-xs text-zinc-500">
        No single NASM tier may back more than {fmtBps(maxBps)} of total NUSD supply (Spec
        Section 3.3). A mint that would push a tier over this line is rejected on-chain.
      </p>
      <div className="glass rounded-2xl p-5 backdrop-blur space-y-4">
        {rows.map((r) => {
          const pct = Number(r.shareBps) / 100;
          const capPct = Number(maxBps) / 100;
          const overCap = Number(r.shareBps) >= Number(maxBps);
          const nearCap = !overCap && Number(r.shareBps) >= Number(maxBps) * 0.9;
          return (
            <div key={r.label} className="space-y-1.5">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium text-zinc-200">{r.label}</span>
                <span className="tabular-nums text-zinc-400">
                  {isLoading ? "…" : `${formatAmount(r.backing, 6)} NUSD · ${pct.toFixed(2)}% of ${capPct.toFixed(0)}% cap`}
                </span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-white/5">
                <div
                  className={`h-full rounded-full ${overCap ? "bg-rose-500" : nearCap ? "bg-amber-400" : "bg-emerald-400"}`}
                  style={{ width: `${Math.min(100, (pct / capPct) * 100)}%` }}
                />
              </div>
              {overCap && (
                <p className="text-[11px] text-rose-400">At or above cap — mints into this tier will be rejected.</p>
              )}
              {nearCap && (
                <p className="text-[11px] text-amber-400">Approaching cap — headroom is limited.</p>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}
