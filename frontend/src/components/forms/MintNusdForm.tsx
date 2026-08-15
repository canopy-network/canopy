"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { addressBytesFromHex } from "@/lib/wallet";
import { parseAmount, formatAmount } from "@/lib/arbor/format";
import {
  Field,
  TextInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MAX_UINT64 = 18446744073709551615n;

export function MintNusdForm({ onMinted }: { onMinted?: () => void }) {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [vaultId, setVaultId] = useState("");
  const [collateralAssetId, setCollateralAssetId] = useState("");
  const [collateralQuantity, setCollateralQuantity] = useState("");
  const [nusdAmount, setNusdAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const busy = phase === "signing" || phase === "submitting" || phase === "waiting";

  let parseError: string | null = null;
  let parsedCollateral = 0n;
  let parsedNusd = 0n;

  if (collateralQuantity.trim()) {
    try {
      parsedCollateral = parseAmount(collateralQuantity, 0);
      if (parsedCollateral > MAX_UINT64) parseError = "Collateral quantity exceeds uint64 max.";
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }
  if (!parseError && nusdAmount.trim()) {
    try {
      parsedNusd = parseAmount(nusdAmount, 6);
      if (parsedNusd > MAX_UINT64) parseError = "NUSD amount exceeds uint64 max.";
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      if (!vaultId.trim()) throw new Error("Vault ID is required.");
      if (!collateralAssetId.trim()) throw new Error("Collateral asset ID is required.");

      const collateral = parseAmount(collateralQuantity, 0);
      const nusd = parseAmount(nusdAmount, 6);

      if (collateral <= 0n) throw new Error("Collateral quantity must be greater than zero.");
      if (nusd <= 0n) throw new Error("NUSD amount requested must be greater than zero.");
      if (collateral > MAX_UINT64 || nusd > MAX_UINT64) throw new Error("Amount exceeds uint64 max.");

      const owner = addressBytesFromHex(wallet.address);

      await submit("mint_nusd", {
        vaultId: vaultId.trim(),
        owner,
        collateralAssetId: collateralAssetId.trim(),
        collateralQuantity: collateral,
        nusdAmountRequested: nusd,
      });

      setVaultId("");
      setCollateralAssetId("");
      setCollateralQuantity("");
      setNusdAmount("");
      onMinted?.();
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
        <h3 className="text-sm font-semibold text-zinc-200">Mint NUSD</h3>
        <span className="text-xs text-zinc-500">Opens a new vault</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>Connect a wallet to mint NUSD.</ErrorText>
      )}

      <Field label="Vault ID" hint="A unique identifier for this vault.">
        <TextInput
          value={vaultId}
          onChange={(e) => setVaultId(e.target.value)}
          placeholder="my-vault-01"
        />
      </Field>

      <Field label="Collateral asset ID" hint="Must be N-0 or N-1 tier eligible (see table above).">
        <TextInput
          value={collateralAssetId}
          onChange={(e) => setCollateralAssetId(e.target.value)}
          placeholder="eth"
        />
      </Field>

      <Field
        label="Collateral quantity"
        error={parseError && collateralQuantity.trim() ? parseError : undefined}
      >
        <TextInput
          value={collateralQuantity}
          onChange={(e) => setCollateralQuantity(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
          invalid={!!parseError && !!collateralQuantity.trim()}
        />
      </Field>

      <Field label="NUSD to mint" hint="6-decimal precision.">
        <TextInput
          value={nusdAmount}
          onChange={(e) => setNusdAmount(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
        />
      </Field>

      {parsedCollateral > 0n && parsedNusd > 0n && !parseError && (
        <p className="text-[11px] text-zinc-600">
          Locking {formatAmount(parsedCollateral, 0)} {collateralAssetId || "?"} to mint{" "}
          {formatAmount(parsedNusd, 6)} NUSD.
        </p>
      )}

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Mint NUSD
      </SubmitButton>
    </form>
  );
}
