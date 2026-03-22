import { useQuery } from "@tanstack/react-query";
import { useDSFetcher } from "@/core/dsFetch";

interface BlockProducerData {
  blocksProduced: number;
  rewards24h: number;
  lastProposedHeight?: number;
}

interface UseBlockProducerDataProps {
  validatorAddress: string;
  enabled?: boolean;
}

export function useBlockProducerData({
  validatorAddress,
  enabled = true,
}: UseBlockProducerDataProps) {
  const dsFetch = useDSFetcher();

  return useQuery({
    queryKey: ["blockProducerData", validatorAddress],
    queryFn: async (): Promise<BlockProducerData> => {
      try {
        // Get current height using DS pattern
        const currentHeight = await dsFetch("height");

        // Get last proposers (this gives us recent block proposers)
        const lastProposersResponse = await dsFetch("lastProposers", {
          height: 0,
          count: 100,
        });
        const proposers = lastProposersResponse.addresses || [];

        // Count how many times this validator has proposed blocks recently
        const blocksProduced = proposers.filter(
          (addr: string) => addr === validatorAddress,
        ).length;

        // Get parameters for accurate reward calculation
        const params = await dsFetch("params");
        const mintPerBlock = params.MintPerBlock || 80000000; // 80 CNPY per block
        const proposerCut = params.ProposerCut || 70; // 70% goes to proposer

        // Calculate rewards per block for this validator
        // Proposer gets a percentage of the mint per block
        const rewardsPerBlock = (mintPerBlock * proposerCut) / 100 / 1000000; // Convert to CNPY
        const rewards24h = blocksProduced * rewardsPerBlock;

        // Find the last height this validator proposed
        const lastProposedHeight =
          proposers.lastIndexOf(validatorAddress) >= 0
            ? currentHeight - proposers.lastIndexOf(validatorAddress)
            : undefined;

        return {
          blocksProduced,
          rewards24h,
          lastProposedHeight,
        };
      } catch (error) {
        console.error("Error fetching block producer data:", error);
        return {
          blocksProduced: 0,
          rewards24h: 0,
        };
      }
    },
    enabled: enabled && !!validatorAddress,
    refetchInterval: 30000, // Refetch every 30 seconds
    staleTime: 15000, // Consider data stale after 15 seconds
  });
}

// Hook for multiple validators
export function useMultipleBlockProducerData(validatorAddresses: string[]) {
  const dsFetch = useDSFetcher();

  return useQuery({
    queryKey: ["multipleBlockProducerData", validatorAddresses],
    queryFn: async (): Promise<Record<string, BlockProducerData>> => {
      try {
        const currentHeight = await dsFetch("height");
        const lastProposersResponse = await dsFetch("lastProposers", {
          height: 0,
          count: 100,
        });
        const proposers = lastProposersResponse.addresses || [];

        const results: Record<string, BlockProducerData> = {};

        // Get parameters for accurate reward calculation
        const params = await dsFetch("params");
        const mintPerBlock = params.MintPerBlock || 80000000; // 80 CNPY per block
        const proposerCut = params.ProposerCut || 70; // 70% goes to proposer

        for (const address of validatorAddresses) {
          const blocksProduced = proposers.filter(
            (addr: string) => addr === address,
          ).length;
          const rewardsPerBlock = (mintPerBlock * proposerCut) / 100 / 1000000; // Convert to CNPY
          const rewards24h = blocksProduced * rewardsPerBlock;

          const lastProposedHeight =
            proposers.lastIndexOf(address) >= 0
              ? currentHeight - proposers.lastIndexOf(address)
              : undefined;

          results[address] = {
            blocksProduced,
            rewards24h,
            lastProposedHeight,
          };
        }

        return results;
      } catch (error) {
        console.error("Error fetching multiple block producer data:", error);
        return {};
      }
    },
    enabled: validatorAddresses.length > 0,
    refetchInterval: 30000,
    staleTime: 15000,
  });
}
