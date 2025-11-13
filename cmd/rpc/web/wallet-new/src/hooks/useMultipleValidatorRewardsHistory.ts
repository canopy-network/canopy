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
 * Hook to calculate rewards for multiple validators using block height comparison
 * Fetches reward events at current height and 24h ago to calculate the difference
 */
export function useMultipleValidatorRewardsHistory(addresses: string[]) {
    const dsFetch = useDSFetcher();
    const { currentHeight, height24hAgo, calculateHistory, isReady } = useHistoryCalculation();


    return useQuery({
        queryKey: ['multipleValidatorRewardsHistory', addresses, currentHeight],
        enabled: addresses.length > 0 && isReady,
        staleTime: 30_000,

        queryFn: async (): Promise<Record<string, HistoryResult>> => {
            const results: Record<string, HistoryResult> = {};

            // Fetch rewards for all validators in parallel
            const validatorPromises = addresses.map(async (address) => {
                try {
                    // Fetch reward events at current height and 24h ago in parallel
                    const [currentEvents, previousEvents] = await Promise.all([
                        dsFetch<EventsResponse>('events.byAddress', {
                            address,
                            height: currentHeight,
                            page: 1,
                            perPage: 10000 // Large number to get all rewards
                        }),
                        dsFetch<EventsResponse>('events.byAddress', {
                            address,
                            height: height24hAgo,
                            page: 1,
                            perPage: 10000
                        })
                    ]);

                    // Calculate total rewards from events
                    const calculateTotalRewards = (response: EventsResponse | null): number => {
                        if (!response || !response.results) return 0;

                        return response.results
                            .filter(event => event.eventType === 'reward')
                            .reduce((sum, event) => sum + (event.msg?.amount || 0), 0);
                    };

                    const currentTotal = calculateTotalRewards(currentEvents);
                    const previousTotal = calculateTotalRewards(previousEvents);

                    results[address] = calculateHistory(currentTotal, previousTotal);
                } catch (error) {
                    console.error(`Error fetching rewards for ${address}:`, error);
                    results[address] = {
                        current: 0,
                        previous24h: 0,
                        change24h: 0,
                        changePercentage: 0,
                        progressPercentage: 0
                    };
                }
            });

            await Promise.all(validatorPromises);

            return results;
        }
    });
}
