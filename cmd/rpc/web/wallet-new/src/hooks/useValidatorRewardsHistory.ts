import { useQuery } from '@tanstack/react-query';
import { useHistoryCalculation, HistoryResult } from './useHistoryCalculation';
import {useDSFetcher} from "@/core/dsFetch";

interface RewardEvent {
    eventType: string;
    msg: {
        amount: number;
    };
    height: number;
    reference: string;
    chainId: number;
    address: string;
}

interface EventsResponse {
    pageNumber: number;
    perPage: number;
    results: RewardEvent[];
    type: string;
    count: number;
    totalPages: number;
    totalCount: number;
}

/**
 * Hook to calculate validator rewards using block height comparison
 * Fetches reward events and calculates total rewards earned in the last 24h
 */
export function useValidatorRewardsHistory(address?: string) {
    const dsFetch = useDSFetcher();
    const { currentHeight, height24hAgo, calculateHistory, isReady } = useHistoryCalculation();

    return useQuery({
        queryKey: ['validatorRewardsHistory', address, currentHeight],
        enabled: !!address && isReady,
        staleTime: 30_000,

        queryFn: async (): Promise<HistoryResult> => {
            // Fetch all reward events
            const events = await dsFetch<RewardEvent[]>('events.byAddress', {
                address,
                height: 0,
                page: 1,
                perPage: 10000 // Large number to get all rewards
            });

            // Filter rewards from the last 24h (between height24hAgo and currentHeight)
            const rewardsLast24h = events
                .filter(event =>
                    event.eventType === 'reward' &&
                    event.height > height24hAgo &&
                    event.height <= currentHeight
                )
                .reduce((sum, event) => sum + (event.msg?.amount || 0), 0);

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
