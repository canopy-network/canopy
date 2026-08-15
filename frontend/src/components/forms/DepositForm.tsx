"use client";

import { useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { useAssetBalance } from "@/components/sections/AssetRows";
import { marketAdmissionFromMarket } from "@/lib/hooks/useMarketAdmission";
import { getBlockedReason } from "@/lib/arbor/admission";
import { parseAmount, formatAmount } from "@/lib/arbor/format";
import { expectedSharesMinted } from "@/lib/arbor/math";
import { addressBytesFromHex } from "@/lib/wallet";
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

export function DepositForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();
  const walletBalance = useAssetBalance(
    marketData?.market.debtAssetId ?? null,
    wallet.address
  );

  const [amount, setAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton rows={2} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market, supplyIndex, lossFactor } = marketData;
  const admission = marketAdmissionFromMarket(market);
  const blocked = getBlockedReason("deposit", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let expectedShares = 0n;
  let parseError: string | null = null;

  if (amount.trim()) {
    try {
      const parsed = parseAmount(amount);

      if (parsed > MAX_UINT64) {
        parseError = "Amount exceeds uint64 max.";
      } else {
        expectedShares = expectedSharesMinted(
          parsed,
          supplyIndex.sRate,
          lossFactor
        );
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
        throw new Error("Amount must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Amount exceeds uint64 max.");
      }

      const address = addressBytesFromHex(wallet.address);

      await submit("deposit", {
        marketId,
        address,
        amount: parsed,
      });

      setAmount("");
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
          Deposit {market.debtAssetId}
        </h3>
        <span className="text-xs text-zinc-500">Supply</span>
      </div>

      <AdmissionGateBanner txType="deposit" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to deposit.
        </ErrorText>
      )}

      {wallet.isConnected && walletBalance != null && (
        <p className="text-[11px] text-zinc-500">
          Wallet balance:{" "}
          <span className="text-zinc-300">
            {walletBalance.toString()} {market.debtAssetId}
          </span>
        </p>
      )}

      <Field
        label="Amount"
        error={parseError || undefined}
        hint={`Expected shares: ${formatAmount(expectedShares, 9)}`}
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
        Deposit
      </SubmitButton>
    </form>
  );
}
