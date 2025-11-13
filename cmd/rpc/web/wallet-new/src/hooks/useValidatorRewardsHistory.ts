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
 * Fetches reward events at current height and 24h ago to calculate the difference
 */
export function useValidatorRewardsHistory(address?: string) {
    const dsFetch = useDSFetcher();
    const { currentHeight, height24hAgo, calculateHistory, isReady } = useHistoryCalculation();

    return useQuery({
        queryKey: ['validatorRewardsHistory', address, currentHeight],
        enabled: !!address && isReady,
        staleTime: 30_000,

        queryFn: async (): Promise<HistoryResult> => {
            // Fetch reward events at current height and 24h ago in parallel
            const [currentEvents, previousEvents] = await Promise.all([
                dsFetch<RewardEvent[]>('events.byAddress', {
                    address,
                    height: 0,
                    page: 1,
                    perPage: 10000 // Large number to get all rewards
                }),
                dsFetch<RewardEvent[]>('events.byAddress', {
                    address,
                    height: (height24hAgo * 2),
                    page: 1,
                    perPage: 10000
                })
            ]);




            const currentTotal = currentEvents
                .filter(event => event.eventType === 'reward' &&  event.height > height24hAgo && event.height <= (height24hAgo * 2) )
                .reduce((sum, event) => sum + (event.msg?.amount || 0), 0);
            const previousTotal = previousEvents
            .filter(event => event.eventType === 'reward' &&  event.height > (height24hAgo * 2) && event.height <= height24hAgo)
            .reduce((sum, event) => sum + (event.msg?.amount || 0), 0);

            return calculateHistory(currentTotal, previousTotal);
        }
    });
}
