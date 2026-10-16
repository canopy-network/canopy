"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useWalletStore } from "@/lib/wallet";
import { useTxStore } from "@/lib/stores/txStore";
import { signAndSubmitArborTx } from "@/lib/tx";
import type { ArborTxType } from "@/lib/arbor/constants";

export function useArborTx() {
  const queryClient = useQueryClient();
  const wallet = useWalletStore();
  const txStore = useTxStore();

  async function submit(
    txType: ArborTxType,
    msg: unknown,
    fee?: bigint
  ): Promise<{ txHash: string; height: number }> {
    if (!wallet.isConnected) {
      throw new Error("Wallet not connected.");
    }

    if (!wallet.privateKeyHex) {
      throw new Error("Private key not loaded.");
    }

    if (!wallet.publicKeyHex) {
      throw new Error("Public key not loaded.");
    }

    txStore.start();

    try {
      txStore.setPhase("submitting");

      const result = await signAndSubmitArborTx({
        txType,
        msg,
        privateKeyHex: wallet.privateKeyHex,
        publicKeyHex: wallet.publicKeyHex,
        fee,
      });

      txStore.setTxHash(result.txHash);
      txStore.setConfirmed(result.height);

      await queryClient.invalidateQueries();

      return result;
    } catch (err: any) {
      txStore.setError(err?.message || String(err));
      throw err;
    }
  }

  return {
    submit,
    reset: txStore.reset,
    phase: txStore.phase,
    txHash: txStore.txHash,
    error: txStore.error,
    blockHeight: txStore.blockHeight,
  };
}
