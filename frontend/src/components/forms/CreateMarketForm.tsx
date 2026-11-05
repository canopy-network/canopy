"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { MIN_REPORTERS } from "@/lib/arbor/constants";
import { addressBytesFromHex } from "@/lib/wallet";
import { hexToBytes } from "@/lib/canopy/decode";
import {
  Field,
  TextInput,
  SelectInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MAX_UINT64 = 18446744073709551615n;

function parseAddressList(input: string): Uint8Array[] {
  const parts = input
    .split(/[,\n]+/)
    .map((s) => s.trim())
    .filter(Boolean);

  return parts.map((part) => {
    const clean = part.startsWith("0x") ? part.slice(2) : part;

    if (!/^[0-9a-fA-F]{40}$/.test(clean)) {
      throw new Error(`Invalid 20-byte hex address: ${part}`);
    }

    return hexToBytes(clean);
  });
}

export function CreateMarketForm() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [marketId, setMarketId] = useState("");
  const [collateralAssetId, setCollateralAssetId] = useState("");
  const [debtAssetId, setDebtAssetId] = useState("");
  const [assetTier, setAssetTier] = useState("1");
  const [reserveFactorBps, setReserveFactorBps] = useState("1000");
  const [authorizedSubmitters, setAuthorizedSubmitters] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let parseError: string | null = null;

  const tier = Number(assetTier);
  const reserveFactor = Number(reserveFactorBps);

  if (!Number.isInteger(tier) || tier < 0 || tier > 3) {
    parseError = "Asset tier must be an integer from 0 to 3.";
  } else if (
    !Number.isInteger(reserveFactor) ||
    reserveFactor < 200 ||
    reserveFactor > 3000
  ) {
    parseError = "Reserve factor must be between 200 and 3000 bps.";
  } else if (authorizedSubmitters.trim()) {
    try {
      const submitters = parseAddressList(authorizedSubmitters);

      if (submitters.length < MIN_REPORTERS) {
        parseError = `authorized_submitters must contain at least ${MIN_REPORTERS} addresses.`;
      }
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  } else {
    parseError = `authorized_submitters must contain at least ${MIN_REPORTERS} addresses.`;
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      if (!marketId.trim()) {
        throw new Error("Market ID is required.");
      }

      if (marketId.trim().length > 64) {
        throw new Error("Market ID must be 64 characters or less.");
      }

      if (!collateralAssetId.trim()) {
        throw new Error("Collateral asset ID is required.");
      }

      if (!debtAssetId.trim()) {
        throw new Error("Debt asset ID is required.");
      }

      const parsedTier = Number(assetTier);

      if (!Number.isInteger(parsedTier) || parsedTier < 0 || parsedTier > 3) {
        throw new Error("Asset tier must be an integer from 0 to 3.");
      }

      const parsedReserveFactor = BigInt(reserveFactorBps);

      if (
        parsedReserveFactor < 200n ||
        parsedReserveFactor > 3000n ||
        parsedReserveFactor > MAX_UINT64
      ) {
        throw new Error("Reserve factor must be between 200 and 3000 bps.");
      }

      const submitters = parseAddressList(authorizedSubmitters);

      if (submitters.length < MIN_REPORTERS) {
        throw new Error(
          `authorized_submitters must contain at least ${MIN_REPORTERS} addresses.`
        );
      }

      const creator = addressBytesFromHex(wallet.address);

      await submit("create_market", {
        marketId: marketId.trim(),
        collateralAssetId: collateralAssetId.trim(),
        debtAssetId: debtAssetId.trim(),
        assetTier: parsedTier,
        reserveFactorBps: parsedReserveFactor,
        creator,
        authorizedSubmitters: submitters,
      });

      setMarketId("");
      setCollateralAssetId("");
      setDebtAssetId("");
      setAuthorizedSubmitters("");
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
          Create market
        </h3>
        <span className="text-xs text-zinc-500">Admin</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to create a market.
        </ErrorText>
      )}

      <Field label="Market ID">
        <TextInput
          value={marketId}
          onChange={(e) => setMarketId(e.target.value)}
          placeholder="CNPY-ETH"
        />
      </Field>

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Collateral asset ID">
          <TextInput
            value={collateralAssetId}
            onChange={(e) => setCollateralAssetId(e.target.value)}
            placeholder="CNPY"
          />
        </Field>

        <Field label="Debt asset ID">
          <TextInput
            value={debtAssetId}
            onChange={(e) => setDebtAssetId(e.target.value)}
            placeholder="ETH"
          />
        </Field>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Asset tier">
          <SelectInput
            value={assetTier}
            onChange={(e) => setAssetTier(e.target.value)}
          >
            <option value="0">0 - Tier 0 / CNPY</option>
            <option value="1">1 - Tier 1 / BTC-ETH</option>
            <option value="2">2 - Tier 2 / Majors</option>
            <option value="3">3 - Tier 3 / Restricted</option>
          </SelectInput>
        </Field>

        <Field label="Reserve factor bps">
          <TextInput
            value={reserveFactorBps}
            onChange={(e) => setReserveFactorBps(e.target.value)}
            placeholder="1000"
            inputMode="numeric"
          />
        </Field>
      </div>

      <Field
        label="Authorized submitters"
        error={parseError || undefined}
        hint={`Comma-separated 20-byte hex addresses. Minimum ${MIN_REPORTERS}.`}
      >
        <textarea
          value={authorizedSubmitters}
          onChange={(e) => setAuthorizedSubmitters(e.target.value)}
          rows={3}
          placeholder="0x..., 0x..., 0x..."
          className="w-full rounded-xl glass backdrop-blur px-3 py-2 text-sm text-zinc-100 outline-none placeholder:text-zinc-600 focus:border-indigo-500/60"
        />
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Create market
      </SubmitButton>
    </form>
  );
}
