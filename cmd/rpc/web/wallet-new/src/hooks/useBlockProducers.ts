import { useDS } from '@/core/useDs';
import { useMemo } from 'react';

interface BlockProposer {
    address: string;
    height: number;
}

interface BlockProducerStats {
    blocksProduced: number;
    totalBlocksQueried: number;
    productionRate: number; // percentage
    lastBlockHeight: number;
}

export const useBlockProducers = (count: number = 1000) => {
    const { data: proposers = [], isLoading, error } = useDS<BlockProposer[]>(
        'lastProposers',
        { count },
        {
            enabled: true,
            select: (data: any) => {
                // The API returns an array of proposers
                if (Array.isArray(data)) {
                    return data;
                }
                // If it returns an object with a results array
                if (data && Array.isArray(data.results)) {
                    return data.results;
                }
                // If it returns an object with proposers directly
                if (data && typeof data === 'object') {
                    return Object.values(data).filter((item: any) =>
                        item && typeof item === 'object' && 'address' in item
                    );
                }
                return [];
            }
        }
    );

    const getStatsForValidator = useMemo(() => {
        return (validatorAddress: string): BlockProducerStats => {
            if (!proposers || proposers.length === 0) {
                return {
                    blocksProduced: 0,
                    totalBlocksQueried: 0,
                    productionRate: 0,
                    lastBlockHeight: 0,
                };
            }

            const validatorBlocks = proposers.filter(
                (proposer) => proposer.address?.toLowerCase() === validatorAddress?.toLowerCase()
            );

            const blocksProduced = validatorBlocks.length;
            const totalBlocksQueried = proposers.length;
            const productionRate = totalBlocksQueried > 0
                ? (blocksProduced / totalBlocksQueried) * 100
                : 0;

            const lastBlock = validatorBlocks.length > 0
                ? Math.max(...validatorBlocks.map(b => b.height || 0))
                : 0;

            return {
                blocksProduced,
                totalBlocksQueried,
                productionRate,
                lastBlockHeight: lastBlock,
            };
        };
    }, [proposers]);

    return {
        proposers,
        getStatsForValidator,
        isLoading,
        error,
    };
};

// Hook to get stats for multiple validators at once
export const useMultipleValidatorBlockStats = (addresses: string[], count: number = 1000) => {
    const { proposers, getStatsForValidator, isLoading, error } = useBlockProducers(count);

    const stats = useMemo(() => {
        const result: Record<string, BlockProducerStats> = {};
        addresses.forEach(address => {
            result[address] = getStatsForValidator(address);
        });
        return result;
    }, [addresses, getStatsForValidator]);

    return {
        stats,
        isLoading,
        error,
    };
};
