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

export function LiquidateNasmVaultForm({ onLiquidated }: { onLiquidated?: () => void }) {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const [vaultId, setVaultId] = useState("");
  const [repayAmount, setRepayAmount] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  const { data: vaultData } = useNasmVault(vaultId.trim() || null);

  const busy = phase === "signing" || phase === "submitting" || phase === "waiting";

  const vaultRecord = vaultData?.vault as Record<string, unknown> | undefined;
  const currentDebt = vaultRecord
    ? BigInt(String(vaultRecord.nusdPrincipal ?? "0"))
    : null;
  const escrowedCollateral = vaultData?.escrowedCollateral ?? null;

  let parseError: string | null = null;
  let parsedRepay = 0n;

  if (repayAmount.trim()) {
    try {
      parsedRepay = parseAmount(repayAmount, 6);
      if (parsedRepay > MAX_UINT64) parseError = "Repay amount exceeds uint64 max.";
    } catch (err: any) {
      parseError = err?.message || String(err);
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLocalError(null);

    try {
      if (!vaultId.trim()) throw new Error("Vault ID is required.");

      const repay = parseAmount(repayAmount, 6);
      if (repay <= 0n) throw new Error("Repay amount must be greater than zero.");
      if (repay > MAX_UINT64) throw new Error("Repay amount exceeds uint64 max.");

      const liquidator = addressBytesFromHex(wallet.address);

      await submit("liquidate_nasm_vault", {
        vaultId: vaultId.trim(),
        liquidator,
        repayAmount: repay,
      });

      setRepayAmount("");
      onLiquidated?.();
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
        <h3 className="text-sm font-semibold text-zinc-200">Liquidate NASM vault</h3>
        <span className="text-xs text-zinc-500">Liquidator</span>
      </div>

      {!wallet.isConnected && (
        <ErrorText>Connect a wallet holding NUSD to liquidate a vault.</ErrorText>
      )}

      <Field
        label="Vault ID"
        hint="Only undercollateralized vaults (HF_n ≤ 1.0) can be liquidated; the chain enforces this at submit."
      >
        <TextInput
          value={vaultId}
          onChange={(e) => setVaultId(e.target.value)}
          placeholder="my-vault-01"
        />
      </Field>

      {vaultId.trim() && (
        <div className="grid grid-cols-2 gap-3 text-xs text-zinc-500">
          <div>
            Vault debt:{" "}
            <span className="text-zinc-300">
              {currentDebt !== null ? formatAmount(currentDebt, 6) + " NUSD" : "loading…"}
            </span>
          </div>
          <div>
            Escrowed collateral:{" "}
            <span className="text-zinc-300">
              {escrowedCollateral !== null ? formatAmount(escrowedCollateral, 9) : "loading…"}
            </span>
          </div>
        </div>
      )}

      <Field
        label="Repay amount (NUSD)"
        hint="Debited from your own NUSD balance. Bad-debt liquidations that exceed locked collateral are rejected on-chain."
        error={parseError || undefined}
      >
        <TextInput
          value={repayAmount}
          onChange={(e) => setRepayAmount(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
          invalid={!!parseError}
        />
      </Field>

      {localError && <ErrorText>{localError}</ErrorText>}

      <SubmitButton busy={busy} disabled={!wallet.isConnected || !!parseError}>
        Liquidate vault
      </SubmitButton>
    </form>
  );
}
