"use client";

import { useCallback, useEffect, useState } from "react";
import { useMarkets, type MarketWithIndices } from "@/lib/hooks/useMarkets";
import { useWalletStore } from "@/lib/stores/walletStore";
import { useLenderPosition } from "@/lib/hooks/useLenderPosition";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { computeHealthFactorScaled } from "@/lib/arbor/math";
import { formatAmount, formatHealthFactor } from "@/lib/arbor/format";
import { TIER_PARAMS } from "@/lib/arbor/constants";

function Monogram({ symbol }: { symbol: string }) {
  return (
    <div className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-indigo-500/80 to-emerald-400/80 text-[10px] font-bold text-[#05070d]">
      {symbol.slice(0, 4).toUpperCase()}
    </div>
  );
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

function LendingRow({
  entry,
  address,
  onPresent,
}: {
  entry: MarketWithIndices;
  address: string;
  onPresent: (id: string, v: boolean) => void;
}) {
  const m = entry.market;
  const { data: lp } = useLenderPosition(m.marketId, address);
  const { data: price } = useAssetPrice(m.debtAssetId);
  const shares = lp?.shares ?? 0n;
  const value =
    price?.available && price.price != null
      ? (Number(shares) / 1e9) * (Number(price.price) / 1e8)
      : null;

  useEffect(() => {
    onPresent(m.marketId, shares > 0n);
    return () => onPresent(m.marketId, false);
  }, [onPresent, m.marketId, shares]);

  if (shares === 0n) return null;

  return (
    <tr className="border-t border-white/5">
      <MarketCell entry={entry} />
      <td className="py-3 pr-4 text-right tabular-nums text-zinc-100">
        {formatAmount(shares, 9)}
      </td>
      <td className="py-3 text-right tabular-nums text-zinc-300">
        {value != null ? `$${value.toFixed(6)}` : "—"}
      </td>
    </tr>
  );
}

function BorrowingRow({
  entry,
  address,
  onPresent,
  onHf,
}: {
  entry: MarketWithIndices;
  address: string;
  onPresent: (id: string, v: boolean) => void;
  onHf: (id: string, v: number | null) => void;
}) {
  const m = entry.market;
  const { data: bp } = useBorrowerPosition(m.marketId, address);
  const { data: collPrice } = useAssetPrice(m.collateralAssetId);
  const { data: debtPrice } = useAssetPrice(m.debtAssetId);
  const tier = TIER_PARAMS[m.assetTier];

  const debt = bp?.currentDebt ?? 0n;
  const coll = bp?.collateralQuantity ?? 0n;
  const present = debt > 0n || coll > 0n;

  const pricesOk =
    !!tier &&
    debt > 0n &&
    !!collPrice?.available &&
    !!debtPrice?.available &&
    collPrice.price != null &&
    debtPrice.price != null;

  const hf = pricesOk
    ? computeHealthFactorScaled(
        coll,
        collPrice!.price as bigint,
        tier.ltvLiqBps,
        debt,
        debtPrice!.price as bigint
      )
    : null;
  const hfActual = hf != null ? Number(hf) / 1e6 : null;
  const distance =
    hfActual != null && hfActual > 1
      ? ((hfActual - 1) / hfActual) * 100
      : null;

  useEffect(() => {
    onPresent(m.marketId, present);
    onHf(m.marketId, hfActual);
    return () => {
      onPresent(m.marketId, false);
      onHf(m.marketId, null);
    };
  }, [onPresent, onHf, m.marketId, present, hfActual]);

  if (!present) return null;

  const debtUsd =
    pricesOk && debtPrice!.price != null
      ? (Number(debt) / 1e9) * (Number(debtPrice!.price) / 1e8)
      : null;
  const collUsd =
    pricesOk && collPrice!.price != null
      ? (Number(coll) / 1e9) * (Number(collPrice!.price) / 1e8)
      : null;

  return (
    <tr className="border-t border-white/5">
      <MarketCell entry={entry} />
      <td className="py-3 pr-4 text-right">
        <p className="tabular-nums text-zinc-100">{formatAmount(debt, 9)}</p>
        <p className="text-[10px] tabular-nums text-zinc-600">
          {debtUsd != null ? `$${debtUsd.toFixed(6)}` : m.debtAssetId}
        </p>
      </td>
      <td className="py-3 pr-4 text-right">
        <p className="tabular-nums text-zinc-100">{formatAmount(coll, 9)}</p>
        <p className="text-[10px] tabular-nums text-zinc-600">
          {collUsd != null ? `$${collUsd.toFixed(6)}` : m.collateralAssetId}
        </p>
      </td>
      <td className="py-3 pr-4 text-right">
        <HfPill hf={hf} />
      </td>
      <td className="py-3 text-right tabular-nums text-zinc-400">
        {distance != null ? `${distance.toFixed(1)}%` : "—"}
      </td>
    </tr>
  );
}

function PanelShell({
  title,
  count,
  connected,
  hasMarkets,
  hasAny,
  emptyCopy,
  children,
}: {
  title: string;
  count: number;
  connected: boolean;
  hasMarkets: boolean;
  hasAny: boolean;
  emptyCopy: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-5 backdrop-blur">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-100">{title}</h3>
        {connected && (
          <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-300">
            {count} active
          </span>
        )}
      </div>
      {!connected ? (
        <p className="mt-4 text-sm text-zinc-500">
          Connect a wallet to view your {title.toLowerCase()}.
        </p>
      ) : !hasMarkets ? (
        <p className="mt-4 text-sm text-zinc-500">
          No markets exist yet to hold positions in.
        </p>
      ) : !hasAny ? (
        <p className="mt-4 text-sm text-zinc-500">{emptyCopy}</p>
      ) : (
        <div className="no-scrollbar mt-3 overflow-x-auto">{children}</div>
      )}
    </div>
  );
}

export function Portfolio() {
  const wallet = useWalletStore();
  const { data: markets } = useMarkets();

  const [lend, setLend] = useState<Record<string, boolean>>({});
  const [borr, setBorr] = useState<Record<string, boolean>>({});
  const [hfs, setHfs] = useState<Record<string, number | null>>({});

  const markLend = useCallback(
    (id: string, v: boolean) => setLend((p) => ({ ...p, [id]: v })),
    []
  );
  const markBorr = useCallback(
    (id: string, v: boolean) => setBorr((p) => ({ ...p, [id]: v })),
    []
  );
  const markHf = useCallback(
    (id: string, v: number | null) => setHfs((p) => ({ ...p, [id]: v })),
    []
  );

  const list = markets ?? [];
  const connected = wallet.isConnected && !!wallet.address;
  const address = wallet.address ?? "";
  const hasMarkets = list.length > 0;
  const lendCount = Object.values(lend).filter(Boolean).length;
  const borrCount = Object.values(borr).filter(Boolean).length;

  const atRisk = Object.entries(hfs).filter(([, v]) => v != null && v < 1.2);

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold tracking-tight text-white">
          Your portfolio
        </h2>
        {connected && (
          <span className="font-mono text-xs tabular-nums text-zinc-500">
            {address.slice(0, 6)}…{address.slice(-4)}
          </span>
        )}
      </div>

      {connected && atRisk.length > 0 && (
        <div className="flex items-start gap-3 rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3">
          <span className="text-amber-300">⚠</span>
          <p className="text-sm text-amber-200">
            Your <span className="font-semibold">{atRisk[0][0]}</span> borrow
            position is approaching the liquidation threshold (HF{" "}
            {atRisk[0][1]?.toFixed(2)}). Consider adding collateral or repaying
            debt.
          </p>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        <PanelShell
          title="Lending positions"
          count={lendCount}
          connected={connected}
          hasMarkets={hasMarkets}
          hasAny={lendCount > 0}
          emptyCopy="No open lending positions. Supply assets in a market to see them here."
        >
          <table className="w-full min-w-[24rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Market</th>
                <th className="pb-2 pr-4 text-right font-medium">Shares</th>
                <th className="pb-2 text-right font-medium">Value</th>
              </tr>
            </thead>
            <tbody>
              {list.map((entry) => (
                <LendingRow
                  key={entry.market.marketId}
                  entry={entry}
                  address={address}
                  onPresent={markLend}
                />
              ))}
            </tbody>
          </table>
        </PanelShell>

        <PanelShell
          title="Borrowing positions"
          count={borrCount}
          connected={connected}
          hasMarkets={hasMarkets}
          hasAny={borrCount > 0}
          emptyCopy="No open borrowing positions. Deposit collateral and borrow to see them here."
        >
          <table className="w-full min-w-[30rem] text-sm">
            <thead>
              <tr className="text-left text-[11px] uppercase tracking-wide text-zinc-500">
                <th className="pb-2 pr-4 font-medium">Market</th>
                <th className="pb-2 pr-4 text-right font-medium">Debt</th>
                <th className="pb-2 pr-4 text-right font-medium">Collateral</th>
                <th className="pb-2 pr-4 text-right font-medium">HF</th>
                <th className="pb-2 text-right font-medium">Liq. distance</th>
              </tr>
            </thead>
            <tbody>
              {list.map((entry) => (
                <BorrowingRow
                  key={entry.market.marketId}
                  entry={entry}
                  address={address}
                  onPresent={markBorr}
                  onHf={markHf}
                />
              ))}
            </tbody>
          </table>
        </PanelShell>
      </div>

      {!connected && (
        <p className="text-center text-xs text-zinc-600">
          Positions, health factors, and liquidation distance are read live from
          the ARBOR plugin once a wallet is connected.
        </p>
      )}
    </section>
  );
}
