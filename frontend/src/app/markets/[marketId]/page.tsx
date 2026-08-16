"use client";
import { useState, type ReactNode } from "react";
import { AssetIcon } from "@/components/AssetIcon";

import { useParams } from "next/navigation";
import { useMarket } from "@/lib/hooks/useMarket";
import { useWalletStore } from "@/lib/wallet";
import { useLenderPosition } from "@/lib/hooks/useLenderPosition";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { EmptyState } from "@/components/widgets/EmptyState";
import { LoadingSkeleton } from "@/components/widgets/LoadingSkeleton";
import { StatusPill } from "@/components/widgets/StatusPill";
import { DepositForm } from "@/components/forms/DepositForm";
import { WithdrawForm } from "@/components/forms/WithdrawForm";
import { DepositCollateralForm } from "@/components/forms/DepositCollateralForm";
import { WithdrawCollateralForm } from "@/components/forms/WithdrawCollateralForm";
import { BorrowForm } from "@/components/forms/BorrowForm";
import { RepayForm } from "@/components/forms/RepayForm";
import { LiquidateForm } from "@/components/forms/LiquidateForm";
import { UpdatePriceForm } from "@/components/forms/UpdatePriceForm";
import { MarketDetailTabs } from "@/components/sections/MarketDetailTabs";
import { formatAmount, formatHealthFactor, formatRay } from "@/lib/arbor/format";

const BINDEX_BASE = 1000000000000000000000n;
function fmtIndex(v: bigint, base: bigint): string {
  return (Number((v * 1000000n) / base) / 1000000).toFixed(6);
}
import { scaledDebt, computeHealthFactorScaled } from "@/lib/arbor/math";
import { TIER_PARAMS } from "@/lib/arbor/constants";


function FormSection({
  symbol,
  title,
  blurb,
  defaultOpen = false,
  children,
}: {
  symbol: string;
  title: string;
  blurb: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <section className="rounded-2xl border border-white/10 bg-white/[0.02]">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="flex w-full items-center gap-3 px-5 py-4 text-left transition hover:bg-white/[0.03]"
      >
        <AssetIcon symbol={symbol} size={28} className="shrink-0 rounded-full" />
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-semibold text-zinc-100">{title}</span>
          <span className="block truncate text-[11px] text-zinc-500">{blurb}</span>
        </span>
        <svg
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          className={`h-4 w-4 shrink-0 text-zinc-500 transition-transform ${open ? "rotate-180" : ""}`}
        >
          <path d="M5 8l5 5 5-5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>
      {open && (
        <div className="grid gap-4 border-t border-white/5 p-5 md:grid-cols-2">
          {children}
        </div>
      )}
    </section>
  );
}

