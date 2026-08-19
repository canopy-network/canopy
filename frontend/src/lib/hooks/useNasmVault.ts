"use client";

import { useQuery } from "@tanstack/react-query";
import {
  getNasmVault,
  getNasmVaultPool,
  getAllNasmVaults,
  getNasmTierBacking,
  getVaultDebt,
} from "@/lib/canopy/pluginRpc";
import { STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";

export function useNasmVault(vaultId: string | null) {
  return useQuery({
    queryKey: ["nasm-vault", vaultId],
    queryFn: async () => {
      if (!vaultId) return null;
      const [vault, pool, debt] = await Promise.all([
        getNasmVault(vaultId),
        getNasmVaultPool(vaultId),
        getVaultDebt(vaultId),
      ]);
      if (!vault) return null;
      // [FIX] currentDebt is the live, SF-scaled figure -- use this for
      // display and for burn_nusd amounts. Fall back to the vault's raw
      // nusdPrincipal only if the vaultdebt route is unreachable, since
      // showing nothing is worse than a slightly stale number here.
      return {
        vault,
        escrowedCollateral: pool?.amount ?? 0n,
        currentDebt: debt?.currentDebt ?? null,
      };
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

// NASM Spec Section 3.3's per-tier mint concentration cap accumulator.
// No vaultId scoping -- {36} is a single global record.
export function useNasmTierBacking() {
  return useQuery({
    queryKey: ["nasm-tier-backing"],
    queryFn: getNasmTierBacking,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
