import { useQuery } from '@tanstack/react-query';
import { useDSFetcher } from "@/core/dsFetch";

interface ValidatorSetMember {
    publicKey: string;
    votingPower: number;
    netAddress: string;
}

interface ValidatorSetResponse {
    validatorSet: ValidatorSetMember[];
}

/**
 * Hook to fetch validator set data for a specific committee using DS pattern
 * @param committeeId - The committee ID to fetch validator set for
 * @param enabled - Whether the query should run
 */
export function useValidatorSet(committeeId: number, enabled: boolean = true) {
    const dsFetch = useDSFetcher();

    return useQuery({
        queryKey: ['validatorSet', committeeId],
        enabled: enabled && committeeId !== undefined,
        staleTime: 30_000,
        queryFn: async (): Promise<ValidatorSetResponse> => {
            return dsFetch<ValidatorSetResponse>('validatorSet', {
                height: 0,
                committeeId: committeeId
            });
        }
    });
}

/**
 * Hook to fetch validator sets for multiple committees using DS pattern
 * @param committeeIds - Array of committee IDs
 */
export function useMultipleValidatorSets(committeeIds: number[]) {
    const dsFetch = useDSFetcher();

    return useQuery({
        queryKey: ['multipleValidatorSets', committeeIds],
        enabled: committeeIds.length > 0,
        staleTime: 30_000,
        queryFn: async (): Promise<Record<number, ValidatorSetResponse>> => {
            const results: Record<number, ValidatorSetResponse> = {};

            // Fetch all validator sets in parallel
            const promises = committeeIds.map(async (committeeId) => {
                try {
                    const data = await dsFetch<ValidatorSetResponse>('validatorSet', {
                        height: 0,
                        committeeId: committeeId
                    });
                    results[committeeId] = data;
                } catch (error) {
                    console.error(`Error fetching validator set for committee ${committeeId}:`, error);
                    results[committeeId] = { validatorSet: [] };
                }
            });

            await Promise.all(promises);
            return results;
        }
    });
}
