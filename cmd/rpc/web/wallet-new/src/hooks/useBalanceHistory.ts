import { useQuery } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';
import { Account, Height } from '@/core/api';

interface BalanceHistory {
  current: number;
  previous24h: number;
  change24h: number;
  changePercentage: number;
  progressPercentage: number;
}

const API_BASE_URL = 'http://localhost:50002/v1';

async function fetchAccountBalanceAtHeight(address: string, height: number): Promise<number> {
  try {
    const accountData = await Account(height, address);
    return accountData.amount || 0;
  } catch (error) {
    console.error(`Error fetching balance for address ${address} at height ${height}:`, error);
    return 0;
  }
}

async function getCurrentBlockHeight(): Promise<number> {
  try {
    const heightResponse = await Height();
    return heightResponse.height || 0;
  } catch (error) {
    console.error('Error fetching current block height:', error);
    return 0;
  }
}

export function useBalanceHistory() {
  const { accounts, loading: accountsLoading } = useAccounts();

  return useQuery({
    queryKey: ['balanceHistory', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async (): Promise<BalanceHistory> => {
      if (accounts.length === 0) {
        return {
          current: 0,
          previous24h: 0,
          change24h: 0,
          changePercentage: 0,
          progressPercentage: 0
        };
      }

      try {
        // Obtener altura actual del bloque
        const currentHeight = await getCurrentBlockHeight();
        
        // Estimar altura de hace 24 horas (asumiendo ~1 bloque por segundo)
        const blocksPerDay = 24 * 60 * 60; // 86400 bloques por día
        const height24hAgo = Math.max(0, currentHeight - blocksPerDay);

        // Obtener balances actuales
        const currentBalancePromises = accounts.map(account => 
          fetchAccountBalanceAtHeight(account.address, currentHeight)
        );
        
        // Obtener balances de hace 24 horas
        const previousBalancePromises = accounts.map(account => 
          fetchAccountBalanceAtHeight(account.address, height24hAgo)
        );

        const [currentBalances, previousBalances] = await Promise.all([
          Promise.all(currentBalancePromises),
          Promise.all(previousBalancePromises)
        ]);

        const currentTotal = currentBalances.reduce((sum, balance) => sum + balance, 0);
        const previousTotal = previousBalances.reduce((sum, balance) => sum + balance, 0);

        const change24h = currentTotal - previousTotal;
        const changePercentage = previousTotal > 0 ? (change24h / previousTotal) * 100 : 0;
        
        // Calcular porcentaje de progreso basado en el cambio
        // Si el cambio es positivo, mostrar progreso hacia arriba
        // Si es negativo, mostrar progreso hacia abajo
        const progressPercentage = Math.min(Math.abs(changePercentage) * 10, 100);

        return {
          current: currentTotal,
          previous24h: previousTotal,
          change24h,
          changePercentage,
          progressPercentage
        };
      } catch (error) {
        console.error('Error calculating balance history:', error);
        return {
          current: 0,
          previous24h: 0,
          change24h: 0,
          changePercentage: 0,
          progressPercentage: 0
        };
      }
    },
    staleTime: 30000, // 30 segundos
    retry: 2,
    retryDelay: 2000,
  });
}
