"use client";

import { useQuery } from "@tanstack/react-query";
import {
  getMarket,
  getReserveFund,
  getLossFactor,
} from "@/lib/canopy/pluginRpc";
import { RAY, STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";
import type { Market, SupplyIndexRecord } from "@/lib/arbor/types";

export interface MarketDetail {
  market: Market;
  bIndex: bigint;
  supplyIndex: SupplyIndexRecord;
  lossFactor: bigint;
  reserveFund: bigint;
}

export function useMarket(marketId: string | null) {
  return useQuery({
    queryKey: ["market", marketId],
    queryFn: async (): Promise<MarketDetail | null> => {
      if (!marketId) return null;
      const market = await getMarket(marketId);
      if (!market) return null;

      const [reserveFund, lossFactor] = await Promise.all([
        getReserveFund(marketId),
        getLossFactor(marketId),
      ]);

      return {
        market,
        bIndex: RAY,
        supplyIndex: { sRate: RAY, totalSharesOutstanding: 0n },
        lossFactor: lossFactor ?? RAY,
        reserveFund,
      };
    },
    enabled: !!marketId,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
