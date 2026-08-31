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
    type?: 'balance' | 'liquid' | 'staked'; // Type of data to fetch
}

function formatAgoLabel(secondsAgo: number): string {
    if (secondsAgo <= 0) return 'Now'

    const minutesAgo = secondsAgo / 60
    const hoursAgo = secondsAgo / 3600

    if (minutesAgo < 60) {
        return `${Math.max(1, Math.round(minutesAgo))}m ago`
    }

    if (hoursAgo < 24) {
        const roundedHours = hoursAgo < 10 ? Math.round(hoursAgo * 10) / 10 : Math.round(hoursAgo)
        return `${roundedHours}h ago`
    }

    return '24h ago'
}

export function useBalanceChart({ points = 7, type = 'balance' }: BalanceChartOptions = {}) {
    const { accounts, loading: accountsLoading } = useAccounts()
    const addresses = accounts.map(a => a.address).filter(Boolean)
    const dsFetch = useDSFetcher()
    const { currentHeight, historyStartHeight, secondsPerBlock, isReady } = useHistoryCalculation()
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
            if (addresses.length === 0 || currentHeight === 0 || secondsPerBlock == null || historyStartHeight == null) {
                return []
            }

            const spanBlocks = Math.max(0, currentHeight - historyStartHeight)
            const pointCount = Math.max(1, Math.min(points, spanBlocks + 1))
            const heights = new Set<number>()

            for (let i = 0; i < pointCount; i++) {
                const ratio = pointCount === 1 ? 1 : i / (pointCount - 1)
                const height = Math.round(historyStartHeight + (spanBlocks * ratio))
                heights.add(Math.max(1, Math.min(currentHeight, height)))
            }

            const sortedHeights = Array.from(heights).sort((a, b) => a - b)

            // Each point carries an `ok` flag so we can distinguish a genuine zero
            // balance from a failed/partial fetch. Failed points are back-filled
            // with the previous good value so the chart never shows a false drop.
            const rawPoints = await Promise.all(
                sortedHeights.map(async (height): Promise<ChartDataPoint & { ok: boolean }> => {
                    const secondsAgo = Math.max(0, (currentHeight - height) * secondsPerBlock)
                    const label = formatAgoLabel(secondsAgo)

                    try {
                        let totalValue = 0

                        // Note: individual fetches are intentionally NOT caught here so a
                        // rejection propagates and marks the whole point as failed (ok=false),
                        // rather than silently contributing 0 and skewing the total.
                        if (type === 'balance' || type === 'liquid') {
                            const [balances, stakes] = await Promise.all([
                                Promise.all(
                                    addresses.map(address =>
                                        dsFetch<number>('accountByHeight', { address, height })
                                            .then(v => v || 0)
                                    )
                                ),
                                Promise.all(
                                    addresses.map(address =>
                                        dsFetch<Record<string, unknown>>('validatorByHeight', { address, height })
                                            .then(v => Number((v as Record<string, unknown>)?.stakedAmount ?? 0) || 0)
                                    )
                                ),
                            ])
                            const totalAccount = balances.reduce((sum, v) => sum + v, 0)
                            const totalStaked = stakes.reduce((sum, v) => sum + v, 0)
                            totalValue = type === 'liquid' ? totalAccount : totalAccount + totalStaked
                        } else if (type === 'staked') {
                            const stakes = await Promise.all(
                                addresses.map(address =>
                                    dsFetch<Record<string, unknown>>('validatorByHeight', { address, height })
                                        .then(v => Number((v as Record<string, unknown>)?.stakedAmount ?? 0) || 0)
                                )
                            )
                            totalValue = stakes.reduce((sum, v) => sum + v, 0)
                        }

                        return { timestamp: height, value: totalValue, label, ok: true }
                    } catch (error) {
                        console.warn(`Error fetching data for height ${height}:`, error)
                        return { timestamp: height, value: 0, label, ok: false }
                    }
                })
            )

            const hasGoodPoint = rawPoints.some(p => p.ok)

            // Every point failed: keep showing the last good series instead of a flat-zero graph.
            if (!hasGoodPoint && lastGoodDataRef.current.length > 0) {
                return lastGoodDataRef.current
            }

            const filled = rawPoints.map(p => ({ ...p, resolved: p.ok }))

            // Carry the previous good value forward across failed points (oldest -> newest).
            let prevGoodValue: number | null = null
            for (const point of filled) {
                if (point.ok) {
                    prevGoodValue = point.value
                } else if (prevGoodValue != null) {
                    point.value = prevGoodValue
                    point.resolved = true
                }
            }

            // Back-fill any remaining (leading) failed points with the next good value (newest -> oldest).
            let nextGoodValue: number | null = null
            for (let i = filled.length - 1; i >= 0; i--) {
                const point = filled[i]
                if (point.ok) {
                    nextGoodValue = point.value
                } else if (!point.resolved && nextGoodValue != null) {
                    point.value = nextGoodValue
                    point.resolved = true
                }
            }

            const dataPoints: ChartDataPoint[] = filled.map(({ timestamp, value, label }) => ({ timestamp, value, label }))

            if (hasGoodPoint) {
                lastGoodDataRef.current = dataPoints
            }

            return dataPoints
        }
    })
}
