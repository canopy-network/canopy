"use client";

import { useQuery } from "@tanstack/react-query";
import { getAllPrices } from "@/lib/canopy/pluginRpc";
import { useBlockHeight } from "./useChainStatus";
import {
  MIN_REPORTERS,
  DEFAULT_STALENESS_BLOCKS,
  STATE_REFRESH_INTERVAL_MS,
} from "@/lib/arbor/constants";
import type { AssetPriceResult } from "@/lib/arbor/types";

export function useAssetPrice(assetId: string | null | undefined) {
  const { data: height } = useBlockHeight();

  return useQuery({
    queryKey: ["asset-price", assetId, height],
    queryFn: async (): Promise<AssetPriceResult> => {
      if (!assetId) {
        return {
          available: false,
          price: null,
          reporters: 0,
          lastBlock: null,
          reason: "No asset selected.",
        };
      }
      if (height === null || height === undefined) {
        return {
          available: false,
          price: null,
          reporters: 0,
          lastBlock: null,
          reason: "Chain height unavailable.",
        };
      }

      const records = await getAllPrices(assetId);
      if (records.length === 0) {
        return {
          available: false,
          price: null,
          reporters: 0,
          lastBlock: null,
          reason: "No oracle submissions found.",
        };
      }

      const fresh = records.filter((r) => {
        const age = BigInt(height) - r.blockHeight;
        return age >= 0n && age <= BigInt(DEFAULT_STALENESS_BLOCKS) && r.price > 0n;
      });

      if (fresh.length < MIN_REPORTERS) {
        return {
          available: false,
          price: null,
          reporters: fresh.length,
          lastBlock: null,
          reason: `Oracle quorum not met: ${fresh.length}/${MIN_REPORTERS} fresh reporters.`,
        };
      }

      const prices = fresh.map((r) => r.price).sort((a, b) => {
        if (a < b) return -1;
        if (a > b) return 1;
        return 0;
      });
      const mid = Math.floor(prices.length / 2);
      const median =
        prices.length % 2 === 1 ? prices[mid] : (prices[mid - 1] + prices[mid]) / 2n;
      const lastBlock = fresh.reduce(
        (max, r) => (r.blockHeight > max ? r.blockHeight : max),
        0n
      );

      return {
        available: true,
        price: median,
        reporters: fresh.length,
        lastBlock,
        reason: null,
      };
    },
    enabled: !!assetId,
    refetchInterval: STATE_REFRESH_INTERVAL_MS,
    refetchIntervalInBackground: true,
    staleTime: 5_000,
  });
}
