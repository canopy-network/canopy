import { useQuery } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';

interface AccountBalance {
  address: string;
  amount: number;
}

interface AccountsResponse {
  pageNumber: number;
  perPage: number;
  totalPages: number;
  totalElements: number;
  results: AccountBalance[];
}

const API_BASE_URL = 'http://localhost:50002/v1';

async function fetchAccountBalance(address: string): Promise<number> {
  try {
    const response = await fetch(`${API_BASE_URL}/query/account`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        address,
        height: 0 // Latest block
      }),
    });

    if (!response.ok) {
      throw new Error(`Error ${response.status}: ${response.statusText}`);
    }

    const data: AccountBalance = await response.json();
    return data.amount || 0;
  } catch (error) {
    console.error(`Error fetching balance for address ${address}:`, error);
    return 0;
  }
}

export function useTotalStage() {
  const { accounts, loading: accountsLoading } = useAccounts();

  return useQuery({
    queryKey: ['totalStage', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async () => {
      if (accounts.length === 0) return 0;

      try {
        // Fetch balances for all accounts in parallel
        const balancePromises = accounts.map(account => 
          fetchAccountBalance(account.address)
        );
        
        const balances = await Promise.all(balancePromises);
        
        // Sum all balances
        const totalStage = balances.reduce((sum, balance) => sum + balance, 0);
        
        return totalStage;
      } catch (error) {
        console.error('Error calculating total stage:', error);
        return 0;
      }
    },
    // Refetch every 20 seconds (inherits from global config)
    // Cache for 10 seconds to avoid too many requests
    staleTime: 10000,
    retry: 2, // Retry failed requests up to 2 times
    retryDelay: 1000, // Wait 1 second between retries
  });
}
