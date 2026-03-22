import React, { useState, useEffect, useMemo } from 'react'

interface AnalyticsFiltersProps {
    fromBlock: string
    toBlock: string
    onFromBlockChange: (block: string) => void
    onToBlockChange: (block: string) => void
    onSearch?: () => void
    isLoading?: boolean
    errorMessage?: string
    blocksData?: any
}

const blockRangeFilters = [
    { key: '10', oldLabel: '10 Blocks', recentLabel: 'Last 1 minute' },
    { key: '25', oldLabel: '25 Blocks', recentLabel: 'Last 5 minutes' },
    { key: '50', oldLabel: '50 Blocks', recentLabel: 'Last 15 minutes' },
    { key: '100', oldLabel: '100 Blocks', recentLabel: 'Last 30 minutes' }
]

const AnalyticsFilters: React.FC<AnalyticsFiltersProps> = ({
    fromBlock,
    toBlock,
    onFromBlockChange,
    onToBlockChange,
    onSearch,
    isLoading = false,
    errorMessage = '',
    blocksData
}) => {
    const [selectedRange, setSelectedRange] = useState<string>('')

    // Determine if blocks are recent (less than 2 months old)
    const areBlocksRecent = useMemo(() => {
        if (!blocksData?.results || !Array.isArray(blocksData.results) || blocksData.results.length === 0) {
            return false
        }

        // Get the most recent block
        const sortedBlocks = [...blocksData.results].sort((a: any, b: any) => {
            const heightA = a.blockHeader?.height || a.height || 0
            const heightB = b.blockHeader?.height || b.height || 0
            return heightB - heightA
        })

        if (sortedBlocks.length === 0) {
            return false
        }

        const mostRecentBlock = sortedBlocks[0]
        const mostRecentTime = mostRecentBlock.blockHeader?.time || mostRecentBlock.time || 0

        if (!mostRecentTime) {
            return false
        }

        // Convert timestamp (may be in microseconds)
        const mostRecentTimeMs = mostRecentTime > 1e12 ? mostRecentTime / 1000 : mostRecentTime
        const now = Date.now()

        // Calculate age of most recent block from now
        const ageOfMostRecentMs = now - mostRecentTimeMs
        const ageOfMostRecentDays = ageOfMostRecentMs / (24 * 60 * 60 * 1000)

        // If blocks are old (2 months or more), return false
        return ageOfMostRecentDays < 60 // 2 months = ~60 days
    }, [blocksData])

    // Detect when custom range is being used
    useEffect(() => {
        if (fromBlock && toBlock) {
            const from = parseInt(fromBlock)
            const to = parseInt(toBlock)
            const range = to - from + 1

            // Check if it matches any predefined range
            const predefinedRanges = ['10', '25', '50', '100']
            const matchingRange = predefinedRanges.find(r => parseInt(r) === range)

            if (matchingRange) {
                setSelectedRange(matchingRange)
            } else {
                setSelectedRange('custom')
            }
        }
    }, [fromBlock, toBlock])

    const handleBlockRangeSelect = (range: string) => {
        setSelectedRange(range)

        if (range === 'custom') return

        const blockCount = parseInt(range)
        const currentToBlock = parseInt(toBlock) || 0
        const newFromBlock = Math.max(0, currentToBlock - blockCount + 1)

        onFromBlockChange(newFromBlock.toString())
    }

    return (
        <div className="flex items-center justify-between flex-col lg:flex-row gap-4 lg:gap-0 space-x-2 mb-8 bg-card border border-gray-800/30 hover:border-gray-800/50 rounded-xl p-4">
            <div className="flex items-center space-x-2">
                {blockRangeFilters.map((filter) => {
                    const isSelected = selectedRange === filter.key
                    const isCustom = filter.key === 'custom'
                    // Use recentLabel if blocks are recent, otherwise use oldLabel
                    const displayText = areBlocksRecent ? filter.recentLabel : filter.oldLabel

                    return (
                        <button
                            key={filter.key}
                            onClick={() => handleBlockRangeSelect(filter.key)}
                            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${isSelected
                                ? 'bg-primary text-black shadow-lg shadow-primary/25'
                                : isCustom
                                    ? 'bg-input text-gray-300 hover:bg-gray-600 hover:text-white'
                                    : 'bg-input text-gray-300 hover:bg-gray-600 hover:text-white'
                                }`}
                        >
                            <div className="flex flex-col items-center">
                                <span>{displayText}</span>
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
