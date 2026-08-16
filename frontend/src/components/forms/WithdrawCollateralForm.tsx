"use client";

import { useEffect, useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useAssetPrice } from "@/lib/hooks/useAssetPrice";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { marketAdmissionFromMarket } from "@/lib/hooks/useMarketAdmission";
import { getBlockedReason } from "@/lib/arbor/admission";
import { parseAmount, formatAmount, formatHealthFactor } from "@/lib/arbor/format";
import {
  scaledDebt,
  computeHealthFactorScaled,
} from "@/lib/arbor/math";
import {
  TIER_PARAMS,
  HF_LIQUIDATABLE_SCALED,
} from "@/lib/arbor/constants";
import { addressBytesFromHex } from "@/lib/wallet";
import { LiveDot } from "@/components/ui/LiveDot";
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

export function WithdrawCollateralForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const { data: position, refetch: refetchPosition } = useBorrowerPosition(marketId, wallet.address);

  useEffect(() => {
    if (phase === "confirmed") {
      refetchPosition();
      const t = setTimeout(() => refetchPosition(), 2500);
      return () => clearTimeout(t);
    }
  }, [phase, refetchPosition]);

  const [quantity, setQuantity] = useState("");
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
  const blocked =
    getBlockedReason("withdraw_collateral", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  const tier = TIER_PARAMS[market.assetTier];

  const currentCollateral = position?.collateralQuantity ?? 0n;

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

  const minCollateralForDebt =
    currentDebt > 0n && pricesAvailable
      ? (currentDebt * (debtPrice.price as bigint) * tier.ltvLiqBps +
          (collateralPrice.price as bigint) * 10000n - 1n) /
        ((collateralPrice.price as bigint) * 10000n)
      : 0n;
  const withdrawableCollateral =
    currentCollateral > minCollateralForDebt
      ? currentCollateral - minCollateralForDebt
      : 0n;

  let previewHf = 0n;
  let parseError: string | null = null;

  if (!tier) {
    parseError = "Asset tier parameters unavailable.";
  } else if (quantity.trim()) {
    try {
      const parsed = parseAmount(quantity, 0);

      if (parsed > MAX_UINT64) {
        parseError = "Quantity exceeds uint64 max.";
      } else if (parsed > currentCollateral) {
        parseError = "Quantity exceeds posted collateral.";
      } else if (currentDebt > 0n && !pricesAvailable) {
        parseError =
          collateralPrice?.reason ||
          debtPrice?.reason ||
          "Oracle prices unavailable.";
      } else {
        const newCollateral = currentCollateral - parsed;

        previewHf = computeHealthFactorScaled(
          newCollateral,
          (collateralPrice?.price as bigint) ?? 0n,
          tier.ltvLiqBps,
          currentDebt,
          (debtPrice?.price as bigint) ?? 0n
        );

        if (previewHf !== 0n && previewHf <= HF_LIQUIDATABLE_SCALED) {
          parseError =
            "Withdrawal would make the position immediately liquidatable.";
        }
      }
    } catch (err: any) {
      const msg = err?.message || String(err);
      parseError = /decimal/i.test(msg)
        ? `Whole units only — ${market.collateralAssetId} amounts on this chain are whole numbers (e.g. 1).`
        : msg;
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const parsed = parseAmount(quantity, 0);

      if (parsed <= 0n) {
        throw new Error("Quantity must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Quantity exceeds uint64 max.");
      }

      if (parsed > currentCollateral) {
        throw new Error("Quantity exceeds posted collateral.");
      }

      const address = addressBytesFromHex(wallet.address);

      await submit("withdraw_collateral", {
        marketId,
        address,
        quantity: parsed,
      });

      setQuantity("");
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
          Withdraw {market.collateralAssetId} collateral
        </h3>
        <span className="text-xs text-zinc-500">Borrower</span>
      </div>

      <AdmissionGateBanner txType="withdraw_collateral" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to withdraw collateral.
        </ErrorText>
      )}

      <div className="grid grid-cols-2 gap-3 text-xs text-zinc-500">
        <div>
          Collateral:{" "}
          <span className="text-zinc-300">
            {formatAmount(currentCollateral, 0)}
          </span>
        </div>
        <div>
          Debt:{" "}
          <span className="text-zinc-300">
            {formatAmount(currentDebt, 0)}
          </span>
        </div>
        <div className="col-span-2">
          Withdrawable now:{" "}
          <span className="text-zinc-300">
            {formatAmount(withdrawableCollateral, 0)} {market.collateralAssetId}
          </span>
          <LiveDot label="Live — depends on current debt and oracle prices; changes without any action from you" />
        </div>
      </div>

      <Field
        label="Quantity"
        error={parseError || undefined}
        hint={
          quantity.trim() && !parseError
            ? `Post-withdrawal health factor: ${formatHealthFactor(previewHf)}`
            : undefined
        }
      >
        <TextInput
          value={quantity}
          onChange={(e) => { setQuantity(e.target.value); setLocalError(null); }}
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
        Withdraw collateral
      </SubmitButton>
    </form>
  );
}
