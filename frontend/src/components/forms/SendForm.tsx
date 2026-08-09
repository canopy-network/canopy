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

export function SendForm() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [toAddress, setToAddress] = useState("");
  const [amount, setAmount] = useState("");
  const [vestingStartHeight, setVestingStartHeight] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const busy =
    phase === "signing" ||
    phase === "submitting" ||
    phase === "waiting";

  let parseError: string | null = null;
  let parsedAmount = 0n;

  if (amount.trim()) {
    try {
      parsedAmount = parseAmount(amount);
      if (parsedAmount > MAX_UINT64) parseError = "Amount exceeds uint64 max.";
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      const cleanTo = toAddress.trim().replace(/^0x/, "");
      if (cleanTo.length !== 40) {
        throw new Error("Recipient address must be 20 bytes (40 hex chars).");
      }

      const parsed = parseAmount(amount);
      if (parsed <= 0n) throw new Error("Amount must be greater than zero.");
      if (parsed > MAX_UINT64) throw new Error("Amount exceeds uint64 max.");

      let vesting = 0n;
      if (vestingStartHeight.trim()) {
        vesting = BigInt(vestingStartHeight.trim());
        if (vesting > MAX_UINT64) throw new Error("Vesting start height exceeds uint64 max.");
      }

      const fromAddress = addressBytesFromHex(wallet.address);
      const toAddressBytes = addressBytesFromHex(cleanTo);

      await submit("send", {
        fromAddress,
        toAddress: toAddressBytes,
        amount: parsed,
        vestingStartHeight: vesting,
      });

      setToAddress("");
      setAmount("");
      setVestingStartHeight("");
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
        <h3 className="text-sm font-semibold text-zinc-200">Send CNPY</h3>
        <span className="text-xs text-zinc-500">Wallet transfer</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>Connect a wallet to send CNPY.</ErrorText>
      )}

      <Field label="Recipient address" hint="20-byte hex address (40 chars, no 0x needed).">
        <TextInput
          value={toAddress}
          onChange={(e) => setToAddress(e.target.value)}
          placeholder="7961113f844bcf86dfd79570f23a8e3a59b10751"
          className="font-mono"
        />
      </Field>

      <Field
        label="Amount (CNPY)"
        error={parseError || undefined}
      >
        <TextInput
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
          invalid={!!parseError}
        />
      </Field>

      <Field
        label="Vesting start height (optional)"
        hint="Leave blank for an immediate, unvested transfer."
      >
        <TextInput
          value={vestingStartHeight}
          onChange={(e) => setVestingStartHeight(e.target.value)}
          placeholder="0"
          inputMode="numeric"
        />
      </Field>

      {parsedAmount > 0n && !parseError && (
        <p className="text-[11px] text-zinc-600">
          Sending {formatAmount(parsedAmount, 9)} CNPY.
        </p>
      )}

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Send
      </SubmitButton>
    </form>
  );
}
