"use client";

import { useQuery } from "@tanstack/react-query";
import {
  getAllMarkets,
  getReserveFund,
  getLossFactor,
} from "@/lib/canopy/pluginRpc";
import { decodeUint128 } from "@/lib/canopy/decode";
import { RAY, STATE_REFRESH_INTERVAL_MS } from "@/lib/arbor/constants";
import type { Market, SupplyIndexRecord } from "@/lib/arbor/types";

export interface MarketWithIndices {
  market: Market;
  bIndex: bigint;
  supplyIndex: SupplyIndexRecord;
  lossFactor: bigint;
  reserveFund: bigint;
}

function b64(v: string | null | undefined): Uint8Array {
  if (!v) return new Uint8Array(0);
  if (typeof Buffer !== "undefined") {
    return new Uint8Array(Buffer.from(v, "base64"));
  }
  const bin = atob(v);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function toBig(v: unknown): bigint {
  if (v === null || v === undefined || v === "") return 0n;
  try {
    return BigInt(String(v));
  } catch {
    return 0n;
  }
}

function toNum(v: unknown): number {
  if (v === null || v === undefined || v === "") return 0;
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function parseMarket(j: any): Market {
  return {
    marketId: String(j?.marketId ?? ""),
    collateralAssetId: String(j?.collateralAssetId ?? ""),
    debtAssetId: String(j?.debtAssetId ?? ""),
    assetTier: toNum(j?.assetTier),
    status:
      j?.status === "PAUSED" ||
      j?.status === "INSOLVENT" ||
      j?.status === "DEPRECATED" ||
      j?.status === "ACTIVE"
        ? j.status
        : "ACTIVE",
    indexOverflowHalted: Boolean(j?.indexOverflowHalted),
    totalBorrowed: toBig(j?.totalBorrowed),
    totalSupplied: toBig(j?.totalSupplied),
    reserveFactorBps: toBig(j?.reserveFactorBps),
    lastAccrualBlock: toBig(j?.lastAccrualBlock),
    layer4PendingCount: toNum(j?.layer4PendingCount),
    layer4PendingBadDebtTotal: j?.layer4PendingBadDebtTotal
      ? decodeUint128(b64(j.layer4PendingBadDebtTotal))
      : 0n,
    creator: j?.creator ? b64(j.creator) : undefined,
    authorizedSubmitters: Array.isArray(j?.authorizedSubmitters)
      ? j.authorizedSubmitters.map((s: string) => b64(s))
      : undefined,
  };
}

export function useMarkets() {
  return useQuery({
    queryKey: ["markets"],
    queryFn: async (): Promise<MarketWithIndices[]> => {
      const raw = await getAllMarkets();
      const markets = (Array.isArray(raw) ? raw : []).map(parseMarket);

      return Promise.all(
        markets.map(async (market) => {
          const [reserveFund, lossFactor] = await Promise.all([
            getReserveFund(market.marketId),
            getLossFactor(market.marketId),
          ]);
          return {
            market,
            bIndex: RAY,
            supplyIndex: { sRate: RAY, totalSharesOutstanding: 0n },
            lossFactor: lossFactor ?? RAY,
            reserveFund,
          };
        })
      );
    },
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    staleTime: 20_000,
  });
}