export default function MarketPage() {
  const params = useParams();

  const marketId =
    typeof params.marketId === "string"
      ? params.marketId
      : Array.isArray(params.marketId)
        ? params.marketId[0]
        : "";

  const { data: marketData, isLoading } = useMarket(marketId || null);
  const wallet = useWalletStore();

  const { data: lenderPosition } = useLenderPosition(
    marketId || null,
    wallet.address
  );

  const { data: borrowerPosition } = useBorrowerPosition(
    marketId || null,
    wallet.address
  );

  const collateralAssetId = marketData?.market.collateralAssetId ?? null;
  const debtAssetId = marketData?.market.debtAssetId ?? null;

  const { data: collateralPrice } = useAssetPrice(collateralAssetId);
  const { data: debtPrice } = useAssetPrice(debtAssetId);

  if (!marketId) {
    return <EmptyState message="Market ID missing." />;
  }

  if (isLoading) {
    return <LoadingSkeleton rows={6} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market, bIndex, supplyIndex, lossFactor, reserveFund } =
    marketData;

  const tier = TIER_PARAMS[market.assetTier];

  const currentDebt = borrowerPosition
    ? scaledDebt(
        borrowerPosition.debtPrincipal,
        bIndex,
        borrowerPosition.borrowIndexAtOpen
      )
    : 0n;

  const pricesAvailable =
    !!collateralPrice?.available &&
    !!debtPrice?.available &&
    collateralPrice.price !== null &&
    debtPrice.price !== null;

  const hf =
    tier && pricesAvailable && borrowerPosition
      ? computeHealthFactorScaled(
          borrowerPosition.collateralQuantity,
          collateralPrice.price as bigint,
          tier.ltvLiqBps,
          currentDebt,
          debtPrice.price as bigint
        )
      : 0n;

  return (
    <div className="space-y-6">
      <div className="rounded-2xl glass backdrop-blur p-5">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-zinc-100">
              {market.marketId}
            </h1>
            <p className="text-xs text-zinc-500">
              <span className="inline-flex flex-wrap items-center gap-2"><AssetIcon symbol={market.collateralAssetId} size={20} className="rounded-full" /> {market.collateralAssetId} collateral / <AssetIcon symbol={market.debtAssetId} size={20} className="rounded-full" /> {market.debtAssetId} debt</span>
            </p>
          </div>

          <StatusPill status={market.status} />
        </div>

        <div className="mt-4 grid grid-cols-2 gap-3 text-xs text-zinc-500 md:grid-cols-4">
          <div>
            Total supplied
            <p className="text-zinc-300">
              {formatAmount(market.totalSupplied, 0)}
            </p>
          </div>

          <div>
            Total borrowed
            <p className="text-zinc-300">
              {formatAmount(market.totalBorrowed, 0)}
            </p>
          </div>

          

          <div>
            Reserve fund
            <p className="text-zinc-300">{formatAmount(reserveFund, 0)}</p>
          </div>

          <div>
            B index
            <p className="text-zinc-300">{fmtIndex(bIndex, BINDEX_BASE)}×</p>
          </div>

          <div>
            S rate
            <p className="text-zinc-300">{formatRay(supplyIndex.sRate)}</p>
          </div>

          <div>
            Loss factor
            <p className="text-zinc-300">{formatRay(lossFactor)}</p>
          </div>

          <div>
            Layer4 pending
            <p className="text-zinc-300">{market.layer4PendingCount}</p>
          </div>
        </div>
      </div>

      <div className="rounded-2xl glass backdrop-blur p-5">
        <h2 className="text-sm font-semibold text-zinc-200">
          Your position
        </h2>

        {!wallet.isConnected && (
          <p className="mt-2 text-xs text-zinc-500">
            Connect a wallet to view positions.
          </p>
        )}

        {wallet.isConnected && (
          <div className="mt-4 grid grid-cols-2 gap-3 text-xs text-zinc-500 md:grid-cols-4">
            <div>
              Lender shares
              <p className="text-zinc-300">
                {formatAmount(lenderPosition?.shares ?? 0n, 0)}
              </p>
            </div>

            <div>
              Collateral
              <p className="text-zinc-300">
                {formatAmount(
                  borrowerPosition?.collateralQuantity ?? 0n,
                  9
                )}
              </p>
            </div>

            <div>
              Debt
              <p className="text-zinc-300">
                {formatAmount(currentDebt, 0)}
              </p>
            </div>

            <div>
              Health factor
              <p className="text-zinc-300">
                {pricesAvailable && borrowerPosition
                  ? formatHealthFactor(hf)
                  : "--"}
              </p>
            </div>
          </div>
        )}
      </div>

      <MarketDetailTabs market={market} reserveFund={reserveFund} lossFactor={lossFactor} />

      <div className="space-y-4">
        <FormSection
          symbol={market.debtAssetId}
          title={`Lend ${market.debtAssetId}`}
          blurb="Supply-side: deposit to earn, withdraw to exit."
          defaultOpen
        >
          <DepositForm marketId={marketId} />
          <WithdrawForm marketId={marketId} />
        </FormSection>

        <FormSection
          symbol={market.collateralAssetId}
          title={`Borrow ${market.debtAssetId}`}
          blurb={`Borrower-side: post ${market.collateralAssetId} as collateral, draw or repay ${market.debtAssetId}.`}
        >
          <DepositCollateralForm marketId={marketId} />
          <WithdrawCollateralForm marketId={marketId} />
          <BorrowForm marketId={marketId} />
          <RepayForm marketId={marketId} />
        </FormSection>

        <FormSection
          symbol="arbor"
          title="Risk & oracle"
          blurb="Liquidations and permissioned price feeds."
        >
          <LiquidateForm marketId={marketId} />
          <UpdatePriceForm marketId={marketId} />
        </FormSection>
      </div>
    </div>
  );
}
