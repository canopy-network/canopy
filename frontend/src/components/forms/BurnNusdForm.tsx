"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { useNasmVault } from "@/lib/hooks/useNasmVault";
import { addressBytesFromHex } from "@/lib/wallet";
import { parseAmount, formatAmount } from "@/lib/arbor/format";
import {
  Field,
  TextInput,
  SubmitButton,
  ErrorText,
} from "@/components/forms/FormPrimitives";

const MAX_UINT64 = 18446744073709551615n;

export function BurnNusdForm({ onBurned }: { onBurned?: () => void }) {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [vaultId, setVaultId] = useState("");
  const [nusdAmount, setNusdAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const { data: vaultData } = useNasmVault(vaultId.trim() || null);

  const busy = phase === "signing" || phase === "submitting" || phase === "waiting";

  // [FIX] currentDebt now comes from useNasmVault's live, stability-fee-
  // scaled figure (vaultdebt RPC), not vault.nusdPrincipal. The stale
  // principal undercounts debt by whatever SF has accrued since last
  // touch, so burns submitted against it landed short of on-chain
  // currentDebt -- fullClosure came back false and the vault/pool were
  // left open with residual dust instead of being deleted.
  const currentDebt = vaultData?.currentDebt ?? null;

  let parseError: string | null = null;
  let parsedNusd = 0n;

  if (nusdAmount.trim()) {
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

      const nusd = parseAmount(nusdAmount, 6);
      if (nusd <= 0n) throw new Error("NUSD amount must be greater than zero.");
      if (nusd > MAX_UINT64) throw new Error("NUSD amount exceeds uint64 max.");

      const sender = addressBytesFromHex(wallet.address);

      await submit("burn_nusd", {
        vaultId: vaultId.trim(),
        sender,
        nusdAmount: nusd,
      });

      setNusdAmount("");
      onBurned?.();
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
        <h3 className="text-sm font-semibold text-zinc-200">Burn NUSD</h3>
        <span className="text-xs text-zinc-500">Repay & release collateral</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>Connect the vault owner&apos;s wallet to burn NUSD.</ErrorText>
      )}

      <Field label="Vault ID">
        <TextInput
          value={vaultId}
          onChange={(e) => setVaultId(e.target.value)}
          placeholder="my-vault-01"
        />
      </Field>

      {vaultId.trim() && (
        <p className="text-[11px] text-zinc-600">
          Current debt: {currentDebt !== null ? formatAmount(currentDebt, 6) + " NUSD" : "loading…"}
          {currentDebt !== null && (
            <button
              type="button"
              onClick={() => setNusdAmount(formatAmount(currentDebt, 6))}
              className="ml-2 text-emerald-400 hover:text-emerald-300 underline underline-offset-2"
            >
              Repay full amount
            </button>
          )}
        </p>
      )}

      <Field
        label="NUSD to burn"
        hint="Only debits up to the vault's actual current debt, never more."
        error={
          parseError ||
          (currentDebt !== null && parsedNusd > 0n && parsedNusd < currentDebt
            ? "This amount is below the current debt — the vault will remain open with a residual balance."
            : undefined)
        }
      >
        <TextInput
          value={nusdAmount}
          onChange={(e) => setNusdAmount(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
          invalid={!!parseError}
        />
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Burn NUSD
      </SubmitButton>
    </form>
  );
}
