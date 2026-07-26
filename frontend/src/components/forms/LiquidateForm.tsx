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
  computeHealthFactorScaled,
  closeFactorBps,
  isLiquidatable,
} from "@/lib/arbor/math";
import { TIER_PARAMS } from "@/lib/arbor/constants";
import { addressBytesFromHex } from "@/lib/arbor/wallet";
import { hexToBytes } from "@/lib/canopy/decode";
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

function parseAddress(hex: string): Uint8Array {
  const clean = hex.startsWith("0x") ? hex.slice(2) : hex;

  if (!/^[0-9a-fA-F]{40}$/.test(clean)) {
    throw new Error("Borrower address must be a 20-byte hex address.");
  }

  return hexToBytes(clean);
}

export function LiquidateForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [borrowerAddress, setBorrowerAddress] = useState("");
  const [repayAmount, setRepayAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const collateralAssetId = marketData?.market.collateralAssetId ?? null;
  const debtAssetId = marketData?.market.debtAssetId ?? null;

  const { data: targetPosition } = useBorrowerPosition(
    marketId,
    borrowerAddress.trim() || null
  );

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
  const blocked =
    getBlockedReason("liquidate_position", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  const tier = TIER_PARAMS[market.assetTier];

  const targetDebt = targetPosition
    ? scaledDebt(
        targetPosition.debtPrincipal,
        bIndex,
        targetPosition.borrowIndexAtOpen
      )
    : 0n;

  const pricesAvailable =
    !!collateralPrice?.available &&
    !!debtPrice?.available &&
    collateralPrice.price !== null &&
    debtPrice.price !== null;

  let hf = 0n;
  let maxRepay = 0n;
  let parseError: string | null = null;

  if (!tier) {
    parseError = "Asset tier parameters unavailable.";
  } else if (borrowerAddress.trim() && targetPosition && pricesAvailable) {
    hf = computeHealthFactorScaled(
      targetPosition.collateralQuantity,
      collateralPrice.price as bigint,
      tier.ltvLiqBps,
      targetDebt,
      debtPrice.price as bigint
    );

    if (!isLiquidatable(hf)) {
      parseError = "Target position is not liquidatable.";
    } else {
      const closeFactor = closeFactorBps(hf);
      maxRepay = (targetDebt * closeFactor) / 10_000n;
    }
  } else if (borrowerAddress.trim() && !targetPosition) {
    parseError = "No borrower position found for that address.";
  } else if (borrowerAddress.trim() && !pricesAvailable) {
    parseError =
      collateralPrice?.reason ||
      debtPrice?.reason ||
      "Oracle prices unavailable.";
  }

  if (repayAmount.trim() && !parseError) {
    try {
      const parsed = parseAmount(repayAmount);

      if (parsed > MAX_UINT64) {
        parseError = "Repay amount exceeds uint64 max.";
      } else if (parsed > maxRepay) {
        parseError = `Repay exceeds close-factor cap: ${formatAmount(maxRepay, 9)}.`;
      }
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const parsed = parseAmount(repayAmount);

      if (parsed <= 0n) {
        throw new Error("Repay amount must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Repay amount exceeds uint64 max.");
      }

      if (parsed > maxRepay) {
        throw new Error(
          `Repay exceeds close-factor cap: ${formatAmount(maxRepay, 9)}.`
        );
      }

      const liquidator = addressBytesFromHex(wallet.address);
      const borrower = parseAddress(borrowerAddress.trim());

      await submit("liquidate_position", {
        marketId,
        liquidator,
        borrowerAddress: borrower,
        repayAmount: parsed,
      });

      setRepayAmount("");
    } catch (err: any) {
      setLocalError(err?.message || String(err));
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="space-y-4 rounded-2xl glass backdrop-blur p-5"
    >
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-200">
          Liquidate position
        </h3>
        <span className="text-xs text-zinc-500">Liquidator</span>
      </div>

      <AdmissionGateBanner txType="liquidate_position" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to liquidate.
        </ErrorText>
      )}

      <Field label="Borrower address">
        <TextInput
          value={borrowerAddress}
          onChange={(e) => setBorrowerAddress(e.target.value)}
          placeholder="0x..."
        />
      </Field>

      <div className="grid grid-cols-2 gap-3 text-xs text-zinc-500">
        <div>
          Target debt:{" "}
          <span className="text-zinc-300">
            {formatAmount(targetDebt, 9)}
          </span>
        </div>
        <div>
          Health factor:{" "}
          <span className="text-zinc-300">
            {targetPosition && pricesAvailable
              ? formatHealthFactor(hf)
              : "--"}
          </span>
        </div>
        <div>
          Max repay:{" "}
          <span className="text-zinc-300">
            {formatAmount(maxRepay, 9)}
          </span>
        </div>
      </div>

      <Field
        label="Repay amount"
        error={parseError || undefined}
        hint="Capped by close factor based on target health factor."
      >
        <TextInput
          value={repayAmount}
          onChange={(e) => setRepayAmount(e.target.value)}
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
        Liquidate
      </SubmitButton>
    </form>
  );
}
