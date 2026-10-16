"use client";

import { useQuery } from "@tanstack/react-query";
import { queryAccount } from "@/lib/canopy/rpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";
import type { AccountInfo } from "@/lib/arbor/types";

export function useAccount(address: string | null) {
  return useQuery({
    queryKey: ["account", address],
    queryFn: async (): Promise<AccountInfo | null> => {
      if (!address) return null;
      return queryAccount(address);
    },
    enabled: !!address,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
