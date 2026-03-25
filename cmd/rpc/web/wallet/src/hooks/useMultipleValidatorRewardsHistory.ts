import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { useHistoryCalculation, HistoryResult } from './useHistoryCalculation';
import { useDSFetcher } from "@/core/dsFetch";
import { useMemo } from 'react';
import { fetchRewardEventsInRange, sumRewards } from "./stakingRewardsEvents";

/**
 * Hook to calculate rewards for multiple validators
 * Fetches reward events and calculates total rewards earned in the last 24h
 */
export function useMultipleValidatorRewardsHistory(addresses: string[]) {
    const dsFetch = useDSFetcher();
    const { currentHeight, secondsPerBlock, isReady } = useHistoryCalculation();

    // Create stable query key from addresses
    const addressesKey = useMemo(
        () => addresses.sort().join(','),
        [addresses]
    );

    return useQuery({
        queryKey: ['multipleValidatorRewardsHistory', addressesKey, currentHeight],
        enabled: addresses.length > 0 && isReady,
        staleTime: 60_000, // Increased staleTime since rewards don't change rapidly
        refetchInterval: 60_000,
        placeholderData: keepPreviousData,

        queryFn: async (): Promise<Record<string, HistoryResult & { rewards24h: number; totalRewards: number }>> => {
            const results: Record<string, HistoryResult & { rewards24h: number; totalRewards: number }> = {};

            // Fetch rewards for all validators in parallel
            const validatorPromises = addresses.map(async (address) => {
                try {
                    const { events } = await fetchRewardEventsInRange(dsFetch, {
                        address,
                        toHeight: currentHeight,
                        secondsPerBlock,
                        hours: 24,
                        perPage: 100,
                        maxPages: 100,
                    });
                    const rewards24h = sumRewards(events);

                    results[address] = {
                        current: rewards24h,
                        previous24h: 0,
                        change24h: rewards24h,
                        changePercentage: 0,
                        progressPercentage: 100,
                        rewards24h: rewards24h,
                        // With a bounded 24h query we only guarantee 24h totals here.
                        totalRewards: rewards24h
                    };
                } catch (error) {
                    console.error(`Error fetching rewards for ${address}:`, error);
                    results[address] = {
                        current: 0,
                        previous24h: 0,
                        change24h: 0,
                        changePercentage: 0,
                        progressPercentage: 0,
                        rewards24h: 0,
                        totalRewards: 0
                    };
                }
            });

            await Promise.all(validatorPromises);

            return results;
        }
    });
}
