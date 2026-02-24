import React from 'react'

interface SearchFiltersProps {
    filters: {
        type: string
        date: string
        sort: string
    }
    onFilterChange: (filters: any) => void
}

const SearchFilters: React.FC<SearchFiltersProps> = ({ filters, onFilterChange }) => {
    const typeOptions = [
        { value: 'all', label: 'All Types' },
        { value: 'blocks', label: 'Blocks' },
        { value: 'transactions', label: 'Transactions' },
        { value: 'addresses', label: 'Addresses' },
        { value: 'validators', label: 'Validators' }
    ]

    const dateOptions = [
        { value: 'all', label: 'All Time' },
        { value: '24h', label: 'Last 24 Hours' },
        { value: '7d', label: 'Last 7 Days' },
        { value: '30d', label: 'Last 30 Days' },
        { value: '1y', label: 'Last Year' }
    ]

    const sortOptions = [
        { value: 'newest', label: 'Newest First' },
        { value: 'oldest', label: 'Oldest First' },
        { value: 'relevance', label: 'Most Relevant' }
    ]

    const handleFilterChange = (key: string, value: string) => {
        onFilterChange({
            ...filters,
            [key]: value
        })
    }

    return (
        <div className="flex flex-wrap gap-4 items-center">
            {/* Type Filter */}
            <div className="flex items-center gap-2">
                <span className="text-sm text-gray-400">Type:</span>
                <select
                    value={filters.type}
                    onChange={(e) => handleFilterChange('type', e.target.value)}
                    className="bg-input border border-gray-800/50 rounded-md px-3 py-1 text-white text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
                >
                    {typeOptions.map(option => (
                        <option key={option.value} value={option.value}>
                            {option.label}
                        </option>
                    ))}
                </select>
            </div>

            {/* Date Filter */}
            <div className="flex items-center gap-2">
                <span className="text-sm text-gray-400">Date:</span>
                <select
                    value={filters.date}
                    onChange={(e) => handleFilterChange('date', e.target.value)}
                    className="bg-input border border-gray-800/50 rounded-md px-3 py-1 text-white text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
                >
                    {dateOptions.map(option => (
                        <option key={option.value} value={option.value}>
                            {option.label}
                        </option>
                    ))}
                </select>
            </div>

            {/* Sort Filter */}
            <div className="flex items-center gap-2">
                <span className="text-sm text-gray-400">Sort:</span>
                <select
                    value={filters.sort}
                    onChange={(e) => handleFilterChange('sort', e.target.value)}
                    className="bg-input border border-gray-800/50 rounded-md px-3 py-1 text-white text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
                >
                    {sortOptions.map(option => (
                        <option key={option.value} value={option.value}>
                            {option.label}
                        </option>
                    ))}
                </select>
            </div>
        </div>
    )
}

export default SearchFilters
