import React, { useState, useEffect } from 'react'

interface AnalyticsFiltersProps {
    fromBlock: string
    toBlock: string
    onFromBlockChange: (block: string) => void
    onToBlockChange: (block: string) => void
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
    onToBlockChange
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
                <div className="flex flex-col gap-1">
                    <input
                        type="number"
                        className="w-24 px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white text-sm"
                        placeholder={fromBlock || "From"}
                        value=""
                        onChange={(e) => onFromBlockChange(e.target.value)}
                        onFocus={(e) => {
                            if (!e.target.value && fromBlock) {
                                e.target.value = fromBlock;
                            }
                        }}
                        min="0"
                    />
                </div>
                <span className="text-gray-400 text-xs">To</span>
                <div className="flex flex-col gap-1">
                    <input
                        type="number"
                        className="w-24 px-3 py-2 bg-input border border-gray-800/80 rounded-md text-white text-sm"
                        placeholder={toBlock || "To"}
                        value=""
                        onChange={(e) => onToBlockChange(e.target.value)}
                        onFocus={(e) => {
                            if (!e.target.value && toBlock) {
                                e.target.value = toBlock;
                            }
                        }}
                        min="0"
                    />
                </div>
            </div>
        </div>
    )
}

export default AnalyticsFilters
