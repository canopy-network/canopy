"use client";

import { useQuery } from "@tanstack/react-query";
import { getPool } from "@/lib/canopy/pluginRpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";

export function useMarketPools(marketId: string | null) {
  return useQuery({
    queryKey: ["market-pools", marketId],
    queryFn: async (): Promise<{ supply: bigint; collateral: bigint } | null> => {
      if (!marketId) return null;
      const [supply, collateral] = await Promise.all([
        getPool(marketId, "supply"),
        getPool(marketId, "collateral"),
      ]);
      return {
        supply: supply?.amount ?? 0n,
        collateral: collateral?.amount ?? 0n,
      };
    },
    enabled: !!marketId,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
