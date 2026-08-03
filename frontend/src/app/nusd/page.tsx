"use client";

import { useEffect, useState } from "react";
import { useMarkets } from "@/lib/hooks/useMarkets";
import { useWalletStore } from "@/lib/stores/walletStore";
import {
  getNusdSupply,
  getStabilityFeeIndex,
  getNasmTier,
  getNusdBalance,
  getTreasury,
  getNasmVault,
  getNasmVaultPool,
} from "@/lib/canopy/pluginRpc";
import { formatAmount } from "@/lib/arbor/format";

const RAY = 1000000000000000000n;

function fmtBps(bps: bigint): string {
  return (Number(bps) / 100).toFixed(2) + "%";
}


function NasmVaults() {
  const [vaultId, setVaultId] = useState("");
  const [tracked, setTracked] = useState<string[]>(() => {
    try { return JSON.parse(localStorage.getItem("arbor-tracked-vaults") ?? "[]") as string[]; } catch { return []; }
  });
  const [vaults, setVaults] = useState<Record<string, { v: Record<string, unknown>; pool: bigint }>>({});
  const [err, setErr] = useState<string | null>(null);

  const load = async (id: string) => {
    if (!id) return;
    const [v, pool] = await Promise.all([getNasmVault(id), getNasmVaultPool(id)]);
    if (!v) { setErr("vault not found: " + id); return; }
    setErr(null);
    setVaults((p) => ({ ...p, [id]: { v, pool: pool?.amount ?? 0n } }));
    setTracked((t) => {
      const n = t.includes(id) ? t : [...t, id];
      try { localStorage.setItem("arbor-tracked-vaults", JSON.stringify(n)); } catch {}
      return n;
    });
  };

  useEffect(() => { tracked.forEach((id) => load(id)); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const ids = Object.keys(vaults);
  return (
    <section className="space-y-4">
      <h2 className="section-h">Vaults</h2>
      <p className="text-xs text-zinc-500">
        There is no on-chain route to enumerate vaults yet, so load one by id (or any vault you
        mint is tracked here). Collateral and minted NUSD principal are read live from
        /v1/query/nasmvault; escrowed collateral from /v1/query/nasmvaultpool.
      </p>
      <div className="flex gap-2">
        <input
          value={vaultId}
          onChange={(e) => setVaultId(e.target.value)}
          placeholder="vault id"
          className="w-full rounded-xl border border-white/10 bg-white/[0.03] px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-amber-400/50"
        />
        <button
          type="button"
          onClick={() => load(vaultId.trim())}
          className="shrink-0 rounded-xl border border-amber-400/30 bg-amber-400/10 px-4 py-2 text-xs font-semibold text-amber-300 transition hover:bg-amber-400/20"
        >
          Load vault
        </button>
      </div>
      {err && <p className="text-xs text-rose-300">{err}</p>}
      {ids.length > 0 && (
        <div className="glass rounded-2xl p-5 backdrop-blur no-scrollbar overflow-x-auto">
          <table className="w-full min-w-[36rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Vault</th>
                <th className="pb-2 pr-4 font-medium">Collateral</th>
                <th className="pb-2 pr-4 font-medium">Escrowed</th>
                <th className="pb-2 pr-4 font-medium">NUSD principal</th>
                <th className="pb-2 font-medium">Owner</th>
              </tr>
            </thead>
            <tbody>
              {ids.map((id) => {
                const { v, pool } = vaults[id];
                return (
                  <tr key={id} className="border-t border-white/5">
                    <td className="py-2.5 pr-4 font-mono text-xs text-zinc-200">{id}</td>
                    <td className="py-2.5 pr-4 tabular-nums text-zinc-300">
                      {formatAmount(BigInt(String(v.collateralQuantity ?? "0")), 9)} {String(v.collateralAssetId ?? "")}
                    </td>
                    <td className="py-2.5 pr-4 tabular-nums text-zinc-300">{formatAmount(pool, 9)}</td>
                    <td className="py-2.5 pr-4 tabular-nums text-amber-200">{formatAmount(BigInt(String(v.nusdPrincipal ?? "0")), 6)}</td>
                    <td className="py-2.5 font-mono text-xs text-zinc-500">{String(v.owner ?? "").slice(0, 10)}…</td>
                  </tr>
                );
              })}
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
    if (!wallet.address) { setBalance(null); return; }
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
            {treasury != null ? formatAmount(treasury, 9) : "—"}
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
                    <td className="py-2.5 pr-4 font-medium text-zinc-200">{a}</td>
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
      <NasmVaults />
    </div>
  );
}
