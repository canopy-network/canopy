"use client";

import { useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useWalletStore } from "@/lib/stores/walletStore";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { marketAdmissionFromMarket } from "@/lib/hooks/useMarketAdmission";
import { getBlockedReason } from "@/lib/arbor/admission";
import { parseAmount, formatAmount } from "@/lib/arbor/format";
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

export function DepositCollateralForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const { data: position } = useBorrowerPosition(marketId, wallet.address);

  const [quantity, setQuantity] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton rows={2} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market } = marketData;
  const admission = marketAdmissionFromMarket(market);
  const blocked = getBlockedReason("deposit_collateral", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  const currentCollateral = position?.collateralQuantity ?? 0n;

  let parseError: string | null = null;

  if (quantity.trim()) {
    try {
      const parsed = parseAmount(quantity);

      if (parsed > MAX_UINT64) {
        parseError = "Quantity exceeds uint64 max.";
      }
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const parsed = parseAmount(quantity);

      if (parsed <= 0n) {
        throw new Error("Quantity must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Quantity exceeds uint64 max.");
      }

      const address = addressBytesFromHex(wallet.address);

      await submit("deposit_collateral", {
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
      className="space-y-4 rounded-2xl border border-white/10 bg-white/[0.03] backdrop-blur p-5"
    >
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-200">
          Deposit {market.collateralAssetId} collateral
        </h3>
        <span className="text-xs text-zinc-500">Borrower</span>
      </div>

      <AdmissionGateBanner txType="deposit_collateral" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to post collateral.
        </ErrorText>
      )}

      <div className="text-xs text-zinc-500">
        Current collateral:{" "}
        <span className="text-zinc-300">
          {formatAmount(currentCollateral, 9)}
        </span>
      </div>

      <Field
        label="Quantity"
        error={parseError || undefined}
        hint="Native units with 9-decimal convention."
      >
        <TextInput
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
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
        Deposit collateral
      </SubmitButton>
    </form>
  );
}
