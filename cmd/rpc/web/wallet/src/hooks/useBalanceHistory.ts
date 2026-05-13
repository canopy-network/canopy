import { useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useDSFetcher } from '@/core/dsFetch'
import { useHistoryCalculation, HistoryResult } from './useHistoryCalculation'
import { useAccountsList } from "@/app/providers/AccountsProvider";

interface BalanceHistoryOptions {
    type?: 'balance' | 'liquid'
}

export function useBalanceHistory({ type = 'balance' }: BalanceHistoryOptions = {}) {
    const { accounts, loading: accountsLoading } = useAccountsList()
    const addresses = accounts.map(a => a.address).filter(Boolean)
    const dsFetch = useDSFetcher()
    const { currentHeight, historyStartHeight, calculateHistory, isReady } = useHistoryCalculation()
    const lastGoodResultRef = useRef<HistoryResult | null>(null)

    return useQuery({
        queryKey: ['balanceHistory', type, addresses, currentHeight],
        enabled: !accountsLoading && addresses.length > 0 && isReady,
        staleTime: 30_000,
        retry: 2,
        retryDelay: 2000,

        queryFn: async (): Promise<HistoryResult> => {
            if (addresses.length === 0) {
                return { current: 0, previous24h: 0, change24h: 0, changePercentage: 0, progressPercentage: 0, periodLabel: 'Live' }
            }

            const fetchBalance = async (address: string, height: number): Promise<number> => {
                try {
                    const result = await dsFetch<number>('accountByHeight', { address, height })
                    return typeof result === 'number' && Number.isFinite(result) ? result : 0
                } catch {
                    return 0
                }
            }

            const fetchStake = async (address: string, height: number): Promise<number> => {
                try {
                    const result = await dsFetch<Record<string, unknown>>('validatorByHeight', { address, height })
                    return Number((result as Record<string, unknown>)?.stakedAmount ?? 0) || 0
                } catch {
                    return 0
                }
            }

            if (historyStartHeight == null) {
                return { current: 0, previous24h: 0, change24h: 0, changePercentage: 0, progressPercentage: 0, periodLabel: 'Live' }
            }

            const [currentBalances, previousBalances, currentStakes, previousStakes] = await Promise.all([
                Promise.all(addresses.map(addr => fetchBalance(addr, currentHeight))),
                Promise.all(addresses.map(addr => fetchBalance(addr, historyStartHeight))),
                Promise.all(addresses.map(addr => fetchStake(addr, currentHeight))),
                Promise.all(addresses.map(addr => fetchStake(addr, historyStartHeight))),
            ])

            const currentTotal =
                currentBalances.reduce((sum, v) => sum + v, 0) +
                (type === 'liquid' ? 0 : currentStakes.reduce((sum, v) => sum + v, 0))
            const previousTotal =
                previousBalances.reduce((sum, v) => sum + v, 0) +
                (type === 'liquid' ? 0 : previousStakes.reduce((sum, v) => sum + v, 0))

            if (currentTotal === 0 && previousTotal === 0) {
                try {
                    const [liveBalances, liveStakes] = await Promise.all([
                        Promise.all(
                            addresses.map(addr =>
                                dsFetch<{ amount?: number }>('account', { account: { address: addr } })
                                    .then(r => (typeof r === 'number' ? r : Number(r?.amount ?? 0)))
                                    .catch(() => 0)
                            )
                        ),
                        Promise.all(addresses.map(addr => fetchStake(addr, currentHeight))),
                    ])
                    const liveTotal =
                        liveBalances.reduce((s, v) => s + (Number.isFinite(v) ? v : 0), 0) +
                        (type === 'liquid' ? 0 : liveStakes.reduce((s, v) => s + (Number.isFinite(v) ? v : 0), 0))
                    if (liveTotal > 0) {
                        const liveResult = calculateHistory(liveTotal, liveTotal)
                        lastGoodResultRef.current = liveResult
                        return liveResult
                    }
                } catch { /* fall through */ }
            }

            const result = calculateHistory(currentTotal, previousTotal)

            if (result.current > 0) {
                lastGoodResultRef.current = result
                return result
            }

            if (lastGoodResultRef.current) {
                return lastGoodResultRef.current
            }

            return result
        }
    })
}
