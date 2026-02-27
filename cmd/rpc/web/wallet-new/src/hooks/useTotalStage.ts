import { useQuery } from '@tanstack/react-query';
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useDSFetcher } from '@/core/dsFetch';

interface AccountBalance {
  address: string;
  amount: number;
}

export function useTotalStage() {
  const { accounts, loading: accountsLoading } = useAccounts();
  const dsFetch = useDSFetcher();

  return useQuery({
    queryKey: ['totalStage', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async () => {
      if (accounts.length === 0) return 0;

      const balancePromises = accounts.map(account =>
        dsFetch<AccountBalance>('account', { account: {address: account.address}, height: 0 })
          .then(data => data?.amount || 0)
          .catch(err => { console.error(`Error fetching balance for ${account.address}:`, err); return 0; })
      );

      const balances = await Promise.all(balancePromises);
      return balances.reduce((sum, balance) => sum + balance, 0);
    },
    staleTime: 10000,
    retry: 2,
    retryDelay: 1000,
  });
}
