import React from 'react'
import blocksTexts from '../../data/blocks.json'

interface BlocksFiltersProps {
    activeFilter: string
    onFilterChange: (filter: string) => void
    totalBlocks: number
}

const BlocksFilters: React.FC<BlocksFiltersProps> = ({
    activeFilter,
    onFilterChange,
    totalBlocks
}) => {
    const filters = [
        { key: 'all', label: blocksTexts.filters.allBlocks },
        { key: 'hour', label: blocksTexts.filters.lastHour },
        { key: '24h', label: blocksTexts.filters.last24h },
        { key: 'week', label: blocksTexts.filters.lastWeek }
    ]

    return (
        <div className="mb-6">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div >
                    <h1 className="text-3xl font-bold text-white">
                        {blocksTexts.page.title}
                    </h1>
                    <p className="text-gray-400">
                        {blocksTexts.page.description}
                    </p>
                </div>

                {/* Live Updates and Total */}
                <div className="flex items-end justify-center gap-4">
                    <div className="flex items-center gap-2">
                        <div className="w-2 h-2 bg-primary rounded-full animate-pulse"></div>
                        <div className="flex items-center bg-green-500/10 rounded-full p-2 py-0.5">
                            <span className="text-xs text-primary">
                                {blocksTexts.filters.liveUpdates}
                            </span>
                        </div>
                    </div>
                    <div className="text-sm text-gray-400">
                        {blocksTexts.page.totalBlocks} {totalBlocks.toLocaleString()} {blocksTexts.page.blocksUnit}
                    </div>
                </div>
            </div>

            {/* Filters and Controls */}
            <div className="flex items-center justify-between bg-card rounded-lg p-4">
                {/* Filter Tabs */}
                <div className="flex items-center gap-1">
                    {filters.map((filter) => (
                        <button
                            key={filter.key}
                            onClick={() => onFilterChange(filter.key)}
                            className={`px-4 py-2 rounded-md text-sm font-medium transition-all duration-200 ${activeFilter === filter.key
                                ? 'bg-primary text-black'
                                : 'bg-gray-700/50 text-gray-300 hover:bg-gray-600/50'
                                }`}
                        >
                            {filter.label}
                        </button>
                    ))}
                </div>

                {/* Sort and Filter Controls */}
                <div className="flex items-center gap-3">
                    <div className="relative">
                        <select className="bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary">
                            <option>{blocksTexts.table.controls.sortBy}</option>
                        </select>
                    </div>
                    <button className="flex items-center gap-2 bg-gray-700/50 border border-gray-600 rounded-md px-3 py-2 text-sm text-gray-300 hover:bg-gray-600/50 transition-colors">
                        <i className="fa-solid fa-filter text-xs"></i>
                        {blocksTexts.table.controls.filter}
                    </button>
                </div>
            </div>

        </div>
    )
}

export default BlocksFilters
