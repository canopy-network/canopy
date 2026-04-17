import { useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useDSFetcher } from '@/core/dsFetch'
import { useHistoryCalculation } from './useHistoryCalculation'
import {useAccounts} from "@/app/providers/AccountsProvider";

export interface ChartDataPoint {
    timestamp: number;
    value: number;
    label: string;
}

interface BalanceChartOptions {
    points?: number; // Number of data points (default: 7 for last 7 days)
    type?: 'balance' | 'staked'; // Type of data to fetch
}

export function useBalanceChart({ points = 7, type = 'balance' }: BalanceChartOptions = {}) {
    const { accounts, loading: accountsLoading } = useAccounts()
    const addresses = accounts.map(a => a.address).filter(Boolean)
    const dsFetch = useDSFetcher()
    const { currentHeight, secondsPerBlock, isReady } = useHistoryCalculation()
    const lastGoodDataRef = useRef<ChartDataPoint[]>([])

    return useQuery({
        queryKey: ['balanceChart', type, addresses, currentHeight, points],
        enabled: !accountsLoading && addresses.length > 0 && isReady,
        staleTime: 60_000, // 1 minute
        retry: 1,
        // Keep previous data visible while refetching — prevents skeleton flash
        // every time currentHeight changes (every ~10 s).
        placeholderData: (prev) => prev,

        queryFn: async (): Promise<ChartDataPoint[]> => {
            if (addresses.length === 0 || currentHeight === 0 || secondsPerBlock == null) {
                return []
            }

            const blocksPerHour = Math.round((60 * 60) / secondsPerBlock)


            const hoursInterval = 24 / (points - 1)

            const heights: number[] = []
            for (let i = 0; i < points; i++) {
                const hoursAgo = Math.round(hoursInterval * (points - 1 - i))
                const heightOffset = Math.round(blocksPerHour * hoursAgo)
                const height = Math.max(0, currentHeight - heightOffset)
                heights.push(height)
            }

            // Get data for each height
            const dataPoints: ChartDataPoint[] = []

            for (let i = 0; i < heights.length; i++) {
                const height = heights[i]
                const hoursAgo = Math.round(hoursInterval * (points - 1 - i))

                try {
                    let totalValue = 0

                    if (type === 'balance') {
                        const [balances, stakes] = await Promise.all([
                            Promise.all(
                                addresses.map(address =>
                                    dsFetch<number>('accountByHeight', { address, height })
                                        .then(v => v || 0)
                                        .catch(() => 0)
                                )
                            ),
                            Promise.all(
                                addresses.map(address =>
                                    dsFetch<Record<string, unknown>>('validatorByHeight', { address, height })
                                        .then(v => (v as Record<string, unknown>)?.stakedAmount as number || 0)
                                        .catch(() => 0)
                                )
                            ),
                        ])
                        const totalAccount = balances.reduce((sum, v) => sum + v, 0)
                        const totalStaked = stakes.reduce((sum, v) => sum + v, 0)
                        totalValue = Math.max(0, totalAccount - totalStaked)
                    } else if (type === 'staked') {
                        // Get staked amounts of all addresses at this height
                        const stakes = await Promise.all(
                            addresses.map(address =>
                                dsFetch<any>('validatorByHeight', { address, height })
                                    .then(v => v?.stakedAmount || 0)
                                    .catch(() => 0)
                            )
                        )
                        totalValue = stakes.reduce((sum, v) => sum + v, 0)
                    }

                    // Create appropriate label for hours
                    let label = ''
                    if (hoursAgo === 0) {
                        label = 'Now'
                    } else if (hoursAgo === 1) {
                        label = '1h ago'
                    } else if (hoursAgo < 24) {
                        label = `${hoursAgo}h ago`
                    } else {
                        label = '24h ago'
                    }

                    dataPoints.push({
                        timestamp: height,
                        value: totalValue,
                        label
                    })
                } catch (error) {
                    console.warn(`Error fetching data for height ${height}:`, error)
                    const errorLabel = hoursAgo === 0 ? 'Now' : hoursAgo === 24 ? '24h ago' : `${hoursAgo}h ago`
                    dataPoints.push({
                        timestamp: height,
                        value: 0,
                        label: errorLabel
                    })
                }
            }

            const hasNonZero = dataPoints.some(p => p.value > 0)
            if (hasNonZero) {
                lastGoodDataRef.current = dataPoints
                return dataPoints
            }

            if (lastGoodDataRef.current.length > 0) {
                return lastGoodDataRef.current
            }

            return dataPoints
        }
    })
}
