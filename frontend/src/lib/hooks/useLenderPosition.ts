"use client";

import { useQuery } from "@tanstack/react-query";
import { getLenderPosition } from "@/lib/canopy/pluginRpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";
import type { LenderPosition } from "@/lib/arbor/types";

export function useLenderPosition(
  marketId: string | null,
  addressHex: string | null
) {
  return useQuery({
    queryKey: ["lender-position", marketId, addressHex],
    queryFn: async (): Promise<LenderPosition | null> => {
      if (!marketId || !addressHex) return null;
      return getLenderPosition(marketId, addressHex);
    },
    enabled: !!marketId && !!addressHex,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
