"use client";

import { useQuery } from "@tanstack/react-query";
import { queryHeight, checkConnection } from "@/lib/canopy/rpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";

export function useChainStatus() {
  return useQuery({
    queryKey: ["chain-status"],
    queryFn: async () => {
      const [connected, height] = await Promise.all([
        checkConnection(),
        queryHeight(),
      ]);

      return {
        connected,
        height,
      };
    },
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}

export function useBlockHeight() {
  return useQuery({
    queryKey: ["block-height"],
    queryFn: queryHeight,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
