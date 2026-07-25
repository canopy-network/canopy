"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/stores/walletStore";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { addressBytesFromHex } from "@/lib/arbor/wallet";
import {
  Field,
  TextInput,
  SelectInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MAX_UINT64 = 18446744073709551615n;

type LifecycleAction =
  | "pause_market"
  | "resume_market"
  | "deprecate_market"
  | "update_market_params";

export function MarketLifecycleForm() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [marketId, setMarketId] = useState("");
  const [action, setAction] = useState<LifecycleAction>("pause_market");
  const [reserveFactorBps, setReserveFactorBps] = useState("1000");
  const [localError, setLocalError] = useState<string | null>(null);

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let parseError: string | null = null;

  if (action === "update_market_params") {
    const reserveFactor = Number(reserveFactorBps);

    if (
      !Number.isInteger(reserveFactor) ||
      reserveFactor < 200 ||
      reserveFactor > 3000
    ) {
      parseError = "Reserve factor must be between 200 and 3000 bps.";
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      if (!marketId.trim()) {
        throw new Error("Market ID is required.");
      }

      const authority = addressBytesFromHex(wallet.address);

      if (action === "update_market_params") {
        const parsedReserveFactor = BigInt(reserveFactorBps);

        if (
          parsedReserveFactor < 200n ||
          parsedReserveFactor > 3000n ||
          parsedReserveFactor > MAX_UINT64
        ) {
          throw new Error(
            "Reserve factor must be between 200 and 3000 bps."
          );
        }

        await submit("update_market_params", {
          marketId: marketId.trim(),
          authority,
          reserveFactorBps: parsedReserveFactor,
        });
      } else {
        await submit(action, {
          marketId: marketId.trim(),
          authority,
        });
      }

      setMarketId("");
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
          Market lifecycle
        </h3>
        <span className="text-xs text-zinc-500">Admin</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>
          Connect a wallet with a 20-byte hex address to run lifecycle actions.
        </ErrorText>
      )}

      <Field label="Market ID">
        <TextInput
          value={marketId}
          onChange={(e) => setMarketId(e.target.value)}
          placeholder="CNPY-ETH"
        />
      </Field>

      <Field label="Action">
        <SelectInput
          value={action}
          onChange={(e) => setAction(e.target.value as LifecycleAction)}
        >
          <option value="pause_market">Pause market</option>
          <option value="resume_market">Resume market</option>
          <option value="deprecate_market">Deprecate market</option>
          <option value="update_market_params">
            Update reserve factor
          </option>
        </SelectInput>
      </Field>

      {action === "update_market_params" && (
        <Field
          label="Reserve factor bps"
          error={parseError || undefined}
        >
          <TextInput
            value={reserveFactorBps}
            onChange={(e) => setReserveFactorBps(e.target.value)}
            placeholder="1000"
            inputMode="numeric"
            invalid={!!parseError}
          />
        </Field>
      )}

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Execute
      </SubmitButton>
    </form>
  );
}
