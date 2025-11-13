import { useQuery } from '@tanstack/react-query';
import { useValidators } from './useValidators';
import { useMultipleValidatorRewardsHistory } from './useMultipleValidatorRewardsHistory';
import { useDS } from '@/core/useDs';
import {useAccounts} from "@/app/providers/AccountsProvider";

interface StakingInfo {
  totalStaked: number;
  totalRewards: number;
  totalRewards24h: number;
  stakingHistory: Array<{
    height: number;
    staked: number;
    rewards: number;
  }>;
  chartData: Array<{
    x: number;
    y: number;
  }>;
}

export function useStakingData() {
  const { accounts, loading: accountsLoading } = useAccounts();
  const { data: validators = [], isLoading: validatorsLoading } = useValidators();
  const { data: currentHeight = 0 } = useDS<number>('height', {}, { staleTimeMs: 30_000 });
  const validatorAddresses = validators.map((v: any) => v.address);
  const { data: rewardsHistory = {}, isLoading: rewardsLoading } = useMultipleValidatorRewardsHistory(validatorAddresses);

  return useQuery({
    queryKey: ['stakingData', accounts.map(acc => acc.address), validatorAddresses, rewardsHistory, currentHeight],
    enabled: !accountsLoading && !validatorsLoading && accounts.length > 0,
    queryFn: async (): Promise<StakingInfo> => {
      if (accounts.length === 0 || validators.length === 0) {
        return { totalStaked: 0, totalRewards: 0, totalRewards24h: 0, stakingHistory: [], chartData: [] };
      }

      const totalStaked = validators.reduce((sum: number, validator: any) => sum + (validator.stakedAmount || 0), 0);
      let totalRewards24h = 0;
      let totalRewards = 0;

      validators.forEach((validator: any) => {
        const rewardData = rewardsHistory[validator.address];
        if (rewardData) {
          totalRewards24h += rewardData.rewards24h || 0;
          totalRewards += rewardData.totalRewards || 0;
        }
      });

      const stakingHistory = [];
      const chartData = [];
      const dataPoints = 7;

      for (let i = 0; i < dataPoints; i++) {
        const dayOffset = dataPoints - i - 1;
        const height = currentHeight - (dayOffset * 4320);
        const estimatedStaked = totalStaked - (totalRewards24h * dayOffset);
        stakingHistory.push({ height, staked: Math.max(0, estimatedStaked), rewards: totalRewards24h * (i + 1) });
        chartData.push({ x: i, y: Math.max(0, estimatedStaked) });
      }

      return { totalStaked, totalRewards, totalRewards24h, stakingHistory, chartData };
    },
    staleTime: 30000,
    retry: 2,
    retryDelay: 2000,
  });
}
