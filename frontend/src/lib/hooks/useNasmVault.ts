"use client";

import { useQuery } from "@tanstack/react-query";
import { getNasmVault, getNasmVaultPool, getAllNasmVaults } from "@/lib/canopy/pluginRpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";

export function useNasmVault(vaultId: string | null) {
  return useQuery({
    queryKey: ["nasm-vault", vaultId],
    queryFn: async () => {
      if (!vaultId) return null;
      const [vault, pool] = await Promise.all([
        getNasmVault(vaultId),
        getNasmVaultPool(vaultId),
      ]);
      if (!vault) return null;
      return { vault, escrowedCollateral: pool?.amount ?? 0n };
    },
    enabled: !!vaultId,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}

export function useNasmVaultsByOwner(ownerHex: string | null) {
  return useQuery({
    queryKey: ["nasm-vaults-by-owner", ownerHex],
    queryFn: async () => {
      if (!ownerHex) return [];
      return getAllNasmVaults(ownerHex);
    },
    enabled: !!ownerHex,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
