"use client";
import { useMemo } from "react";
import { useMarkets } from "./useMarkets";
import { useWalletStore } from "@/lib/wallet";
import { bytesToHex } from "@/lib/canopy/decode";

// Devnet protocol authority: the single hardcoded address the plugin gates
// pause/resume/deprecate/update_market_params/set_asset_tier to
// (pause_market.go marketPauseAuthority; deprecate/params/asset_tier reuse it).
// Placeholder for the future {22} governance param store (not on-chain yet).
export const PROTOCOL_AUTHORITY = "7961113f844bcf86dfd79570f23a8e3a59b10751";

// Normalize an address (protojson base64 bytes, or a hex string) to lowercase
// hex without 0x, so on-chain bytes compare equal to the wallet's hex string.
function normAddr(x: Uint8Array | string | undefined | null): string {
  if (!x) return "";
  const hex = typeof x === "string" ? x : bytesToHex(x);
  return hex.replace(/^0x/, "").toLowerCase();
}

export interface Roles {
  address: string;
  connected: boolean;
  isProtocolAuthority: boolean;
  oracleFor: string[];
  isPublic: boolean;
}

export function useRoles(): Roles {
  const wallet = useWalletStore();
  const { data: markets } = useMarkets();
  return useMemo(() => {
    const address = normAddr(wallet.address);
    const connected = address.length > 0;
    const isProtocolAuthority = connected && address === PROTOCOL_AUTHORITY;
    const oracleFor: string[] = [];
    if (connected) {
      for (const m of markets ?? []) {
        const subs = m.market.authorizedSubmitters ?? [];
        if (subs.some((s) => normAddr(s) === address)) oracleFor.push(m.market.marketId);
      }
    }
    return { address, connected, isProtocolAuthority, oracleFor, isPublic: true };
  }, [wallet.address, markets]);
}
