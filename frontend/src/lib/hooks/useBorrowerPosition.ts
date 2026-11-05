"use client";

import { useQuery } from "@tanstack/react-query";
import { getBorrowerPosition } from "@/lib/canopy/pluginRpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";
import type { BorrowerPosition } from "@/lib/arbor/types";

export function useBorrowerPosition(
  marketId: string | null,
  addressHex: string | null
) {
  return useQuery({
    queryKey: ["borrower-position", marketId, addressHex],
    queryFn: async (): Promise<(BorrowerPosition & { currentDebt: bigint }) | null> => {
      if (!marketId || !addressHex) return null;
      return getBorrowerPosition(marketId, addressHex);
    },
    enabled: !!marketId && !!addressHex,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
