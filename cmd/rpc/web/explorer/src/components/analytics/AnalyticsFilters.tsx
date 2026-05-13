import React, { useState, useEffect, useMemo, useRef } from 'react'

interface AnalyticsFiltersProps {
    fromBlock: string
    toBlock: string
    onFromBlockChange: (block: string) => void
    onToBlockChange: (block: string) => void
    onSearch?: () => void
    isLoading?: boolean
    errorMessage?: string
    blocksData?: any
    blockTime?: number // seconds per block
}

// target time ranges in seconds
const timeTargets = [
    { seconds: 60, label: '1 min' },
    { seconds: 300, label: '5 min' },
    { seconds: 900, label: '15 min' },
    { seconds: 1800, label: '30 min' },
]

// snap to a round number so small blockTime fluctuations don't change labels
const snapToRound = (n: number): number => {
    if (n <= 5) return n
    if (n <= 20) return Math.round(n / 5) * 5
    if (n <= 50) return Math.round(n / 10) * 10
    return Math.round(n / 25) * 25
}

const AnalyticsFilters: React.FC<AnalyticsFiltersProps> = ({
    fromBlock,
    toBlock,
    onFromBlockChange,
    onToBlockChange,
    onSearch,
    isLoading = false,
    errorMessage = '',
    blocksData,
    blockTime = 20,
}) => {
    const [selectedRange, setSelectedRange] = useState<string>('')
    // lock filters once computed so blockTime jitter doesn't change them
    const lockedFilters = useRef<{ key: string; label: string }[] | null>(null)

    // compute block counts once from block time, then lock them
    const blockRangeFilters = useMemo(() => {
        if (lockedFilters.current) return lockedFilters.current

        const filters = timeTargets.map(target => {
            const raw = Math.max(1, Math.round(target.seconds / blockTime))
            const snapped = snapToRound(raw)
            const capped = Math.min(snapped, 100)
            return {
                key: capped.toString(),
                label: `Last ${capped} blocks (~${target.label})`,
            }
        })
        // deduplicate if multiple targets resolve to the same block count
        .filter((f, i, arr) => arr.findIndex(x => x.key === f.key) === i)

        lockedFilters.current = filters
        return filters
    }, [blockTime])

    // when toBlock changes and a range is selected, auto-update fromBlock to maintain the window
    useEffect(() => {
        if (selectedRange && selectedRange !== 'custom' && toBlock) {
            const blockCount = parseInt(selectedRange)
            const currentToBlock = parseInt(toBlock) || 0
            const expectedFrom = Math.max(0, currentToBlock - blockCount + 1)
            const currentFrom = parseInt(fromBlock) || 0

            if (currentFrom !== expectedFrom) {
                onFromBlockChange(expectedFrom.toString())
            }
        }
    }, [toBlock, selectedRange])

    const handleBlockRangeSelect = (range: string) => {
        setSelectedRange(range)

        if (range === 'custom') return

        const blockCount = parseInt(range)
        const currentToBlock = parseInt(toBlock) || 0
        const newFromBlock = Math.max(0, currentToBlock - blockCount + 1)

        onFromBlockChange(newFromBlock.toString())
    }

    return (
        <div className="flex items-center justify-between flex-col lg:flex-row gap-4 lg:gap-0 space-x-2 mb-8 bg-card border border-white/5 hover:border-white/8 rounded-xl p-4">
            <div className="flex items-center space-x-2">
                {blockRangeFilters.map((filter) => {
                    const isSelected = selectedRange === filter.key

                    return (
                        <button
                            key={filter.key}
                            onClick={() => handleBlockRangeSelect(filter.key)}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${isSelected
                                ? 'bg-primary text-black shadow-lg shadow-primary/25'
                                : 'bg-input text-gray-300 hover:bg-white/10 hover:text-white'
                                }`}
                        >
                            <div className="flex flex-col items-center">
                                <span>{filter.label}</span>
                            </div>
                        </button>
                    )
                })}
            </div>
            {/* Sync animation */}
            {isLoading && (
                <div className="flex items-center">
                    <div className="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-primary"></div>
                    <span className="ml-2 text-xs text-primary">Syncing...</span>
                </div>
            )}

            {/* Error message */}
            {errorMessage && (
                <div className="flex items-center">
                    <span className="text-xs text-red-500">{errorMessage}</span>
                </div>
            )}
        </div>
    )
}

export default AnalyticsFilters
