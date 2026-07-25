import { useDS } from "@/core/useDs"
import { useConfig } from '@/app/providers/ConfigProvider'
import { useBlockTime } from '@/hooks/useBlockTime'

export interface HistoryResult {
    current: number;
    previous24h: number;
    change24h: number;
    changePercentage: number;
    progressPercentage: number;
    periodLabel: string;
}

function formatHistoryPeriodLabel(seconds: number | null): string {
    if (seconds == null || seconds <= 0) return 'Live';
    const minutes = seconds / 60;
    const hours = seconds / 3600;

    if (minutes < 60) {
        return `${Math.max(1, Math.round(minutes))}m`;
    }

    if (hours < 24) {
        const roundedHours = hours < 10 ? Math.round(hours * 10) / 10 : Math.round(hours);
        return `${roundedHours}h`;
    }

    return '24h';
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
    const hasFull24hHistory = blocksPerDay != null && currentHeight > blocksPerDay
    const historyStartHeight = currentHeight > 0
        ? (hasFull24hHistory ? (height24hAgo ?? 1) : 1)
        : null
    const historySpanBlocks = historyStartHeight != null
        ? Math.max(0, currentHeight - historyStartHeight)
        : null
    const historySpanSeconds = historySpanBlocks != null && secondsPerBlock != null
        ? historySpanBlocks * secondsPerBlock
        : null
    const periodLabel = hasFull24hHistory ? '24h' : formatHistoryPeriodLabel(historySpanSeconds)

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
            progressPercentage,
            periodLabel,
        }
    }

    return {
        currentHeight,
        height24hAgo,
        historyStartHeight,
        historySpanBlocks,
        historySpanSeconds,
        hasFull24hHistory,
        periodLabel,
        blocksPerDay,
        secondsPerBlock,
        calculateHistory,
        isReady: currentHeight > 0 && secondsPerBlock != null
    }
}
