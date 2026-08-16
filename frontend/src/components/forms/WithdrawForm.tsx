"use client";

import { useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useLenderPosition } from "@/lib/hooks/useLenderPosition";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { marketAdmissionFromMarket } from "@/lib/hooks/useMarketAdmission";
import { getBlockedReason } from "@/lib/arbor/admission";
import { parseAmount, formatAmount } from "@/lib/arbor/format";
import { expectedTokensRedeemed } from "@/lib/arbor/math";
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

export function WithdrawForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const { data: position } = useLenderPosition(marketId, wallet.address);

  const [shares, setShares] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton rows={2} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market, supplyIndex, lossFactor } = marketData;
  const admission = marketAdmissionFromMarket(market);
  const blocked = getBlockedReason("withdraw", admission) !== null;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  const availableShares = position?.shares ?? 0n;

  // Liquidity-aware withdraw limit
  const availableLiquidity =
    market.totalSupplied > market.totalBorrowed
      ? market.totalSupplied - market.totalBorrowed
      : 0n;
  const maxTokensForShares = expectedTokensRedeemed(
    availableShares,
    supplyIndex.sRate,
    lossFactor
  );
  const withdrawableNow =
    maxTokensForShares < availableLiquidity
      ? maxTokensForShares
      : availableLiquidity;
  const liquidityLimited = withdrawableNow < maxTokensForShares;

  let expectedTokens = 0n;
  let parseError: string | null = null;

  if (shares.trim()) {
    try {
      const parsed = parseAmount(shares, 0);

      if (parsed > MAX_UINT64) {
        parseError = "Shares exceed uint64 max.";
      } else if (parsed > availableShares) {
        parseError = "Shares exceed available lender position.";
      } else {
        expectedTokens = expectedTokensRedeemed(
          parsed,
          supplyIndex.sRate,
          lossFactor
        );
        if (expectedTokens > availableLiquidity) {
          parseError = `Exceeds available market liquidity (${formatAmount(availableLiquidity, 0)} ${market.debtAssetId}).`;
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
      const parsed = parseAmount(shares, 0);

      if (parsed <= 0n) {
        throw new Error("Shares must be greater than zero.");
      }

      if (parsed > MAX_UINT64) {
        throw new Error("Shares exceed uint64 max.");
      }

      if (parsed > availableShares) {
        throw new Error("Shares exceed available lender position.");
      }

      const requestedTokens = expectedTokensRedeemed(
        parsed,
        supplyIndex.sRate,
        lossFactor
      );
      if (requestedTokens > availableLiquidity) {
        throw new Error(
          `Exceeds available market liquidity (${formatAmount(availableLiquidity, 0)} ${market.debtAssetId}).`
        );
      }

      const address = addressBytesFromHex(wallet.address);

      await submit("withdraw", {
        marketId,
        address,
        shares: parsed,
      });

      setShares("");
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
          Withdraw {market.debtAssetId}
        </h3>
        <span className="text-xs text-zinc-500">Redeem shares</span>
      </div>

      <AdmissionGateBanner txType="withdraw" status={admission} />

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to withdraw.
        </ErrorText>
      )}

      <Field
        label="Shares"
        error={parseError || undefined}
        hint={`Available: ${formatAmount(availableShares, 0)} shares | Withdrawable now: ${formatAmount(withdrawableNow, 0)} ${market.debtAssetId}${liquidityLimited ? ' (limited by market liquidity)' : ''}`}
      >
        <TextInput
          value={shares}
          onChange={(e) => setShares(e.target.value)}
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
        Withdraw
      </SubmitButton>
    </form>
  );
}
