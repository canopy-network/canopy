"use client";

import { useState } from "react";
import { useMarket } from "@/lib/hooks/useMarket";
import { useWalletStore } from "@/lib/stores/walletStore";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { parseAmount } from "@/lib/arbor/format";
import { PRICE_DECIMALS } from "@/lib/arbor/constants";
import { addressBytesFromHex } from "@/lib/arbor/wallet";
import { bytesToHex } from "@/lib/canopy/decode";
import { EmptyState } from "@/components/widgets/EmptyState";
import { LoadingSkeleton } from "@/components/widgets/LoadingSkeleton";
import {
  Field,
  TextInput,
  SelectInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MAX_UINT64 = 18446744073709551615n;

export function UpdatePriceForm({ marketId }: { marketId: string }) {
  const { data: marketData, isLoading } = useMarket(marketId);
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [assetId, setAssetId] = useState("");
  const [price, setPrice] = useState("");
  const [confidenceBps, setConfidenceBps] = useState("100");
  const [localError, setLocalError] = useState<string | null>(null);

  if (isLoading) {
    return <LoadingSkeleton rows={3} />;
  }

  if (!marketData) {
    return <EmptyState message="Market not found." />;
  }

  const { market } = marketData;

  const selectedAssetId =
    assetId || market.collateralAssetId || market.debtAssetId;

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let isAuthorized = false;

  try {
    const submitter = addressBytesFromHex(wallet.address);
    const submitterHex = bytesToHex(submitter);

    isAuthorized = (market.authorizedSubmitters || []).some(
      (s) => bytesToHex(s) === submitterHex
    );
  } catch {
    isAuthorized = false;
  }

  let parseError: string | null = null;

  if (price.trim()) {
    try {
      const parsed = parseAmount(price, PRICE_DECIMALS);

      if (parsed <= 0n) {
        parseError = "Price must be greater than zero.";
      } else if (parsed > MAX_UINT64) {
        parseError = "Price exceeds uint64 max.";
      }
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  const confidence = Number(confidenceBps);

  if (
    !Number.isInteger(confidence) ||
    confidence < 0 ||
    confidence > 10000
  ) {
    parseError = "Confidence must be an integer between 0 and 10000 bps.";
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const parsedPrice = parseAmount(price, PRICE_DECIMALS);

      if (parsedPrice <= 0n) {
        throw new Error("Price must be greater than zero.");
      }

      if (parsedPrice > MAX_UINT64) {
        throw new Error("Price exceeds uint64 max.");
      }

      const parsedConfidence = Number(confidenceBps);

      if (
        !Number.isInteger(parsedConfidence) ||
        parsedConfidence < 0 ||
        parsedConfidence > 10000
      ) {
        throw new Error(
          "Confidence must be an integer between 0 and 10000 bps."
        );
      }

      const submitter = addressBytesFromHex(wallet.address);

      await submit("update_price", {
        assetId: selectedAssetId,
        price: parsedPrice,
        confidenceBps: parsedConfidence,
        submitter,
        marketId,
      });

      setPrice("");
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
          Update oracle price
        </h3>
        <span className="text-xs text-zinc-500">Permissioned feed</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to submit prices.
        </ErrorText>
      )}

      {wallet.isConnected && !isAuthorized && (
        <ErrorText>
          Connected wallet is not in this market's authorized_submitters list.
        </ErrorText>
      )}

      <Field label="Asset">
        <SelectInput
          value={selectedAssetId}
          onChange={(e) => setAssetId(e.target.value)}
        >
          <option value={market.collateralAssetId}>
            {market.collateralAssetId} (collateral)
          </option>
          <option value={market.debtAssetId}>
            {market.debtAssetId} (debt)
          </option>
        </SelectInput>
      </Field>

      <Field
        label="Price USD x1e8"
        error={parseError || undefined}
        hint="Example: 1.23456789 = 123456789"
      >
        <TextInput
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          placeholder="0.00000000"
          inputMode="decimal"
          invalid={!!parseError}
        />
      </Field>

      <Field label="Confidence bps">
        <TextInput
          value={confidenceBps}
          onChange={(e) => setConfidenceBps(e.target.value)}
          placeholder="100"
          inputMode="numeric"
        />
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton
        busy={busy}
        disabled={!wallet.isConnected || !isAuthorized || !!parseError}
      >
        Submit price
      </SubmitButton>
    </form>
  );
}
