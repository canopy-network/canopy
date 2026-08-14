"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { useMarkets, type MarketWithIndices } from "@/lib/hooks/useMarkets";
import { useLenderPosition } from "@/lib/hooks/useLenderPosition";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useNasmVaultsByOwner } from "@/lib/hooks/useNasmVault";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { useAccount } from "@/lib/hooks/useAccount";
import { getAssetBalance } from "@/lib/canopy/pluginRpc";
import { computeHealthFactorScaled } from "@/lib/arbor/math";
import { formatAmount, formatHealthFactor, formatAddress } from "@/lib/arbor/format";
import { TIER_PARAMS } from "@/lib/arbor/constants";
import { AssetIcon } from "@/components/AssetIcon";

function Monogram({ symbol }: { symbol: string }) {
  return <AssetIcon symbol={symbol} size={36} className="shrink-0 rounded-full shadow-md shadow-black/30" />;
}

function HfPill({ hf }: { hf: bigint | null }) {
  if (hf == null) return <span className="text-xs text-zinc-600">—</span>;
  const v = Number(hf) / 1e6;
  const cls =
    v < 1.0
      ? "bg-rose-500/15 text-rose-300"
      : v < 1.5
        ? "bg-amber-500/15 text-amber-300"
        : "bg-emerald-500/15 text-emerald-300";
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium tabular-nums ${cls}`}
    >
      {formatHealthFactor(hf)}
    </span>
  );
}

function MarketCell({ entry }: { entry: MarketWithIndices }) {
  const m = entry.market;
  return (
    <td className="py-3 pr-4">
      <div className="flex items-center gap-3">
        <Monogram symbol={m.collateralAssetId} />
        <div className="leading-tight">
          <p className="text-sm font-medium text-zinc-100">{m.marketId}</p>
          <p className="text-[11px] uppercase text-zinc-500">
            {m.collateralAssetId}/{m.debtAssetId}
          </p>
        </div>
      </div>
    </td>
  );
}

function LendingRow({ entry, address }: { entry: MarketWithIndices; address: string }) {
  const m = entry.market;
  const { data: lp } = useLenderPosition(m.marketId, address);
  const { data: price } = useAssetPrice(m.debtAssetId);
  const shares = lp?.shares ?? 0n;
  const value =
    price?.available && price.price != null
      ? (Number(shares) / 1e9) * (Number(price.price) / 1e8)
      : null;

  if (shares === 0n) return null;

  return (
    <tr className="border-t border-white/5">
      <MarketCell entry={entry} />
      <td className="py-3 text-right">
        <p className="text-sm tabular-nums text-zinc-100">
          {formatAmount(shares, 9)}
        </p>
        <p className="text-[11px] text-zinc-500">
          {value != null ? `$${value.toFixed(2)}` : "—"}
        </p>
      </td>
      <td className="py-3 text-right text-sm text-zinc-400">—</td>
    </tr>
  );
}

function BorrowingRow({ entry, address }: { entry: MarketWithIndices; address: string }) {
  const m = entry.market;
  const { data: bp } = useBorrowerPosition(m.marketId, address);
  const { data: collPrice } = useAssetPrice(m.collateralAssetId);
  const { data: debtPrice } = useAssetPrice(m.debtAssetId);
  const tier = TIER_PARAMS[m.assetTier];

  const debt = bp?.currentDebt ?? 0n;
  const coll = bp?.collateralQuantity ?? 0n;

  if (debt === 0n && coll === 0n) return null;

  const hf = tier && collPrice?.available && collPrice.price != null && debtPrice?.available && debtPrice.price != null
    ? computeHealthFactorScaled(
        coll,
        collPrice.price,
        tier.ltvLiqBps,
        debt,
        debtPrice.price,
      )
    : null;

  return (
    <tr className="border-t border-white/5">
      <MarketCell entry={entry} />
      <td className="py-3 text-right">
        <p className="text-sm tabular-nums text-zinc-100">
          {formatAmount(coll, 9)}
        </p>
        <p className="text-[11px] text-zinc-500">{m.collateralAssetId}</p>
      </td>
      <td className="py-3 text-right">
        <p className="text-sm tabular-nums text-zinc-100">
          {formatAmount(debt, 9)}
        </p>
        <p className="text-[11px] text-zinc-500">{m.debtAssetId}</p>
      </td>
      <td className="py-3 text-right">
        <HfPill hf={hf} />
      </td>
    </tr>
  );
}


const WATCH_ASSETS: { id: string; icon: string; name: string; note: string }[] = [
  { id: "CNPY", icon: "canopy", name: "Canopy", note: "Faucet balance (whole units)" },
  { id: "BTC", icon: "bitcoin", name: "Bitcoin", note: "Collateral asset" },
  { id: "ETH", icon: "eth", name: "Ether", note: "Collateral asset" },
  { id: "USDC", icon: "usdc", name: "USD Coin", note: "Stablecoin" },
];

function AssetRows({ address }: { address: string }) {
  const [amounts, setAmounts] = useState<Record<string, bigint | null>>({});
  useEffect(() => {
    let alive = true;
    (async () => {
      const entries = await Promise.all(
        WATCH_ASSETS.map(async (a) => {
          const r = await getAssetBalance(a.id, address);
          return [a.id, r?.amount ?? null] as const;
        })
      );
      if (!alive) return;
      setAmounts(Object.fromEntries(entries));
    })().catch(() => {});
    return () => {
      alive = false;
    };
  }, [address]);

  return (
    <>
      {WATCH_ASSETS.map((a) => {
        const amt = amounts[a.id];
        if (amt == null || amt <= 0n) return null;
        return (
          <div key={a.id} className="flex items-center justify-between py-2">
            <div className="flex items-center gap-3">
              <Monogram symbol={a.icon} />
            <div>
                <p className="text-sm font-medium text-zinc-100">{a.name}</p>
                <p className="text-[11px] text-zinc-500">{a.note}</p>
              </div>
            </div>
            <p className="text-sm font-semibold tabular-nums text-white">
              {amt.toString()}
            </p>
          </div>
        );
      })}
    </>
  );
}

export default function WatchPage() {
  const params = useParams();
  const router = useRouter();
  const watchedAddress = (params.address as string).toLowerCase();
  const [inputAddr, setInputAddr] = useState(watchedAddress);

  const { data: markets } = useMarkets();
  const { data: accountData } = useAccount(watchedAddress);
  const { data: nasmVaults } = useNasmVaultsByOwner(watchedAddress);

  const [lendingPresent, setLendingPresent] = useState<Record<string, boolean>>({});
  const [borrowingPresent, setBorrowingPresent] = useState<Record<string, boolean>>({});

  const trackLending = useCallback((id: string, v: boolean) => {
    setLendingPresent((prev) => ({ ...prev, [id]: v }));
  }, []);

  const trackBorrowing = useCallback((id: string, v: boolean) => {
    setBorrowingPresent((prev) => ({ ...prev, [id]: v }));
  }, []);

  useEffect(() => {
    setInputAddr(watchedAddress);
  }, [watchedAddress]);

  const handleSwitch = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = inputAddr.trim().toLowerCase();
    if (trimmed && trimmed !== watchedAddress) {
      router.push(`/watch/${trimmed}`);
    }
  };

  const cnpy = accountData?.amount ?? null;
  const staked = accountData?.stakedAmount ?? null;

  return (
    <div className="space-y-8">
      <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
        <h1 className="display-title text-2xl font-bold text-white mb-4">
          Watch mode
        </h1>
        <p className="text-sm text-zinc-400 mb-4">
          Read-only view of any wallet's positions on the Arbor protocol.
        </p>
        <form onSubmit={handleSwitch} className="flex gap-2">
          <input
            type="text"
            value={inputAddr}
            onChange={(e) => setInputAddr(e.target.value)}
            placeholder="Enter address (40 hex chars)"
            className="flex-1 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:border-indigo-400 focus:outline-none"
          />
          <button
            type="submit"
            className="rounded-lg bg-indigo-500 px-6 py-2 text-sm font-medium text-white transition hover:bg-indigo-400"
          >
            Switch
          </button>
        </form>
        <p className="mt-3 text-xs text-zinc-500 font-mono break-all">
          Watching: {watchedAddress}
        </p>
      </div>

      <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
        <h2 className="text-lg font-semibold text-white mb-4">Balances</h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between py-2">
            <div className="flex items-center gap-3">
              <Monogram symbol="ARBOR" />
              <div>
                <p className="text-sm font-medium text-zinc-100">Arbor</p>
                <p className="text-[11px] text-zinc-500">Native token</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-sm font-semibold tabular-nums text-white">
                {cnpy != null ? formatAmount(cnpy, 9) : "—"}
              </p>
              {staked != null && staked > 0n && (
                <p className="text-[11px] text-zinc-500">
                  Staked: {formatAmount(staked, 9)}
                </p>
              )}
            </div>
          </div>
          <AssetRows address={watchedAddress} />
        </div>
      </div>

      {markets && markets.length > 0 && (
        <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
          <h2 className="text-lg font-semibold text-white mb-4">Lending positions</h2>
          <table className="w-full">
            <thead>
              <tr className="border-b border-white/5 text-left text-[11px] uppercase text-zinc-500">
                <th className="py-2 pr-4">Market</th>
                <th className="py-2 text-right">Shares</th>
                <th className="py-2 text-right">Value</th>
              </tr>
            </thead>
            <tbody>
              {markets.map((entry) => (
                <LendingRow key={entry.market.marketId} entry={entry} address={watchedAddress} />
              ))}
            </tbody>
          </table>
          {Object.values(lendingPresent).every((v) => !v) && (
            <p className="py-8 text-center text-sm text-zinc-500">
              No lending positions for this address.
            </p>
          )}
        </div>
      )}

      {markets && markets.length > 0 && (
        <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
          <h2 className="text-lg font-semibold text-white mb-4">Borrowing positions</h2>
          <table className="w-full">
            <thead>
              <tr className="border-b border-white/5 text-left text-[11px] uppercase text-zinc-500">
                <th className="py-2 pr-4">Market</th>
                <th className="py-2 text-right">Collateral</th>
                <th className="py-2 text-right">Debt</th>
                <th className="py-2 text-right">Health</th>
              </tr>
            </thead>
            <tbody>
              {markets.map((entry) => (
                <BorrowingRow key={entry.market.marketId} entry={entry} address={watchedAddress} />
              ))}
            </tbody>
          </table>
          {Object.values(borrowingPresent).every((v) => !v) && (
            <p className="py-8 text-center text-sm text-zinc-500">
              No borrowing positions for this address.
            </p>
          )}
        </div>
      )}

      {nasmVaults && nasmVaults.length > 0 && (
        <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
          <h2 className="text-lg font-semibold text-white mb-4">NASM vaults</h2>
          <div className="space-y-3">
            {nasmVaults.map((vault: any) => (
              <div key={String(vault.vaultId)} className="rounded-lg border border-white/5 bg-white/[0.03] p-4">
                <p className="text-sm font-medium text-zinc-100 mb-2">
                  Vault {String(vault.vaultId)} — {String(vault.collateralAssetId)}
                </p>
                <div className="grid grid-cols-2 gap-4 text-xs">
                  <div>
                    <p className="text-zinc-500">Escrowed</p>
                    <p className="text-zinc-100 tabular-nums">{formatAmount(vault.escrowed as bigint, 9)}</p>
                  </div>
                  <div>
                    <p className="text-zinc-500">Principal</p>
                    <p className="text-zinc-100 tabular-nums">{formatAmount(vault.principal as bigint, 9)}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
