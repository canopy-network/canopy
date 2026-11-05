"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";

interface ClaimResult {
  ok: boolean;
  error?: string;
  txHash?: string;
  amountUarb?: string;
  blocksRemaining?: number;
}

export function ArborFaucetCard() {
  const wallet = useWalletStore();
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ClaimResult | null>(null);

  async function handleClaim() {
    if (!wallet.isConnected || !wallet.address) return;
    setBusy(true);
    setResult(null);
    try {
      const res = await fetch("/api/faucet/claim", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ address: wallet.address }),
      });
      const json = (await res.json()) as ClaimResult;
      setResult(json);
    } catch {
      setResult({ ok: false, error: "could not reach faucet service" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
      <h2 className="text-lg font-semibold text-white mb-2">ARBOR faucet</h2>
      <p className="text-sm text-zinc-400 mb-4">
        Claim a small amount of testnet ARBOR from the community pool — no
        gas required. Limited to one claim per wallet every 24 hours.
      </p>
      <button
        type="button"
        onClick={handleClaim}
        disabled={!wallet.isConnected || busy}
        className="w-full rounded-lg bg-emerald-500 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {busy ? "Claiming..." : "Claim ARBOR"}
      </button>
      {!wallet.isConnected && (
        <p className="mt-2 text-xs text-rose-300">Connect a wallet to claim.</p>
      )}
      {result && !result.ok && (
        <p className="mt-3 text-xs text-rose-300">
          {result.error}
          {typeof result.blocksRemaining === "number" &&
            ` — try again in ~${result.blocksRemaining} blocks`}
        </p>
      )}
      {result && result.ok && (
        <p className="mt-3 text-xs text-emerald-300">
          Claimed{" "}
          {result.amountUarb
            ? (Number(result.amountUarb) / 1_000_000).toString()
            : ""}{" "}
          ARBOR — tx {result.txHash?.slice(0, 10)}…
        </p>
      )}
    </div>
  );
}
