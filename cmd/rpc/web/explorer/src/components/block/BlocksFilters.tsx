import React from 'react'
import blocksTexts from '../../data/blocks.json'

interface DynamicFilter {
    key: string
    label: string
}

interface BlocksFiltersProps {
    activeFilter: string
    onFilterChange: (filter: string) => void
    totalBlocks: number
    dynamicFilters: DynamicFilter[]
}

const BlocksFilters: React.FC<BlocksFiltersProps> = ({
    activeFilter,
    onFilterChange,
    totalBlocks,
    dynamicFilters
}) => {
    const filters = dynamicFilters

    return (
        <div className="mb-6">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h1 className="text-3xl font-bold text-white">
                        {blocksTexts.page.title}
                    </h1>
                    <p className="text-gray-400">
                        {blocksTexts.page.description}
                    </p>
                </div>

                <div className="flex items-end justify-center gap-4">
                    <div className="text-sm text-gray-400">
                        {blocksTexts.page.totalBlocks} {totalBlocks.toLocaleString()} {blocksTexts.page.blocksUnit}
                    </div>
                </div>
            </div>

            {/* Filters */}
            <div className="flex items-center justify-between bg-card rounded-lg p-4">
                <div className="flex items-center gap-1">
                    {filters.map((filter) => (
                        <button
                            key={filter.key}
                            onClick={() => onFilterChange(filter.key)}
                            className={`px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 ${
                                activeFilter === filter.key
                                    ? 'bg-primary text-black'
                                    : 'bg-gray-700/50 text-gray-300 hover:bg-white/8'
                            }`}
                        >
                            {filter.label}
                        </button>
                    ))}
                    {activeFilter !== 'all' && (
                        <span className="ml-3 text-xs text-gray-400 italic">
                            Filtered by time from the last 100 cached blocks
                        </span>
                    )}
                </div>
            </div>
        </div>
    )
}

export default BlocksFilters
