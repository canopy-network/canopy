"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { addressBytesFromHex } from "@/lib/wallet";
import {
  Field,
  TextInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MIN_BPS = 25;
const MAX_BPS = 150;

export function SetTreasuryCutForm() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [bps, setBps] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let parseError: string | null = null;
  const parsedBps = Number(bps);

  if (bps.trim()) {
    if (!Number.isInteger(parsedBps) || parsedBps < MIN_BPS || parsedBps > MAX_BPS) {
      parseError = `Treasury cut must be an integer from ${MIN_BPS} to ${MAX_BPS} bps (0.25%-1.5%).`;
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const parsed = Number(bps);

      if (!Number.isInteger(parsed) || parsed < MIN_BPS || parsed > MAX_BPS) {
        throw new Error(`Treasury cut must be an integer from ${MIN_BPS} to ${MAX_BPS} bps (0.25%-1.5%).`);
      }

      const authority = addressBytesFromHex(wallet.address);

      await submit("set_treasury_cut", {
        authority,
        treasuryCutBps: BigInt(parsed),
      });

      setBps("");
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
          Set treasury cut
        </h3>
        <span className="text-xs text-zinc-500">Admin · global</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to set the treasury cut.
        </ErrorText>
      )}

      <Field
        label="Treasury cut (bps)"
        hint="0.25%-1.5% of each market's interest routed to Arbor's treasury on accrual. Global, not per-market."
        error={parseError || undefined}
      >
        <TextInput
          value={bps}
          onChange={(e) => setBps(e.target.value)}
          placeholder="100"
          inputMode="numeric"
          invalid={!!parseError}
        />
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Set treasury cut
      </SubmitButton>
    </form>
  );
}
