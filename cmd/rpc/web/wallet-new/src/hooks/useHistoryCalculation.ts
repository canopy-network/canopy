import { useDS } from "@/core/useDs"
import { useConfig } from '@/app/providers/ConfigProvider'

export interface HistoryResult {
    current: number;
    previous24h: number;
    change24h: number;
    changePercentage: number;
    progressPercentage: number;
}

/**
 * Hook to get consistent block height calculations for 24h history
 * This ensures all charts and difference calculations use the same logic
 */
export function useHistoryCalculation() {
    const { chain } = useConfig()
    const { data: currentHeight = 0 } = useDS<number>('height', {}, { staleTimeMs: 15_000 })

    // Calculate height 24h ago using consistent logic
    const secondsPerBlock = Number(chain?.params?.avgBlockTimeSec) > 0
        ? Number(chain?.params?.avgBlockTimeSec)
        : 20 // Default to 20 seconds if not available

    const blocksPerDay = Math.round((24 * 60 * 60) / secondsPerBlock)
    const height24hAgo = Math.max(0, currentHeight - blocksPerDay)

    /**
     * Calculate history metrics from current and previous values
     */
    const calculateHistory = (currentTotal: number, previousTotal: number): HistoryResult => {
        const change24h = currentTotal - previousTotal
        const changePercentage = previousTotal > 0 ? (change24h / previousTotal) * 100 : 0
        const progressPercentage = Math.min(Math.abs(changePercentage), 100)

        return {
            current: currentTotal,
            previous24h: previousTotal,
            change24h,
            changePercentage,
            progressPercentage
        }
    }

    return {
        currentHeight,
        height24hAgo,
        blocksPerDay,
        secondsPerBlock,
        calculateHistory,
        isReady: currentHeight > 0
    }
}
