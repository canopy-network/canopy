import React, { useState, useEffect } from 'react'

interface AnalyticsFiltersProps {
    fromBlock: string
    toBlock: string
    onFromBlockChange: (block: string) => void
    onToBlockChange: (block: string) => void
    onSearch?: () => void
    isLoading?: boolean
    errorMessage?: string
}

const blockRangeFilters = [
    { key: '10', label: '10 Blocks' },
    { key: '25', label: '25 Blocks' },
    { key: '50', label: '50 Blocks' },
    { key: '100', label: '100 Blocks' },
    { key: 'custom', label: 'Custom Range' }
]

const AnalyticsFilters: React.FC<AnalyticsFiltersProps> = ({
    fromBlock,
    toBlock,
    onFromBlockChange,
    onToBlockChange,
    onSearch,
    isLoading = false,
    errorMessage = ''
}) => {
    const [selectedRange, setSelectedRange] = useState<string>('')

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
                            {filter.label}
                        </button>
                    )
                })}
            </div>
            <div className="flex items-center gap-2">
                <span className="text-gray-400 text-xs">From</span>
                <div className="flex flex-col gap-1 relative">
                    <input
                        type="text"
                        className="w-24 px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white text-sm"
                        placeholder="From"
                        value={fromBlock}
                        onChange={(e) => onFromBlockChange(e.target.value)}
                        min="0"
                        disabled={isLoading}
                    />
                </div>
                <span className="text-gray-400 text-xs">To</span>
                <div className="flex flex-col gap-1 relative">
                    <input
                        type="text"
                        className="w-24 px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white text-sm"
                        placeholder="To"
                        value={toBlock}
                        onChange={(e) => onToBlockChange(e.target.value)}
                        min="0"
                        disabled={isLoading}
                    />
                </div>

                {/* Sync animation */}
                {isLoading && (
                    <div className="flex items-center ml-2">
                        <div className="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-primary"></div>
                        <span className="ml-2 text-xs text-primary">Syncing...</span>
                    </div>
                )}

                {/* Error message */}
                {errorMessage && (
                    <div className="flex items-center ml-2">
                        <span className="text-xs text-red-500">{errorMessage}</span>
                    </div>
                )}

                {/* Search button */}
                <button
                    onClick={onSearch}
                    disabled={isLoading || !fromBlock || !toBlock || !!errorMessage}
                    className={`ml-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 
                        ${(isLoading || !fromBlock || !toBlock || !!errorMessage)
                            ? 'bg-gray-600 text-gray-400 cursor-not-allowed'
                            : 'bg-primary text-black hover:bg-primary/80'}`}
                >
                    <i className="fas fa-search mr-2"></i>
                    Search
                </button>
            </div>
        </div>
    )
}

export default AnalyticsFilters
