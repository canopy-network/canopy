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
 * Hook to calculate rewards for multiple validators
 * Fetches reward events and calculates total rewards earned in the last 24h
 */
export function useMultipleValidatorRewardsHistory(addresses: string[]) {
    const dsFetch = useDSFetcher();
    const { currentHeight, height24hAgo, isReady } = useHistoryCalculation();


    return useQuery({
        queryKey: ['multipleValidatorRewardsHistory', addresses, currentHeight],
        enabled: addresses.length > 0 && isReady,
        staleTime: 30_000,

        queryFn: async (): Promise<Record<string, HistoryResult & { rewards24h: number; totalRewards: number }>> => {
            const results: Record<string, HistoryResult & { rewards24h: number; totalRewards: number }> = {};

            // Fetch rewards for all validators in parallel
            const validatorPromises = addresses.map(async (address) => {
                try {
                    // Fetch all reward events for this validator
                    const eventsResponse = await dsFetch<RewardEvent[] | EventsResponse>('events.byAddress', {
                        address,
                        height: 0,
                        page: 1,
                        perPage: 10000 // Large number to get all rewards
                    });


                    // Handle both array format and object format
                    let allEvents: RewardEvent[] = [];
                    if (Array.isArray(eventsResponse)) {
                        allEvents = eventsResponse;
                    } else if (eventsResponse?.results) {
                        allEvents = eventsResponse.results;
                    }

                    const rewardEvents = allEvents.filter(event => event.eventType === 'reward');


                    // Calculate total rewards (all time)
                    const totalRewards = rewardEvents.reduce((sum, event) => sum + (event.msg?.amount || 0), 0);

                    // Calculate rewards from the last 24h
                    const rewards24h = rewardEvents
                        .filter(event =>
                            event.height > height24hAgo &&
                            event.height <= currentHeight
                        )
                        .reduce((sum, event) => sum + (event.msg?.amount || 0), 0);

                    results[address] = {
                        current: rewards24h,
                        previous24h: 0,
                        change24h: rewards24h,
                        changePercentage: 0,
                        progressPercentage: 100,
                        rewards24h: rewards24h,
                        totalRewards: totalRewards
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
