import { useQuery } from '@tanstack/react-query'
import { useDSFetcher } from '@/core/dsFetch'
import { useHistoryCalculation, HistoryResult } from './useHistoryCalculation'
import {useAccounts} from "@/app/providers/AccountsProvider";

export function useStakedBalanceHistory() {
    const { accounts, loading: accountsLoading } = useAccounts()
    const addresses = accounts.map(a => a.address).filter(Boolean)
    const dsFetch = useDSFetcher()
    const { currentHeight, height24hAgo, calculateHistory, isReady } = useHistoryCalculation()

    return useQuery({
        queryKey: ['stakedBalanceHistory', addresses, currentHeight],
        enabled: !accountsLoading && addresses.length > 0 && isReady,
        staleTime: 30_000,
        retry: 2,
        retryDelay: 2000,

        queryFn: async (): Promise<HistoryResult> => {
            if (addresses.length === 0) {
                return { current: 0, previous24h: 0, change24h: 0, changePercentage: 0, progressPercentage: 0 }
            }

            // Fetch current and previous staked amounts in parallel
            const currentPromises = addresses.map(address =>
                dsFetch<any>('validatorByHeight', { address, height: currentHeight })
                    .then(v => v?.stakedAmount || 0)
                    .catch(() => 0)
            )
            const previousPromises = addresses.map(address =>
                dsFetch<any>('validatorByHeight', { address, height: height24hAgo })
                    .then(v => v?.stakedAmount || 0)
                    .catch(() => 0)
            )

            const [currentStakes, previousStakes] = await Promise.all([
                Promise.all(currentPromises),
                Promise.all(previousPromises),
            ])

            const currentTotal = currentStakes.reduce((sum, v) => sum + (v || 0), 0)
            const previousTotal = previousStakes.reduce((sum, v) => sum + (v || 0), 0)

            return calculateHistory(currentTotal, previousTotal)
        }
    })
}
