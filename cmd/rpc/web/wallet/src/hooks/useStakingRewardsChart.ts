import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useDSFetcher } from "@/core/dsFetch";
import { useHistoryCalculation } from "./useHistoryCalculation";
import {
  fetchRewardEventsInRange,
  sumRewards,
  type RewardEvent,
} from "./stakingRewardsEvents";

export interface RewardChartPoint {
  index: number;
  label: string;
  fromHeight: number;
  toHeight: number;
  amount: number;
  cumulative: number;
  timestampMs: number;
}

export interface StakingRewardsChartResult {
  address: string;
  targetHeight: number;
  fromHeight: number;
  blocksInRange: number;
  totalRewards24h: number;
  eventsCount: number;
  points: RewardChartPoint[];
}

interface UseStakingRewardsChartOptions {
  address?: string;
  height?: number;
  hours?: number;
  points?: number;
  perPage?: number;
  maxPages?: number;
  enabled?: boolean;
  staleTimeMs?: number;
  refetchIntervalMs?: number;
}

export function useStakingRewardsChart({
  address,
  height,
  hours = 24,
  points = 12,
  perPage = 100,
  maxPages = 100,
  enabled = true,
  staleTimeMs = 60_000,
  refetchIntervalMs = 60_000,
}: UseStakingRewardsChartOptions = {}) {
  const dsFetch = useDSFetcher();
  const { currentHeight, secondsPerBlock, isReady } = useHistoryCalculation();

  const targetHeight = useMemo(
    () => (typeof height === "number" && height > 0 ? height : currentHeight),
    [height, currentHeight]
  );

  return useQuery({
    queryKey: [
      "stakingRewardsChart",
      address,
      targetHeight,
      hours,
      points,
      perPage,
      maxPages,
    ],
    enabled: enabled && !!address && isReady && targetHeight > 0,
    staleTime: staleTimeMs,
    refetchInterval: refetchIntervalMs,
    queryFn: async (): Promise<StakingRewardsChartResult> => {
      const safeAddress = address || "";
      const safePoints = Math.max(2, Math.floor(points));

      const { events, fromHeight, toHeight, blocksInRange } =
        await fetchRewardEventsInRange(dsFetch, {
          address: safeAddress,
          toHeight: targetHeight,
          secondsPerBlock,
          hours,
          perPage,
          maxPages,
        });

      const totalRewards24h = sumRewards(events);

      const heightSpan = Math.max(1, toHeight - fromHeight + 1);
      const bucketSize = Math.max(1, Math.ceil(heightSpan / safePoints));
      const nowMs = Date.now();
      const blocksToMs = (blocks: number) => blocks * secondsPerBlock * 1000;

      const chartPoints: RewardChartPoint[] = [];
      let cumulative = 0;

      for (let i = 0; i < safePoints; i += 1) {
        const bucketFrom = fromHeight + i * bucketSize;
        const bucketTo =
          i === safePoints - 1
            ? toHeight
            : Math.min(toHeight, bucketFrom + bucketSize - 1);

        const amount = events
          .filter((event: RewardEvent) => event.height >= bucketFrom && event.height <= bucketTo)
          .reduce((sum, event) => sum + (event.msg?.amount || 0), 0);

        cumulative += amount;

        const blocksAgo = Math.max(0, toHeight - bucketTo);
        const hoursAgo = Math.max(
          0,
          Math.round((blocksAgo * secondsPerBlock) / (60 * 60))
        );
        const label = i === safePoints - 1 ? "Now" : `${hoursAgo}h`;

        chartPoints.push({
          index: i,
          label,
          fromHeight: bucketFrom,
          toHeight: bucketTo,
          amount,
          cumulative,
          timestampMs: nowMs - blocksToMs(blocksAgo),
        });
      }

      return {
        address: safeAddress,
        targetHeight: toHeight,
        fromHeight,
        blocksInRange,
        totalRewards24h,
        eventsCount: events.length,
        points: chartPoints,
      };
    },
  });
}
