import { useDS } from "@/core/useDs";
import { useMemo } from "react";

interface RewardEvent {
  type: string;
  msg: {
    amount: number;
  };
  height: number;
  time: string;
  ref: string;
  chainId: string;
  indexedAddress: string;
}

interface RewardsData {
  totalRewards: number;
  rewardEvents: RewardEvent[];
  last24hRewards: number;
  last7dRewards: number;
  averageRewardPerBlock: number;
}

export const useValidatorRewards = (address?: string) => {
  const {
    data: events = [],
    isLoading,
    error,
  } = useDS<RewardEvent[]>(
    "events.byAddress",
    { address: address || "", page: 1, perPage: 1000 },
    {
      enabled: !!address,
      select: (data) => {
        // Filter only reward events
        if (Array.isArray(data)) {
          return data.filter((event: any) => event.type === "reward");
        }
        return [];
      },
    },
  );

  const rewardsData = useMemo<RewardsData>(() => {
    if (!events || events.length === 0) {
      return {
        totalRewards: 0,
        rewardEvents: [],
        last24hRewards: 0,
        last7dRewards: 0,
        averageRewardPerBlock: 0,
      };
    }

    const now = Date.now();
    const oneDayAgo = now - 24 * 60 * 60 * 1000;
    const sevenDaysAgo = now - 7 * 24 * 60 * 60 * 1000;

    let totalRewards = 0;
    let last24hRewards = 0;
    let last7dRewards = 0;

    events.forEach((event: any) => {
      const amount = event.msg?.amount || 0;
      totalRewards += amount;

      const eventTime = new Date(event.time).getTime();
      if (eventTime >= oneDayAgo) {
        last24hRewards += amount;
      }
      if (eventTime >= sevenDaysAgo) {
        last7dRewards += amount;
      }
    });

    const averageRewardPerBlock =
      events.length > 0 ? totalRewards / events.length : 0;

    return {
      totalRewards,
      rewardEvents: events,
      last24hRewards,
      last7dRewards,
      averageRewardPerBlock,
    };
  }, [events]);

  return {
    ...rewardsData,
    isLoading,
    error,
  };
};
