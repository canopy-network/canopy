"use client";

import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useWalletStore } from "@/lib/wallet";
import {
  getAssetBalance,
  getNusdBalance,
  getBorrowerPosition,
  getLenderPosition,
  getAllNasmVaults,
  getAllMarkets,
} from "@/lib/canopy/pluginRpc";
import { useChainStatus } from "@/lib/hooks/useChainStatus";
import { queryTxsBySender } from "@/lib/canopy/rpc";
import { formatAmount } from "@/lib/arbor/format";
import { ActivityLog } from "@/components/explorer/ActivityLog";

const ASSETS = ["BTC", "ETH", "USDC"];
const card = "rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur";

function LiveNetworkWidget({ txCount }: { txCount: number | null }) {
  const { data: status } = useChainStatus();
  const { data: markets } = useQuery({
    queryKey: ["markets-count"],
    queryFn: getAllMarkets,
    staleTime: 30_000,
  });

  return (
    <div className="relative overflow-hidden rounded-2xl border border-emerald-500/30 bg-gradient-to-br from-emerald-500/10 via-white/[0.02] to-transparent p-6 explorer-glow">
      <div className="absolute inset-0 opacity-10">
        <div
          className="absolute inset-0 explorer-grid-drift"
          style={{
            backgroundImage:
              "linear-gradient(rgba(16,185,129,0.3) 1px, transparent 1px), linear-gradient(90deg, rgba(16,185,129,0.3) 1px, transparent 1px)",
            backgroundSize: "40px 40px",
          }}
        />
      </div>
      <div className="relative flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="relative">
            <div className="h-16 w-16 rounded-full bg-emerald-500/20 explorer-pulse" />
            <div className="absolute inset-2 rounded-full bg-emerald-500/40 explorer-pulse" style={{ animationDelay: "0.5s" }} />
            <div className="absolute inset-4 rounded-full bg-emerald-400 explorer-pulse" style={{ animationDelay: "1s" }} />
          </div>
          <div>
            <div className="text-xs font-semibold uppercase tracking-wider text-emerald-400">Network Status</div>
            <div className="mt-1 text-2xl font-bold text-white">
              {status?.connected ? "Online" : "Offline"}
            </div>
          </div>
        </div>
        <div className="grid grid-cols-3 gap-6 text-right">
          <div>
            <div className="text-xs text-zinc-500">Block Height</div>
            <div className="text-xl font-bold tabular-nums text-white">
              {status?.height?.toLocaleString() ?? "—"}
            </div>
          </div>
          <div>
            <div className="text-xs text-zinc-500">Markets</div>
            <div className="text-xl font-bold tabular-nums text-white">
              {markets?.length ?? "—"}
            </div>
          </div>
          <div>
            <div className="text-xs text-zinc-500">Address TXs</div>
            <div className="text-xl font-bold tabular-nums text-emerald-400">
              {txCount === null || txCount === undefined ? "—" : txCount.toLocaleString()}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function AddressDossier({ address }: { address: string }) {
  const { data: markets } = useQuery({
    queryKey: ["markets-for-dossier"],
    queryFn: getAllMarkets,
    staleTime: 30_000,
  });

  const { data: assetBalances } = useQuery({
    queryKey: ["asset-balances", address],
    queryFn: async () => {
      const balances = await Promise.all(
        ASSETS.map(async (asset) => {
          const bal = await getAssetBalance(asset, address);
          return { asset, amount: bal?.amount ?? BigInt(0) };
        })
      );
      return balances;
    },
    enabled: !!address,
  });

  const { data: nusdBalance } = useQuery({
    queryKey: ["nusd-balance", address],
    queryFn: () => getNusdBalance(address),
    enabled: !!address,
  });

  const { data: borrowerPositions } = useQuery({
    queryKey: ["borrower-positions-dossier", address],
    queryFn: async () => {
      if (!markets) return [];
      const positions = await Promise.all(
        markets.map(async (m: any) => {
          const pos = await getBorrowerPosition(m.marketId, address);
          return pos ? { ...pos, marketId: m.marketId } : null;
        })
      );
      return positions.filter(Boolean);
    },
    enabled: !!address && !!markets,
  });

  const { data: lenderPositions } = useQuery({
    queryKey: ["lender-positions-dossier", address],
    queryFn: async () => {
      if (!markets) return [];
      const positions = await Promise.all(
        markets.map(async (m: any) => {
          const pos = await getLenderPosition(m.marketId, address);
          return pos ? { ...pos, marketId: m.marketId } : null;
        })
      );
      return positions.filter(Boolean);
    },
    enabled: !!address && !!markets,
  });

  const { data: vaults } = useQuery({
    queryKey: ["vaults-dossier", address],
    queryFn: () => getAllNasmVaults(address),
    enabled: !!address,
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-white">Address Dossier</h2>
        <div className="flex items-center gap-2 rounded-full bg-emerald-500/10 px-3 py-1">
          <div className="h-2 w-2 rounded-full bg-emerald-400 explorer-pulse" />
          <span className="text-xs font-medium text-emerald-300">Live</span>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 explorer-card-stagger">
        {/* Asset Balances */}
        <div className={`${card} p-5`}>
          <h3 className="text-sm font-semibold text-white">Asset Balances</h3>
          <div className="mt-3 space-y-2">
            {assetBalances?.map((b) => (
              <div key={b.asset} className="flex items-center justify-between text-sm">
                <span className="text-zinc-400">{b.asset}</span>
                <span className="font-mono text-white">
                  {formatAmount(b.amount, b.asset === "USDC" ? 6 : 8)}
                </span>
              </div>
            )) ?? <div className="text-sm text-zinc-600">Loading…</div>}
          </div>
        </div>

        {/* NUSD Balance */}
        <div className={`${card} p-5`}>
          <h3 className="text-sm font-semibold text-white">NUSD Balance</h3>
          <div className="mt-3">
            <div className="text-2xl font-bold tabular-nums text-white">
              {nusdBalance ? formatAmount(nusdBalance.amount, 6) : "0.00"}
            </div>
            <div className="mt-1 text-xs text-zinc-500">Stablecoin holdings</div>
          </div>
        </div>

        {/* Borrower Positions */}
        <div className={`${card} p-5`}>
          <h3 className="text-sm font-semibold text-white">Borrower Positions</h3>
          <div className="mt-3 space-y-2">
            {borrowerPositions?.length ? (
              borrowerPositions.map((p: any) => (
                <div key={p.marketId} className="flex items-center justify-between text-sm">
                  <span className="text-zinc-400">{p.marketId}</span>
                  <div className="text-right">
                    <div className="font-mono text-white">
                      {formatAmount(BigInt(p.currentDebt ?? p.debtPrincipal), 6)} debt
                    </div>
                    <div className="text-xs text-zinc-500">
                      {formatAmount(BigInt(p.collateralQuantity ?? 0), 8)} collateral
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-sm text-zinc-600">No borrower positions</div>
            )}
          </div>
        </div>

        {/* Lender Positions */}
        <div className={`${card} p-5`}>
          <h3 className="text-sm font-semibold text-white">Lender Positions</h3>
          <div className="mt-3 space-y-2">
            {lenderPositions?.length ? (
              lenderPositions.map((p: any) => (
                <div key={p.marketId} className="flex items-center justify-between text-sm">
                  <span className="text-zinc-400">{p.marketId}</span>
                  <span className="font-mono text-white">
                    {formatAmount(BigInt(p.shares ?? 0), 6)} shares
                  </span>
                </div>
              ))
            ) : (
              <div className="text-sm text-zinc-600">No lender positions</div>
            )}
          </div>
        </div>

        {/* NASM Vaults */}
        <div className={`${card} p-5 md:col-span-2`}>
          <h3 className="text-sm font-semibold text-white">NASM Vaults</h3>
          <div className="mt-3 space-y-2">
            {vaults?.length ? (
              vaults.map((v: any) => (
                <div key={v.vaultId} className="flex items-center justify-between text-sm">
                  <span className="font-mono text-xs text-zinc-400">{v.vaultId}</span>
                  <div className="text-right">
                    <div className="text-white">{v.collateralAssetId}</div>
                    <div className="text-xs text-zinc-500">
                      {formatAmount(BigInt(v.collateralQuantity ?? 0), 8)}
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-sm text-zinc-600">No NASM vaults</div>
            )}
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-4">
        <div className="flex items-start gap-3">
          <svg className="h-5 w-5 shrink-0 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          <div className="text-sm text-emerald-200">
            <p className="font-medium">Privacy-preserving lookups</p>
            <p className="mt-1 text-xs text-emerald-300/80">
              All queries run locally in your browser against public chain state.
              No search history is stored or relayed through any server.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function EventsPage() {
  const { address } = useWalletStore();
  const [lookupAddress, setLookupAddress] = useState("");
  const [activeAddress, setActiveAddress] = useState<string | null>(null);

  useEffect(() => {
    if (address && !activeAddress) {
      setActiveAddress(address);
    }
  }, [address, activeAddress]);

  const { data: txTotal } = useQuery({
    queryKey: ["tx-total", activeAddress],
    queryFn: async () => (await queryTxsBySender(activeAddress as string)).length,
    enabled: !!activeAddress,
    refetchInterval: 15_000,
  });

  function handleLookup(e: React.FormEvent) {
    e.preventDefault();
    if (lookupAddress.trim()) {
      setActiveAddress(lookupAddress.trim().replace(/^0x/, ""));
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-2 flex items-center gap-2">
        <h1 className="text-3xl font-semibold tracking-tight text-white">Arbor Explorer</h1>
        <span className="rounded-full bg-emerald-500/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-300">
          Live
        </span>
      </div>
      <p className="mb-8 max-w-2xl text-sm text-zinc-400">
        Real-time network monitoring and address exploration. Look up any address to see
        asset balances, positions, and vault ownership — all queries run client-side for
        maximum privacy.
      </p>

      <LiveNetworkWidget txCount={txTotal ?? null} />

      <div className="mt-8 mb-8">
        <form onSubmit={handleLookup} className="flex gap-3">
          <input
            type="text"
            value={lookupAddress}
            onChange={(e) => setLookupAddress(e.target.value)}
            placeholder="Enter address (0x... or hex) to explore"
            className="flex-1 rounded-xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder-zinc-600 outline-none focus:border-emerald-500/40"
          />
          {address && (
            <button
              type="button"
              onClick={() => setActiveAddress(address)}
              className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm font-medium text-emerald-300 hover:bg-emerald-500/20"
            >
              Use my wallet
            </button>
          )}
          <button
            type="submit"
            disabled={!lookupAddress.trim()}
            className="rounded-xl bg-gradient-to-r from-emerald-500 to-teal-600 px-6 py-3 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
          >
            Look up
          </button>
        </form>
      </div>

      {activeAddress ? (
        <AddressDossier address={activeAddress} />
      ) : (
        <div className={`${card} p-12 text-center`}>
          <div className="text-4xl">🔍</div>
          <p className="mt-4 text-sm text-zinc-500">
            Enter an address above to explore its on-chain state
          </p>
        </div>
      )}
    <div className="mt-12 border-t border-white/5 pt-8">
      <ActivityLog externalAddress={activeAddress} />
    </div>
    </div>
  );
}
