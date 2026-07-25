"use client";

import { useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { useWalletStore } from "@/lib/stores/walletStore";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { marketAdmissionFromMarket } from "@/lib/hooks/useMarketAdmission";
import { getBlockedReason } from "@/lib/arbor/admission";
import { parseAmount, formatAmount, formatHealthFactor } from "@/lib/arbor/format";
import {
  scaledDebt,
  maxBorrowQty,
  computeHealthFactorScaled,
} from "@/lib/arbor/math";
import {
  TIER_PARAMS,
  HF_LIQUIDATABLE_SCALED,
} from "@/lib/arbor/constants";
import { addressBytesFromHex } from "@/lib/arbor/wallet";
import { EmptyState } from "@/components/widgets/EmptyState";
import { LoadingSkeleton } from "@/components/widgets/LoadingSkeleton";
import { AdmissionGateBanner } from "@/components/widgets/AdmissionGateBanner";
import {
  Field,
  TextInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MAX_UINT64 = 18446744073709551615n;

export function BorrowForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const { data: position } = useBorrowerPosition(marketId, wallet.address);

  const [amount, setAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const collateralAssetId = marketData?.market.collateralAssetId ?? null;
  const debtAssetId = marketData?.market.debtAssetId ?? null;

  const { data: collateralPrice } = useAssetPrice(collateralAssetId);
  const { data: debtPrice } = useAssetPrice(debtAssetId);

  if (isLoading) {
    return <LoadingSkeleton rows={3} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market, bIndex } = marketData;
  const admission = marketAdmissionFromMarket(market);
  const blocked = getBlockedReason("borrow", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  const tier = TIER_PARAMS[market.assetTier];

  const collateralQty = position?.collateralQuantity ?? 0n;

  const currentDebt = position
    ? scaledDebt(
        position.debtPrincipal,
        bIndex,
        position.borrowIndexAtOpen
      )
    : 0n;

  const pricesAvailable =
    !!collateralPrice?.available &&
    !!debtPrice?.available &&
    collateralPrice.price !== null &&
    debtPrice.price !== null;

  const maxBorrow =
    tier && pricesAvailable
      ? maxBorrowQty(
          collateralQty,
          collateralPrice.price as bigint,
          tier.ltvMaxBps,
          debtPrice.price as bigint
        )
      : 0n;

  const remainingBorrow =
    maxBorrow > currentDebt ? maxBorrow - currentDebt : 0n;

  let previewHf = 0n;
  let parseError: string | null = null;

  if (!tier) {
    parseError = "Asset tier parameters unavailable.";
  } else if (!pricesAvailable) {
    parseError =
      collateralPrice?.reason ||
      debtPrice?.reason ||
      "Oracle prices unavailable.";
  } else if (amount.trim()) {
    try {
      const parsed = parseAmount(amount);

      if (parsed > MAX_UINT64) {
        parseError = "Borrow amount exceeds uint64 max.";
      } else if (parsed > remainingBorrow) {
        parseError = `Borrow exceeds max borrowable amount: ${formatAmount(remainingBorrow, 9)}.`;
      } else {
        const newDebt = currentDebt + parsed;

        previewHf = computeHealthFactorScaled(
          collateralQty,
          collateralPrice.price as bigint,
          tier.ltvLiqBps,
          newDebt,
          debtPrice.price as bigint
        );

        if (previewHf !== 0n && previewHf <= HF_LIQUIDATABLE_SCALED) {
          parseError =
            "Borrow would make the position immediately liquidatable.";
        }
      }
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const parsed = parseAmount(amount);

      if (parsed <= 0n) {
        throw new Error("Borrow amount must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Borrow amount exceeds uint64 max.");
      }

      if (parsed > remainingBorrow) {
        throw new Error(
          `Borrow exceeds max borrowable amount: ${formatAmount(remainingBorrow, 9)}.`
        );
      }

      const address = addressBytesFromHex(wallet.address);

      await submit("borrow", {
        marketId,
        address,
        borrowAmount: parsed,
      });

      setAmount("");
    } catch (err: any) {
      setLocalError(err?.message || String(err));
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="space-y-4 rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur p-5"
    >
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-200">
          Borrow {market.debtAssetId}
        </h3>
        <span className="text-xs text-zinc-500">Borrower</span>
      </div>

      <AdmissionGateBanner txType="borrow" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to borrow.
        </ErrorText>
      )}

      <div className="grid grid-cols-2 gap-3 text-xs text-zinc-500">
        <div>
          Current debt:{" "}
          <span className="text-zinc-300">
            {formatAmount(currentDebt, 9)}
          </span>
        </div>
        <div>
          Max borrow:{" "}
          <span className="text-zinc-300">
            {formatAmount(remainingBorrow, 9)}
          </span>
        </div>
      </div>

      <Field
        label="Borrow amount"
        error={parseError || undefined}
        hint={
          amount.trim() && !parseError
            ? `Post-borrow health factor: ${formatHealthFactor(previewHf)}`
            : undefined
        }
      >
        <TextInput
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
          invalid={!!parseError}
        />
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton
        busy={busy}
        disabled={!wallet.isConnected || blocked || !!parseError}
      >
        Borrow
      </SubmitButton>
    </form>
  );
}
