"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { addressBytesFromHex } from "@/lib/wallet";
import {
  Field,
  TextInput,
  SelectInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

export function SetAssetTierForm() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [assetId, setAssetId] = useState("");
  const [tier, setTier] = useState("1");
  const [localError, setLocalError] = useState<string | null>(null);

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let parseError: string | null = null;

  const parsedTier = Number(tier);

  if (!Number.isInteger(parsedTier) || parsedTier < 0 || parsedTier > 3) {
    parseError = "Asset tier must be an integer from 0 to 3.";
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      if (!assetId.trim()) {
        throw new Error("Asset ID is required.");
      }

      const parsed = Number(tier);

      if (!Number.isInteger(parsed) || parsed < 0 || parsed > 3) {
        throw new Error("Asset tier must be an integer from 0 to 3.");
      }

      const authority = addressBytesFromHex(wallet.address);

      await submit("set_asset_tier", {
        assetId: assetId.trim(),
        tier: parsed,
        authority,
      });

      setAssetId("");
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
          Set asset tier
        </h3>
        <span className="text-xs text-zinc-500">Admin</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to set asset tiers.
        </ErrorText>
      )}

      <Field label="Asset ID">
        <TextInput
          value={assetId}
          onChange={(e) => setAssetId(e.target.value)}
          placeholder="CNPY"
        />
      </Field>

      <Field label="Tier" error={parseError || undefined}>
        <SelectInput
          value={tier}
          onChange={(e) => setTier(e.target.value)}
          invalid={!!parseError}
        >
          <option value="0">0 - Tier 0 / CNPY</option>
          <option value="1">1 - Tier 1 / BTC-ETH</option>
          <option value="2">2 - Tier 2 / Majors</option>
          <option value="3">3 - Tier 3 / Restricted</option>
        </SelectInput>
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Set tier
      </SubmitButton>
    </form>
  );
}
