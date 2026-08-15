"use client";

import { useState } from "react";
import { useWalletStore } from "@/lib/wallet";
import { useArborTx } from "@/lib/hooks/useArborTx";
import { addressBytesFromHex } from "@/lib/wallet";
import { getAssetBalance } from "@/lib/canopy/pluginRpc";
import { useQuery } from "@tanstack/react-query";

const FAUCET_ASSETS = [
  { id: "ARBOR", name: "Arbor", note: "Native token (6-dec)" },
  { id: "BTC", name: "Bitcoin", note: "Collateral asset" },
  { id: "ETH", name: "Ether", note: "Collateral asset" },
  { id: "USDC", name: "USD Coin", note: "Stablecoin" },
];

export function FaucetCard() {
  const wallet = useWalletStore();
  const { submit, phase } = useArborTx();
  const [claimingAsset, setClaimingAsset] = useState<string | null>(null);

  const busy = phase === "signing" || phase === "submitting" || phase === "waiting";

  async function handleClaim(assetId: string) {
    if (!wallet.isConnected || !wallet.address) return;
    setClaimingAsset(assetId);
    try {
      const address = addressBytesFromHex(wallet.address);
      await submit("claim_faucet", { address });
      setClaimingAsset(null);
    } catch (err: any) {
      setClaimingAsset(null);
      // Error surfaces via the toast system
    }
  }

  return (
    <div className="rounded-2xl border border-white/5 bg-white/[0.02] p-6 backdrop-blur-sm">
      <h2 className="text-lg font-semibold text-white mb-2">Asset faucet</h2>
      <p className="text-sm text-zinc-400 mb-4">
        Claim test assets to explore lending, borrowing, and NASM vaults.
      </p>
      <div className="grid grid-cols-2 gap-3">
        {FAUCET_ASSETS.map((asset) => (
          <FaucetButton
            key={asset.id}
            asset={asset}
            walletAddress={wallet.address}
            onClaim={() => handleClaim(asset.id)}
            busy={busy || claimingAsset === asset.id}
            isConnected={wallet.isConnected}
          />
        ))}
      </div>
    </div>
  );
}

function FaucetButton({
  asset,
  walletAddress,
  onClaim,
  busy,
  isConnected,
}: {
  asset: { id: string; name: string; note: string };
  walletAddress: string | null;
  onClaim: () => void;
  busy: boolean;
  isConnected: boolean;
}) {
  const { data: balance } = useQuery({
    queryKey: ["assetBalance", asset.id, walletAddress],
    queryFn: async () => {
      if (!walletAddress) return null;
      const result = await getAssetBalance(asset.id, walletAddress);
      return result?.amount ?? null;
    },
    enabled: !!walletAddress,
    refetchInterval: 10000,
  });

  return (
    <div className="rounded-lg border border-white/5 bg-white/[0.03] p-4">
      <div className="flex items-start justify-between mb-2">
        <div>
          <p className="text-sm font-medium text-zinc-100">{asset.name}</p>
          <p className="text-[11px] text-zinc-500">{asset.note}</p>
        </div>
        {balance != null && balance > 0n && (
          <p className="text-xs font-semibold tabular-nums text-zinc-300">
            {balance.toString()}
          </p>
        )}
      </div>
      <button
        type="button"
        onClick={onClaim}
        disabled={!isConnected || busy}
        className="w-full rounded-lg bg-indigo-500 px-3 py-2 text-xs font-medium text-white transition hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {busy ? "Claiming..." : "Claim"}
      </button>
    </div>
  );
}
