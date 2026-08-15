"use client";

import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { addressBytesFromHex } from "@/lib/wallet";

export function FaucetCard() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();

  const busy =
    phase === "signing" || phase === "submitting" || phase === "waiting";

  async function handleClaimAll() {
    if (!wallet.isConnected || !wallet.address) return;
    const address = addressBytesFromHex(wallet.address);
    await submit("claim_faucet", { address });
  }

  return (
    <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
      <h2 className="text-lg font-semibold text-white mb-2">Asset faucet</h2>
      <p className="text-sm text-zinc-400 mb-4">
        One claim tops up your BTC, ETH and USDC test balances — they appear
        in “Your balances” above. Cooldowns apply per claim.
      </p>
      <button
        type="button"
        onClick={handleClaimAll}
        disabled={!wallet.isConnected || busy}
        className="w-full rounded-lg bg-indigo-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {busy ? "Claiming..." : "Claim all"}
      </button>
      {!wallet.isConnected && (
        <p className="mt-2 text-xs text-rose-300">Connect a wallet to claim.</p>
      )}
      <p className="mt-3 text-[11px] text-zinc-500">
        ARBOR uses a separate claim mechanism — coming soon.
      </p>
    </div>
  );
}
