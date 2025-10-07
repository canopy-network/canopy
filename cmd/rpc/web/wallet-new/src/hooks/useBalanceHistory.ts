import { useQuery } from '@tanstack/react-query'
import { useAccounts } from './useAccounts'
import { useDSFetcher } from '@/core/dsFetch'
import { useConfig } from '@/app/providers/ConfigProvider'
import {useDS} from "@/core/useDs";

interface BalanceHistory {
    current: number;
    previous24h: number;
    change24h: number;
    changePercentage: number;
    progressPercentage: number;
}

export function useBalanceHistory() {
    const { accounts, loading: accountsLoading } = useAccounts()
    const addresses = accounts.map(a => a.address).filter(Boolean)
    const { chain } = useConfig()
    const dsFetch = useDSFetcher()

    // 1) Altura actual (cacheada via DS)
    const { data: currentHeight = 0 } = useDS<number>('height', {}, { staleTimeMs: 15_000 })


    // 2) Query agregada para el histórico (depende de addresses + height)
    return useQuery({
        queryKey: ['balanceHistory', addresses, currentHeight],
        enabled: !accountsLoading && addresses.length > 0 && currentHeight > 0,
        staleTime: 30_000,
        retry: 2,
        retryDelay: 2000,

        queryFn: async (): Promise<BalanceHistory> => {
            if (addresses.length === 0) {
                return { current: 0, previous24h: 0, change24h: 0, changePercentage: 0, progressPercentage: 0 }
            }

            // 2.1 calcular altura hace 24h
            const secondsPerBlock =
                Number(chain?.params?.avgBlockTimeSec) > 0 ? Number(chain?.params?.avgBlockTimeSec) : 120
            const blocksPerDay = Math.round((24 * 60) * 60 / secondsPerBlock)
            const height24hAgo = Math.max(0, currentHeight - blocksPerDay)
            // 2.2 pedir balances actuales y de hace 24h en paralelo usando DS
            const currentPromises = addresses.map(address =>
                dsFetch<number>('accountByHeight', { address: address, height: currentHeight })
            )
            const previousPromises = addresses.map(address =>
                dsFetch<number>('accountByHeight', { address, height: height24hAgo })
            )

            const [currentBalances, previousBalances] = await Promise.all([
                Promise.all(currentPromises),
                Promise.all(previousPromises),
            ])

            console.log('currentBalances', currentBalances)


            const currentTotal  = currentBalances.reduce((sum: any, v: any) => sum + (v || 0), 0)
            const previousTotal = previousBalances.reduce((sum: any, v: any) => sum + (v || 0), 0)
            const change24h = currentTotal - previousTotal
            const changePercentage = previousTotal > 0 ? (change24h / previousTotal) * 100 : 0
            const progressPercentage = Math.min(Math.abs(changePercentage) * 10, 100)

            return { current: currentTotal, previous24h: previousTotal, change24h, changePercentage, progressPercentage }
        }
    })
}
