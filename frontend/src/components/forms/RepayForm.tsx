"use client";

import { useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useBorrowerPosition } from "@/lib/hooks/useBorrowerPosition";
import { useWalletStore } from "@/lib/stores/walletStore";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { marketAdmissionFromMarket } from "@/lib/hooks/useMarketAdmission";
import { getBlockedReason } from "@/lib/arbor/admission";
import { parseAmount, formatAmount } from "@/lib/arbor/format";
import { scaledDebt } from "@/lib/arbor/math";
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

export function RepayForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const { data: position } = useBorrowerPosition(marketId, wallet.address);

  const [amount, setAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton rows={2} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market, bIndex } = marketData;
  const admission = marketAdmissionFromMarket(market);
  const blocked = getBlockedReason("repay", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  const currentDebt = position
    ? scaledDebt(
        position.debtPrincipal,
        bIndex,
        position.borrowIndexAtOpen
      )
    : 0n;

  let parseError: string | null = null;

  if (amount.trim()) {
    try {
      const parsed = parseAmount(amount);

      if (parsed > MAX_UINT64) {
        parseError = "Repay amount exceeds uint64 max.";
      } else if (parsed > currentDebt) {
        parseError = "Repay amount exceeds current debt.";
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
        throw new Error("Repay amount must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Repay amount exceeds uint64 max.");
      }

      if (parsed > currentDebt) {
        throw new Error("Repay amount exceeds current debt.");
      }

      const address = addressBytesFromHex(wallet.address);

      await submit("repay", {
        marketId,
        address,
        repayAmount: parsed,
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
          Repay {market.debtAssetId}
        </h3>
        <span className="text-xs text-zinc-500">Borrower</span>
      </div>

      <AdmissionGateBanner txType="repay" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to repay.
        </ErrorText>
      )}

      <div className="text-xs text-zinc-500">
        Current debt:{" "}
        <span className="text-zinc-300">
          {formatAmount(currentDebt, 9)}
        </span>
      </div>

      <Field
        label="Repay amount"
        error={parseError || undefined}
        hint="Repays debt principal plus accrued index-adjusted debt."
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
        Repay
      </SubmitButton>
    </form>
  );
}
