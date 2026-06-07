import { useQuery } from '@tanstack/react-query';
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useDSFetcher } from '@/core/dsFetch';

interface AccountBalance {
  address: string;
  amount: number;
}

const parseMaybeJson = (v: unknown) =>
    (typeof v === 'string' && /^\s*[{[]/.test(v)) ? JSON.parse(v) : v

export function useTotalStage() {
  const { accounts, loading: accountsLoading } = useAccounts();
  const dsFetch = useDSFetcher();

  return useQuery({
    queryKey: ['totalStage', accounts.map(acc => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async () => {
      if (accounts.length === 0) return 0;

      const [balances, validators] = await Promise.all([
        Promise.all(
          accounts.map(account =>
            dsFetch<AccountBalance>('account', { account: { address: account.address }, height: 0 })
              .then(data => data?.amount || 0)
              .catch(() => 0)
          )
        ),
        dsFetch<unknown[]>('validators', {}).catch(() => [])
      ]);

      const liquidTotal = balances.reduce((sum, balance) => sum + balance, 0);

      const validatorsList = Array.isArray(validators) ? validators : [];
      const addressSet = new Set(accounts.map(a => a.address));
      let stakedTotal = 0;
      for (const v of validatorsList) {
        const obj = parseMaybeJson(v) as Record<string, unknown>;
        const key = String(obj?.address ?? obj?.validatorAddress ?? obj?.operatorAddress ?? '');
        if (key && addressSet.has(key)) {
          stakedTotal += Number(obj?.stakedAmount ?? obj?.stake ?? 0) || 0;
        }
      }

      return liquidTotal + stakedTotal;
    },
    staleTime: 10000,
    retry: 2,
    retryDelay: 1000,
  });
}
