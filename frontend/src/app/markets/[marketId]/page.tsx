"use client";

import { useParams } from "next/navigation";
import { useMarket } from "@/lib/hooks/useMarket";
import { useWalletStore } from "@/lib/stores/walletStore";
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
import { formatAmount, formatHealthFactor } from "@/lib/arbor/format";
import { scaledDebt, computeHealthFactorScaled } from "@/lib/arbor/math";
import { TIER_PARAMS } from "@/lib/arbor/constants";

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
      <div className="rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur p-5">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-zinc-100">
              {market.marketId}
            </h1>
            <p className="text-xs text-zinc-500">
              {market.collateralAssetId} collateral / {market.debtAssetId} debt
            </p>
          </div>

          <StatusPill status={market.status} />
        </div>

        <div className="mt-4 grid grid-cols-2 gap-3 text-xs text-zinc-500 md:grid-cols-4">
          <div>
            Total supplied
            <p className="text-zinc-300">
              {formatAmount(market.totalSupplied, 9)}
            </p>
          </div>

          <div>
            Total borrowed
            <p className="text-zinc-300">
              {formatAmount(market.totalBorrowed, 9)}
            </p>
          </div>

          

          <div>
            Reserve fund
            <p className="text-zinc-300">{formatAmount(reserveFund, 9)}</p>
          </div>

          <div>
            B index
            <p className="text-zinc-300">{bIndex.toString()}</p>
          </div>

          <div>
            S rate
            <p className="text-zinc-300">{supplyIndex.sRate.toString()}</p>
          </div>

          <div>
            Loss factor
            <p className="text-zinc-300">{lossFactor.toString()}</p>
          </div>

          <div>
            Layer4 pending
            <p className="text-zinc-300">{market.layer4PendingCount}</p>
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur p-5">
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
                {formatAmount(lenderPosition?.shares ?? 0n, 9)}
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
                {formatAmount(currentDebt, 9)}
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

      <div className="grid gap-4 lg:grid-cols-2">
        <DepositForm marketId={marketId} />
        <WithdrawForm marketId={marketId} />
        <DepositCollateralForm marketId={marketId} />
        <WithdrawCollateralForm marketId={marketId} />
        <BorrowForm marketId={marketId} />
        <RepayForm marketId={marketId} />
        <LiquidateForm marketId={marketId} />
        <UpdatePriceForm marketId={marketId} />
      </div>
    </div>
  );
}
