import { useQuery } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';
import { Account, Validators } from '@/core/api';

interface AccountBalance {
  address: string;
  amount: number;
  nickname?: string;
}

interface StakingData {
  address: string;
  staked: number;
  rewards: number;
  nickname?: string;
}

async function fetchAccountBalance(address: string, nickname?: string): Promise<AccountBalance> {
  try {
    // Use height 0 for account queries (as per original wallet)
    const accountData = await Account(0, address);
    
    return {
      address,
      amount: accountData.amount || 0,
      nickname
    };
  } catch (error) {
    console.error(`Error fetching balance for address ${address}:`, error);
    return {
      address,
      amount: 0,
      nickname
    };
  }
}

async function fetchStakingData(address: string, nickname?: string): Promise<StakingData> {
  try {
    // Get all validators and find if this address is a validator
    const allValidatorsResponse = await Validators(0);
    const allValidators = allValidatorsResponse.results || [];
    const validator = allValidators.find((v: any) => v.address === address);
    
    if (validator) {
      return {
        address,
        staked: validator.stakedAmount || 0,
        rewards: 0, // Rewards would need to be calculated separately
        nickname
      };
    } else {
      // Address is not a validator
      return {
        address,
        staked: 0,
        rewards: 0,
        nickname
      };
    }
  } catch (error) {
    console.error(`Error fetching staking data for address ${address}:`, error);
    return {
      address,
      staked: 0,
      rewards: 0,
      nickname
    };
  }
}

export function useAccountData() {
  const { accounts, loading: accountsLoading } = useAccounts();

  const balanceQuery = useQuery({
    queryKey: ['accountBalances', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async () => {
      if (accounts.length === 0) return { totalBalance: 0, balances: [] };

      const balancePromises = accounts.map(account => 
        fetchAccountBalance(account.address, account.nickname)
      );
      
      const balances = await Promise.all(balancePromises);
      const totalBalance = balances.reduce((sum, balance) => sum + balance.amount, 0);
      
      return { totalBalance, balances };
    },
    staleTime: 10000,
    retry: 2,
    retryDelay: 1000,
  });

  const stakingQuery = useQuery({
    queryKey: ['stakingData', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async () => {
      if (accounts.length === 0) return { totalStaked: 0, stakingData: [] };

      const stakingPromises = accounts.map(account => 
        fetchStakingData(account.address, account.nickname)
      );
      
      const stakingData = await Promise.all(stakingPromises);
      const totalStaked = stakingData.reduce((sum, data) => sum + data.staked, 0);
      
      return { totalStaked, stakingData };
    },
    staleTime: 10000,
    retry: 2,
    retryDelay: 1000,
  });

  return {
    totalBalance: balanceQuery.data?.totalBalance || 0,
    totalStaked: stakingQuery.data?.totalStaked || 0,
    balances: balanceQuery.data?.balances || [],
    stakingData: stakingQuery.data?.stakingData || [],
    loading: balanceQuery.isLoading || stakingQuery.isLoading,
    error: balanceQuery.error || stakingQuery.error,
  };
}
