import { useDS } from "@/core/useDs"
import { useConfig } from '@/app/providers/ConfigProvider'
import { useBlockTime } from '@/hooks/useBlockTime'

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
    const { blockTimeSec } = useBlockTime(chain)
    const { data: currentHeightRaw } = useDS<unknown>('height', {}, { staleTimeMs: 30_000 })

    // DS `height` can come as number or object ({ height: number }).
    const currentHeight =
        typeof currentHeightRaw === "number"
            ? currentHeightRaw
            : Number((currentHeightRaw as Record<string, unknown>)?.height ?? 0)

    const secondsPerBlock = blockTimeSec

    const blocksPerDay = secondsPerBlock != null
        ? Math.round((24 * 60 * 60) / secondsPerBlock)
        : null
    const height24hAgo = blocksPerDay != null
        ? Math.max(0, currentHeight - blocksPerDay)
        : null

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
        isReady: currentHeight > 0 && secondsPerBlock != null
    }
}
