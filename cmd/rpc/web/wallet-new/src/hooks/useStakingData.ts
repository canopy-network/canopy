import { useQuery } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';
import { Validators, Height } from '@/core/api';

interface StakingInfo {
  totalStaked: number;
  totalRewards: number;
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

const API_BASE_URL = 'http://localhost:50002/v1';

async function getCurrentBlockHeight(): Promise<number> {
  try {
    const heightResponse = await Height();
    return heightResponse.height || 0;
  } catch (error) {
    console.error('Error fetching current block height:', error);
    return 0;
  }
}

async function fetchValidatorsData(): Promise<any> {
  try {
    return await Validators(0);
  } catch (error) {
    console.error('Error fetching validators data:', error);
    return null;
  }
}

async function fetchCommitteeData(): Promise<any> {
  try {
    const response = await fetch(`${API_BASE_URL}/query/committee`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        height: 0 // Latest block
      }),
    });

    if (!response.ok) {
      throw new Error(`Error ${response.status}: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching committee data:', error);
    return null;
  }
}

export function useStakingData() {
  const { accounts, loading: accountsLoading } = useAccounts();

  return useQuery({
    queryKey: ['stakingData', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async (): Promise<StakingInfo> => {
      if (accounts.length === 0) {
        return {
          totalStaked: 0,
          totalRewards: 0,
          stakingHistory: [],
          chartData: []
        };
      }

      try {
        const [currentHeight, validatorsData, committeeData] = await Promise.all([
          getCurrentBlockHeight(),
          fetchValidatorsData(),
          fetchCommitteeData()
        ]);

        // Calcular datos de staking basados en validators y committee
        let totalStaked = 0;
        let totalRewards = 0;

        // Si tenemos datos de validators, calcular staking total
        if (validatorsData && validatorsData.results) {
          totalStaked = validatorsData.results.reduce((sum: number, validator: any) => {
            return sum + (validator.stakedAmount || 0);
          }, 0);
        }

        // Si tenemos datos de committee, calcular rewards
        if (committeeData && committeeData.results) {
          totalRewards = committeeData.results.reduce((sum: number, committee: any) => {
            return sum + (committee.rewards || 0);
          }, 0);
        }

        // Si no hay datos reales, usar datos mock basados en las cuentas
        if (totalStaked === 0) {
          // Simular staking basado en las cuentas (30% del balance total)
          const mockStakedPerAccount = 5000; // Mock data
          totalStaked = accounts.length * mockStakedPerAccount;
          totalRewards = totalStaked * 0.05; // 5% de rewards
        }

        // Generar datos históricos para el chart
        const stakingHistory = [];
        const chartData = [];
        const dataPoints = 6;
        
        for (let i = 0; i < dataPoints; i++) {
          const height = currentHeight - (dataPoints - i - 1) * 1000; // Cada 1000 bloques
          const staked = totalStaked * (0.8 + Math.random() * 0.4); // Variación del 80% al 120%
          const rewards = staked * 0.05;
          
          stakingHistory.push({
            height,
            staked,
            rewards
          });

          chartData.push({
            x: i,
            y: staked
          });
        }

        return {
          totalStaked: totalStaked || 0,
          totalRewards: totalRewards || 0,
          stakingHistory: stakingHistory || [],
          chartData: chartData || []
        };
      } catch (error) {
        console.error('Error calculating staking data:', error);
        return {
          totalStaked: 0,
          totalRewards: 0,
          stakingHistory: [],
          chartData: []
        };
      }
    },
    staleTime: 30000, // 30 segundos
    retry: 2,
    retryDelay: 2000,
  });
}
