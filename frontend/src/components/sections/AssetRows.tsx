"use client";

import { useEffect, useState } from "react";
import { getAssetBalance } from "@/lib/canopy/pluginRpc";
import { AssetIcon } from "@/components/AssetIcon";

export const FAUCET_ASSETS: { id: string; icon: string; name: string; note: string }[] = [
  { id: "BTC", icon: "bitcoin", name: "Bitcoin", note: "Collateral asset" },
  { id: "ETH", icon: "eth", name: "Ether", note: "Collateral asset" },
  { id: "USDC", icon: "usdc", name: "USD Coin", note: "Stablecoin" },
];

export function AssetRows({ address }: { address: string | null }) {
  const [amounts, setAmounts] = useState<Record<string, bigint | null>>({});

  useEffect(() => {
    if (!address) return;
    let alive = true;
    (async () => {
      const entries = await Promise.all(
        FAUCET_ASSETS.map(async (a) => {
          const r = await getAssetBalance(a.id, address);
          return [a.id, r?.amount ?? null] as const;
        })
      );
      if (!alive) return;
      setAmounts(Object.fromEntries(entries));
    })().catch(() => {});
    return () => {
      alive = false;
    };
  }, [address]);

  if (!address) return null;

  return (
    <>
      {FAUCET_ASSETS.map((a) => {
        const amt = amounts[a.id];
        if (amt == null || amt <= 0n) return null;
        return (
          <div key={a.id} className="flex items-center justify-between py-3">
            <div className="flex items-center gap-3">
              <AssetIcon symbol={a.icon} size={36} className="shrink-0 rounded-full shadow-md shadow-black/30" />
              <div>
                <p className="text-sm font-medium text-zinc-100">{a.name}</p>
                <p className="text-[11px] text-zinc-500">{a.note}</p>
              </div>
            </div>
            <p className="text-sm font-semibold tabular-nums text-white">
              {amt.toString()}
            </p>
          </div>
        );
      })}
    </>
  );
}
