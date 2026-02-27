import { useQuery } from '@tanstack/react-query';
import { useHistoryCalculation, HistoryResult } from './useHistoryCalculation';
import {useDSFetcher} from "@/core/dsFetch";
import { fetchRewardEventsInRange, sumRewards } from "./stakingRewardsEvents";

/**
 * Hook to calculate validator rewards using block height comparison
 * Fetches reward events and calculates total rewards earned in the last 24h
 */
export function useValidatorRewardsHistory(address?: string) {
    const dsFetch = useDSFetcher();
    const { currentHeight, secondsPerBlock, isReady } = useHistoryCalculation();

    return useQuery({
        queryKey: ['validatorRewardsHistory', address, currentHeight],
        enabled: !!address && isReady,
        staleTime: 30_000,

        queryFn: async (): Promise<HistoryResult> => {
            const { events } = await fetchRewardEventsInRange(dsFetch, {
                address: address || "",
                toHeight: currentHeight,
                secondsPerBlock,
                hours: 24,
                perPage: 100,
                maxPages: 100,
            });

            const rewardsLast24h = sumRewards(events);

            // Return the total as both current and change24h
            // This will display the actual rewards earned in the last 24h
            return {
                current: rewardsLast24h,
                previous24h: 0,
                change24h: rewardsLast24h,
                changePercentage: 0,
                progressPercentage: 100
            };
        }
    });
}
